package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/calghar/gas-cli/internal/config"
	"github.com/calghar/gas-cli/internal/ghsetup"
	"github.com/calghar/gas-cli/internal/git"
	"github.com/calghar/gas-cli/internal/profile"
)

type screen int

const (
	screenMain screen = iota
	screenForm
	screenConfirm
	screenMessage
	screenPicker
)

type formMode int

const (
	formAdd formMode = iota
	formEdit
	formPAT
)

type model struct {
	cfg       *config.Config
	configMgr *config.ConfigManager

	screen screen
	width  int
	height int

	profileIdx int
	statusMsg  string
	cwd        string
	hasGitRepo bool
	busy       bool

	formKind   formMode
	formFields []textinput.Model
	formFocus  int

	messageOpen        bool
	messageTitle       string
	messageText        string
	messageDismissedAt time.Time

	confirmName string

	pickerIdx         int
	pendingGhUsername string

	quitting bool
}

func newModel(cfg *config.Config, configMgr *config.ConfigManager) model {
	m := model{
		cfg:       cfg,
		configMgr: configMgr,
		screen:    screenMain,
	}
	m.loadCwd()
	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case repoSetupResult:
		return m.handleRepoResult(msg)

	case tea.KeyMsg:
		if m.messageOpen {
			m.dismissMessage()
			return m, nil
		}

		switch m.screen {
		case screenForm:
			return m.updateFormKey(msg)
		case screenConfirm:
			return m.updateConfirmKey(msg)
		case screenPicker:
			return m.updatePickerKey(msg)
		}

		if m.busy && (msg.String() == "r" || msg.String() == "enter") {
			return m, nil
		}
		if m.shouldIgnoreRepoShortcut(msg.String()) {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			return m, nil
		case "up", "k":
			m.profileIdx = max(0, m.profileIdx-1)
		case "down", "j":
			names := m.cfg.ProfileNames()
			if len(names) > 0 {
				m.profileIdx = min(len(names)-1, m.profileIdx+1)
			}
		case "g":
			m.profileIdx = 0
		case "G":
			names := m.cfg.ProfileNames()
			if len(names) > 0 {
				m.profileIdx = len(names) - 1
			}
		case "a":
			if !m.busy {
				m.openForm(formAdd, nil)
			}
		case "e":
			if p := m.selectedProfile(); p != nil {
				m.openForm(formEdit, p)
			} else {
				m.showMessage("No profile", "Add a profile first (press a).")
			}
		case "p":
			if p := m.selectedProfile(); p != nil {
				m.openForm(formPAT, p)
			} else {
				m.showMessage("No profile", "Add a profile first (press a).")
			}
		case "r":
			return m.startConfigureRepo()
		case "d":
			if p := m.selectedProfile(); p != nil {
				m.openConfirmDelete(p.Name)
			} else {
				m.showMessage("No profile", "Add a profile first (press a).")
			}
		case "enter":
			return m.startConfigureRepo()
		}
		m.loadCwd()
		return m, nil
	}

	// Forward blink to focused form field.
	if m.screen == screenForm && m.formFocus < len(m.formFields) {
		var cmd tea.Cmd
		m.formFields[m.formFocus], cmd = m.formFields[m.formFocus].Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) updateFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenMain
		m.formFields = nil
		return m, nil
	case "ctrl+s":
		return m.saveForm()
	case "enter":
		if m.formKind == formPAT || m.formFocus >= len(m.formFields)-1 {
			return m.saveForm()
		}
		m.formFocus++
		m.syncFormFocus()
		return m, textinput.Blink
	case "up", "shift+tab":
		m.formFocus = max(0, m.formFocus-1)
		m.syncFormFocus()
		return m, textinput.Blink
	case "down", "tab":
		m.formFocus = min(len(m.formFields)-1, m.formFocus+1)
		m.syncFormFocus()
		return m, textinput.Blink
	}

	if m.formFocus < len(m.formFields) {
		var cmd tea.Cmd
		m.formFields[m.formFocus], cmd = m.formFields[m.formFocus].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) updateConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.screen = screenMain
		m.deleteSelectedProfile()
	case "n", "N", "esc":
		m.screen = screenMain
		m.confirmName = ""
	}
	return m, nil
}

func (m model) updatePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	names := m.cfg.ProfileNames()
	switch msg.String() {
	case "esc", "n", "N":
		m.screen = screenMain
		m.pendingGhUsername = ""
		m.setStatus("Repo setup cancelled")
		return m, nil
	case "enter":
		if m.pickerIdx < 0 || m.pickerIdx >= len(names) {
			return m, nil
		}
		profileName := names[m.pickerIdx]
		m.screen = screenMain
		m.pendingGhUsername = ""
		m.selectProfileByName(profileName)
		return m.startRepoSetup(profileName)
	case "up", "k":
		m.pickerIdx = max(0, m.pickerIdx-1)
	case "down", "j":
		if len(names) > 0 {
			m.pickerIdx = min(len(names)-1, m.pickerIdx+1)
		}
	}
	return m, nil
}

func (m model) startConfigureRepo() (tea.Model, tea.Cmd) {
	if len(m.cfg.ProfileNames()) == 0 {
		m.showMessage("No profiles", "Add a profile first (press a).")
		return m, nil
	}
	if !m.hasGitRepo {
		title, text := ghsetup.UserFacingError(git.ErrNotGitRepository)
		m.showMessage(title, text)
		return m, nil
	}
	return m.startRepoSetup("")
}

func (m model) startRepoSetup(profileName string) (tea.Model, tea.Cmd) {
	if m.busy {
		return m, nil
	}
	m.busy = true
	if profileName == "" {
		m.statusMsg = "Checking repository..."
	} else {
		m.statusMsg = "Configuring repository..."
	}

	preferred := m.selectedProfileName()
	cfg := m.cfg
	return m, func() tea.Msg {
		return executeRepoSetup(cfg, profileName, preferred)
	}
}

func (m model) handleRepoResult(msg repoSetupResult) (tea.Model, tea.Cmd) {
	m.busy = false
	m.statusMsg = ""

	switch msg.outcome {
	case repoOutcomeMessage:
		m.showMessage(msg.title, msg.message)
	case repoOutcomePicker:
		m.openProfilePicker(msg.ghUsername)
	case repoOutcomeConfigured:
		if msg.profileName != m.selectedProfileName() {
			m.selectProfileByName(msg.profileName)
		}
		if err := profile.Save(m.configMgr, m.cfg); err != nil {
			m.showMessage("Error", err.Error())
			return m, nil
		}
		m.setStatus(fmt.Sprintf("Configured %s with profile '%s'", msg.result.RepoName, msg.result.ProfileName))
		m.showMessage("Repo configured",
			fmt.Sprintf("Repository: %s\nRoot: %s\nProfile: %s\nUser: %s\nEmail: %s\n\nRemote verified successfully.",
				msg.result.RepoName, msg.result.RepoRoot, msg.result.ProfileName, msg.result.UserName, msg.result.Email))
	}
	return m, nil
}

func (m *model) showMessage(title, text string) {
	m.screen = screenMessage
	m.messageOpen = true
	m.messageTitle = title
	m.messageText = text + "\n\n" + lipgloss.NewStyle().Foreground(colorGray).Render("Press any key to continue")
}

func (m *model) dismissMessage() {
	m.messageOpen = false
	m.screen = screenMain
	m.messageTitle = ""
	m.messageText = ""
	m.messageDismissedAt = time.Now()
}

func (m *model) shouldIgnoreRepoShortcut(key string) bool {
	if key != "r" && key != "enter" {
		return false
	}
	return time.Since(m.messageDismissedAt) < 300*time.Millisecond
}

func (m *model) openForm(kind formMode, existing *config.Profile) {
	m.screen = screenForm
	m.formKind = kind
	m.formFocus = 0
	m.formFields = buildFormFields(kind, existing)
	m.syncFormFocus()
}

func buildFormFields(kind formMode, existing *config.Profile) []textinput.Model {
	newField := func(label, placeholder, value string) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = 256
		ti.Width = 40
		ti.Prompt = label + ": "
		ti.SetValue(value)
		return ti
	}

	var fields []textinput.Model
	switch kind {
	case formPAT:
		fields = append(fields, newField("PAT", "enter new token", ""))
	case formEdit:
		gitName, email := "", ""
		if existing != nil {
			gitName = existing.GitName
			email = existing.PrimaryEmail
		}
		fields = append(fields,
			newField("Git name", "GitHub username", gitName),
			newField("Email", "commit email", email),
			newField("PAT", "leave blank to keep current PAT", ""),
		)
	default:
		fields = append(fields,
			newField("Name", "profile name (e.g. work)", ""),
			newField("Git name", "GitHub username", ""),
			newField("Email", "commit email", ""),
			newField("PAT", "ghp_... or github_pat_...", ""),
		)
	}
	return fields
}

func (m *model) syncFormFocus() {
	for i := range m.formFields {
		if i == m.formFocus {
			m.formFields[i].Focus()
		} else {
			m.formFields[i].Blur()
		}
	}
}

func (m model) saveForm() (tea.Model, tea.Cmd) {
	var name, gitName, email, pat string

	switch m.formKind {
	case formPAT:
		if len(m.formFields) > 0 {
			pat = strings.TrimSpace(m.formFields[0].Value())
		}
		existing := m.selectedProfile()
		if existing == nil {
			m.showMessage("Error", "No profile selected")
			return m, nil
		}
		name = existing.Name
		gitName = existing.GitName
		email = existing.PrimaryEmail
		if pat == "" {
			pat = existing.PAT
		}
	case formEdit:
		if len(m.formFields) >= 3 {
			gitName = strings.TrimSpace(m.formFields[0].Value())
			email = strings.TrimSpace(m.formFields[1].Value())
			pat = strings.TrimSpace(m.formFields[2].Value())
		}
		existing := m.selectedProfile()
		if existing == nil {
			m.showMessage("Error", "No profile selected")
			return m, nil
		}
		name = existing.Name
		if gitName == "" {
			gitName = existing.GitName
		}
		if email == "" {
			email = existing.PrimaryEmail
		}
		if pat == "" {
			pat = existing.PAT
		}
	default:
		if len(m.formFields) >= 4 {
			name = strings.TrimSpace(m.formFields[0].Value())
			gitName = strings.TrimSpace(m.formFields[1].Value())
			email = strings.TrimSpace(m.formFields[2].Value())
			pat = strings.TrimSpace(m.formFields[3].Value())
		}
	}

	if name == "" {
		m.showMessage("Error", "Profile name is required")
		return m, nil
	}

	prof := &config.Profile{
		Name:         name,
		GitName:      gitName,
		PrimaryEmail: email,
	}
	if email != "" {
		prof.Emails = []string{email}
	}
	if pat != "" {
		prof.PAT = pat
	}

	if err := profile.Upsert(m.cfg, prof); err != nil {
		m.showMessage("Error", err.Error())
		return m, nil
	}
	if err := profile.Save(m.configMgr, m.cfg); err != nil {
		m.showMessage("Error", err.Error())
		return m, nil
	}

	saved, _ := m.cfg.GetProfile(name)
	if saved != nil {
		if err := profile.AfterSave(m.cfg, saved); err != nil {
			m.screen = screenMain
			m.formFields = nil
			m.setStatus(fmt.Sprintf("Saved profile '%s' (warning: %s)", name, err.Error()))
			return m, nil
		}
	}

	m.screen = screenMain
	m.formFields = nil
	m.setStatus(fmt.Sprintf("Saved profile '%s'", name))
	return m, nil
}

func (m *model) openConfirmDelete(name string) {
	m.screen = screenConfirm
	m.confirmName = name
}

func (m *model) deleteSelectedProfile() {
	name := m.selectedProfileName()
	if name == "" {
		return
	}
	if err := m.cfg.RemoveProfile(name); err != nil {
		m.showMessage("Error", err.Error())
		return
	}
	if err := profile.Save(m.configMgr, m.cfg); err != nil {
		m.showMessage("Error", err.Error())
		return
	}
	if err := profile.AfterRemove(name); err != nil {
		m.setStatus(fmt.Sprintf("Deleted profile '%s' (warning: %s)", name, err.Error()))
		return
	}
	m.setStatus(fmt.Sprintf("Deleted profile '%s'", name))
}

func (m *model) openProfilePicker(ghUsername string) {
	m.screen = screenPicker
	m.pendingGhUsername = ghUsername
	m.pickerIdx = 0
}

func (m *model) selectedProfile() *config.Profile {
	names := m.cfg.ProfileNames()
	if len(names) == 0 {
		return nil
	}
	idx := m.profileIdx
	if idx < 0 || idx >= len(names) {
		idx = 0
	}
	p, _ := m.cfg.GetProfile(names[idx])
	return p
}

func (m *model) selectedProfileName() string {
	if p := m.selectedProfile(); p != nil {
		return p.Name
	}
	return ""
}

func (m *model) selectProfileByName(name string) {
	names := m.cfg.ProfileNames()
	for i, n := range names {
		if n == name {
			m.profileIdx = i
			return
		}
	}
}

func (m *model) loadCwd() {
	if wd, err := os.Getwd(); err == nil {
		m.cwd = wd
		m.hasGitRepo = git.HasGitInCwd()
	} else {
		m.cwd = ""
		m.hasGitRepo = false
	}
}

func (m *model) setStatus(msg string) {
	m.statusMsg = msg
}

func patStatus(p *config.Profile) string {
	if p == nil || p.PAT == "" {
		return lipgloss.NewStyle().Foreground(colorRed).Render("no PAT")
	}
	return lipgloss.NewStyle().Foreground(colorGreen).Render("PAT set")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
