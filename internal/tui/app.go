package tui

import (
	"fmt"
	"os"
	"strings"

	ui "github.com/morpheum-labs/mormtui"
	"github.com/morpheum-labs/mormtui/widgets"

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

const (
	modalMessageW = 64
	modalMessageH = 14
	modalConfirmW = 52
	modalConfirmH = 12
	modalFormW    = 64
	modalFormH    = 16
	modalPickerW  = 58
	modalPickerH  = 14
)

type formMode int

const (
	formAdd formMode = iota
	formEdit
	formPAT
)

// Run starts the interactive identity management console.
func Run() error {
	configMgr, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}

	cfg, err := configMgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := ui.Init(); err != nil {
		return fmt.Errorf("failed to initialize terminal UI: %w", err)
	}
	defer ui.Close()

	app := newApp(cfg, configMgr)
	app.layout()
	app.refresh()
	app.render()

	events := ui.PollEvents()
	for e := range events {
		if e.Type != ui.KeyboardEvent && e.Type != ui.ResizeEvent {
			continue
		}
		if e.Type == ui.ResizeEvent {
			app.layout()
			switch app.screen {
			case screenForm:
				app.resizeModal(modalFormW, modalFormH)
			case screenConfirm:
				app.resizeModal(modalConfirmW, modalConfirmH)
			case screenMessage:
				app.resizeModal(modalMessageW, modalMessageH)
			case screenPicker:
				app.resizeModal(modalPickerW, modalPickerH)
			}
			app.render()
			continue
		}

		if app.handleKey(e.ID) {
			return nil
		}
		app.render()
	}
	return nil
}

type app struct {
	cfg       *config.Config
	configMgr *config.ConfigManager

	lm          *ui.LayerManager
	header      *widgets.Paragraph
	profileList *widgets.List
	details     *widgets.Paragraph
	footer      *widgets.Paragraph

	screen   screen
	formKind formMode
	modal    *widgets.Modal
	form     *widgets.InputList
	message  *widgets.Paragraph
	pickerList *widgets.List

	statusMsg         string
	cwd               string
	busy              bool
	pendingGhUsername string
}

func newApp(cfg *config.Config, configMgr *config.ConfigManager) *app {
	a := &app{
		cfg:       cfg,
		configMgr: configMgr,
		lm:        ui.NewLayerManager(),
		header:    widgets.NewParagraph(),
		profileList: widgets.NewList(),
		details:     widgets.NewParagraph(),
		footer:      widgets.NewParagraph(),
	}

	a.header.Title = " gas-cli — GitHub Identity Manager "
	a.header.BorderStyle.Fg = ui.ColorCyan
	a.header.TextStyle.Fg = ui.ColorWhite

	a.profileList.Title = " Profiles "
	a.profileList.TextStyle = ui.NewStyle(ui.ColorWhite)
	a.profileList.SelectedRowStyle = ui.NewStyle(ui.ColorYellow, ui.ColorBlue, ui.ModifierBold)
	a.profileList.WrapText = false

	a.details.Title = " Details "
	a.details.BorderStyle.Fg = ui.ColorCyan
	a.details.TextStyle.Fg = ui.ColorWhite
	a.details.WrapText = true

	a.footer.Border = false
	a.footer.TextStyle.Fg = ui.ColorBlack
	a.footer.TextStyle.Bg = ui.ColorCyan

	_ = a.lm.AddLayer(ui.LayerBase, a.header, a.profileList, a.details, a.footer)
	return a
}

func (a *app) layout() {
	w, h := ui.TerminalDimensions()
	if w < 60 {
		w = 60
	}
	if h < 20 {
		h = 20
	}

	a.header.SetRect(0, 0, w, 3)

	listW := w / 3
	if listW < 28 {
		listW = 28
	}
	if listW > w-20 {
		listW = w / 2
	}

	a.profileList.SetRect(0, 3, listW, h-3)
	a.details.SetRect(listW, 3, w, h-3)
	a.footer.SetRect(0, h-3, w, h)
}

func (a *app) handleKey(key string) (quit bool) {
	switch a.screen {
	case screenForm:
		return a.handleFormKey(key)
	case screenConfirm:
		return a.handleConfirmKey(key)
	case screenMessage:
		return a.handleMessageKey(key)
	case screenPicker:
		return a.handlePickerKey(key)
	}

	if a.busy && (key == "r" || key == "<Enter>") {
		return false
	}

	switch key {
	case "q", "<C-c>", "<Escape>":
		return true
	case "j", "<Down>":
		a.profileList.ScrollDown()
	case "k", "<Up>":
		a.profileList.ScrollUp()
	case "g", "<Home>":
		a.profileList.ScrollTop()
	case "G", "<End>":
		a.profileList.ScrollBottom()
	case "a":
		if !a.busy {
			a.openForm(formAdd, nil)
		}
	case "e":
		if p := a.selectedProfile(); p != nil {
			a.openForm(formEdit, p)
		} else {
			a.showMessage("No profile", "Add a profile first (press a).")
		}
	case "p":
		if p := a.selectedProfile(); p != nil {
			a.openForm(formPAT, p)
		} else {
			a.showMessage("No profile", "Add a profile first (press a).")
		}
	case "r":
		a.configureRepo()
	case "d", "<Delete>":
		if p := a.selectedProfile(); p != nil {
			a.openConfirmDelete(p.Name)
		} else {
			a.showMessage("No profile", "Add a profile first (press a).")
		}
	case "<Enter>":
		a.configureRepo()
	}
	a.refresh()
	return false
}

func (a *app) handleFormKey(key string) bool {
	switch key {
	case "<Escape>":
		a.closeModal()
		return false
	case "<C-s>":
		a.saveForm()
		return false
	}

	if a.form != nil && a.form.HandleKeyEvent(key) {
		return false
	}
	return false
}

func (a *app) handleConfirmKey(key string) bool {
	switch key {
	case "y", "Y", "<Enter>":
		a.closeModal()
		a.deleteSelectedProfile()
	case "n", "N", "<Escape>":
		a.closeModal()
	}
	a.refresh()
	return false
}

func (a *app) handleMessageKey(key string) bool {
	switch key {
	case "<Enter>", "<Escape>", "q":
		a.closeModal()
	}
	a.refresh()
	return false
}

func (a *app) handlePickerKey(key string) bool {
	if a.pickerList == nil {
		return false
	}

	switch key {
	case "<Escape>", "n", "N":
		a.closeModal()
		a.pendingGhUsername = ""
		a.setStatus("Repo setup cancelled")
		return false
	case "<Enter>":
		names := a.cfg.ProfileNames()
		idx := a.pickerList.SelectedRow
		if idx < 0 || idx >= len(names) {
			return false
		}
		profileName := names[idx]
		a.closeModal()
		a.pendingGhUsername = ""
		a.selectProfileByName(profileName)
		a.runRepoSetup(profileName)
		return false
	case "j", "<Down>":
		a.pickerList.ScrollDown()
	case "k", "<Up>":
		a.pickerList.ScrollUp()
	}
	return false
}

func (a *app) selectedProfile() *config.Profile {
	names := a.cfg.ProfileNames()
	if len(names) == 0 {
		return nil
	}
	idx := a.profileList.SelectedRow
	if idx < 0 || idx >= len(names) {
		idx = 0
	}
	p, _ := a.cfg.GetProfile(names[idx])
	return p
}

func (a *app) selectedProfileName() string {
	if p := a.selectedProfile(); p != nil {
		return p.Name
	}
	return ""
}

func (a *app) selectProfileByName(name string) {
	names := a.cfg.ProfileNames()
	for i, n := range names {
		if n == name {
			a.profileList.SelectedRow = i
			return
		}
	}
}

func (a *app) loadCwd() {
	if wd, err := os.Getwd(); err == nil {
		a.cwd = wd
	} else {
		a.cwd = ""
	}
}

func (a *app) refresh() {
	a.loadCwd()

	names := a.cfg.ProfileNames()
	rows := make([]string, 0, len(names))

	if len(names) == 0 {
		rows = append(rows, "[(no profiles — press a to add)](fg:gray)")
		a.profileList.SelectedRow = 0
	} else {
		if a.profileList.SelectedRow >= len(names) {
			a.profileList.SelectedRow = len(names) - 1
		}
		if a.profileList.SelectedRow < 0 {
			a.profileList.SelectedRow = 0
		}

		for _, name := range names {
			marker := " "
			if name == a.cfg.CurrentProfile {
				marker = "*"
			}
			p, _ := a.cfg.GetProfile(name)
			pat := patStatus(p)
			rows = append(rows, fmt.Sprintf("%s %s  %s", marker, name, pat))
		}
	}

	a.profileList.Rows = rows
	a.details.Text = a.buildDetailsText()
	a.footer.Text = a.buildFooterText()
}

func (a *app) buildDetailsText() string {
	var b strings.Builder

	if a.cwd != "" {
		fmt.Fprintf(&b, "[Directory](mod:bold):   %s\n", a.cwd)
	}
	if a.busy {
		b.WriteString("[Working...](fg:yellow)\n")
	}
	if a.cwd != "" || a.busy {
		b.WriteString("\n")
	}

	p := a.selectedProfile()
	if p == nil {
		b.WriteString("No profiles yet.\n\nPress [a] to add your first GitHub identity.\nEach profile stores a PAT for HTTPS repo access.")
		if a.statusMsg != "" {
			fmt.Fprintf(&b, "\n\n[Status](mod:bold): %s", a.statusMsg)
		}
		return b.String()
	}

	fmt.Fprintf(&b, "[Name](mod:bold):       %s\n", p.Name)
	if p.GitName != "" {
		fmt.Fprintf(&b, "[Git name](mod:bold):   %s\n", p.GitName)
	} else {
		b.WriteString("[Git name](mod:bold):   (not set)\n")
	}
	if p.PrimaryEmail != "" {
		fmt.Fprintf(&b, "[Email](mod:bold):      %s\n", p.PrimaryEmail)
	} else {
		b.WriteString("[Email](mod:bold):      (not set)\n")
	}
	fmt.Fprintf(&b, "[PAT](mod:bold):        %s\n", patStatus(p))
	if p.GPGKey != "" {
		fmt.Fprintf(&b, "[GPG key](mod:bold):    %s\n", p.GPGKey)
	}
	if a.cfg.CurrentProfile == p.Name {
		b.WriteString("\n[★ Current profile](fg:green)")
	}

	b.WriteString("\n\n[Repo setup](mod:bold)\n")
	b.WriteString("Press [r] or [Enter] to configure this repo:\n")
	b.WriteString("  • set user.name / user.email\n")
	b.WriteString("  • rewrite origin with PAT (HTTPS)\n")
	b.WriteString("  • verify remote access")

	if a.statusMsg != "" {
		fmt.Fprintf(&b, "\n\n[Status](mod:bold): %s", a.statusMsg)
	}
	return b.String()
}

func (a *app) buildFooterText() string {
	if a.busy && a.screen == screenMain {
		return " Working... please wait "
	}
	switch a.screen {
	case screenForm:
		return " ↑↓ field  type to edit  Ctrl+S save  Esc cancel "
	case screenConfirm:
		return " y confirm delete  n/Esc cancel "
	case screenMessage:
		return " Enter/Esc close "
	case screenPicker:
		return " j/k select  Enter confirm  Esc cancel "
	default:
		return " a add  e edit  p PAT  r repo  d delete  j/k nav  Esc/q quit "
	}
}

func patStatus(p *config.Profile) string {
	if p == nil || p.PAT == "" {
		return "[no PAT](fg:red)"
	}
	return "[PAT set](fg:green)"
}

func (a *app) setStatus(msg string) {
	a.statusMsg = msg
}

func (a *app) openForm(kind formMode, existing *config.Profile) {
	a.screen = screenForm
	a.formKind = kind
	a.form = widgets.NewInputList()
	a.form.LabelStyle = ui.NewStyle(ui.ColorCyan)
	a.form.TextStyle = ui.NewStyle(ui.ColorWhite)
	a.form.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorWhite, ui.ModifierBold)

	nameRow := &widgets.InputRow{
		Label:       "Name",
		Placeholder: "profile name (e.g. work)",
		InputType:   widgets.InputTypeText,
	}
	gitNameRow := &widgets.InputRow{
		Label:       "Git name",
		Placeholder: "GitHub username",
		InputType:   widgets.InputTypeText,
	}
	emailRow := &widgets.InputRow{
		Label:       "Email",
		Placeholder: "commit email",
		InputType:   widgets.InputTypeEmail,
	}
	patRow := &widgets.InputRow{
		Label:       "PAT",
		Placeholder: "ghp_... or github_pat_...",
		InputType:   widgets.InputTypePassword,
	}

	if existing != nil {
		nameRow.Value = existing.Name
		gitNameRow.Value = existing.GitName
		emailRow.Value = existing.PrimaryEmail
		patRow.Value = existing.PAT
	}
	if kind == formAdd {
		nameRow.Value = ""
	}
	switch kind {
	case formPAT:
		a.form.AddInputRow(patRow)
	case formEdit:
		a.form.AddInputRow(gitNameRow)
		a.form.AddInputRow(emailRow)
		a.form.AddInputRow(patRow)
	default:
		a.form.AddInputRow(nameRow)
		a.form.AddInputRow(gitNameRow)
		a.form.AddInputRow(emailRow)
		a.form.AddInputRow(patRow)
	}

	title := " Add Profile "
	switch kind {
	case formEdit:
		title = " Edit Profile "
	case formPAT:
		title = " Set PAT "
	}

	a.openModal(a.form, modalFormW, modalFormH, title, ui.ColorYellow)
}

func (a *app) saveForm() {
	if a.form == nil {
		return
	}

	var name, gitName, email, pat string
	for _, row := range a.form.Rows {
		input, ok := row.(*widgets.InputRow)
		if !ok {
			continue
		}
		switch input.Label {
		case "Name":
			name = strings.TrimSpace(input.Value)
		case "Git name":
			gitName = strings.TrimSpace(input.Value)
		case "Email":
			email = strings.TrimSpace(input.Value)
		case "PAT":
			pat = strings.TrimSpace(input.Value)
		}
	}

	if a.formKind == formPAT || a.formKind == formEdit {
		existing := a.selectedProfile()
		if existing == nil {
			a.showMessage("Error", "No profile selected")
			return
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
	}

	if name == "" {
		a.showMessage("Error", "Profile name is required")
		return
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

	if err := profile.Upsert(a.cfg, prof); err != nil {
		a.showMessage("Error", err.Error())
		return
	}
	if err := profile.Save(a.configMgr, a.cfg); err != nil {
		a.showMessage("Error", err.Error())
		return
	}

	saved, _ := a.cfg.GetProfile(name)
	if saved != nil {
		if err := profile.AfterSave(a.cfg, saved); err != nil {
			a.closeModal()
			a.setStatus(fmt.Sprintf("Saved profile '%s' (warning: %s)", name, err.Error()))
			a.refresh()
			return
		}
	}

	a.closeModal()
	a.setStatus(fmt.Sprintf("Saved profile '%s'", name))
	a.refresh()
}

func (a *app) openConfirmDelete(name string) {
	a.screen = screenConfirm
	content := widgets.NewParagraph()
	content.Text = fmt.Sprintf("Delete profile '%s'?\n\nThis removes the profile and its directory rules.\nPAT and credentials are cleared.", name)
	content.TextStyle.Fg = ui.ColorWhite
	content.WrapText = true
	content.Border = false

	a.openModal(content, modalConfirmW, modalConfirmH, " Confirm Delete ", ui.ColorRed)
}

func (a *app) deleteSelectedProfile() {
	name := a.selectedProfileName()
	if name == "" {
		return
	}
	if err := a.cfg.RemoveProfile(name); err != nil {
		a.showMessage("Error", err.Error())
		return
	}
	if err := profile.Save(a.configMgr, a.cfg); err != nil {
		a.showMessage("Error", err.Error())
		return
	}
	if err := profile.AfterRemove(name); err != nil {
		a.setStatus(fmt.Sprintf("Deleted profile '%s' (warning: %s)", name, err.Error()))
		return
	}
	a.setStatus(fmt.Sprintf("Deleted profile '%s'", name))
}

func (a *app) openModal(content ui.Drawable, width, height int, title string, borderFg ui.Color) {
	a.modal = widgets.NewModal()
	a.modal.SetContent(content)
	a.modal.CenterModal(width, height)
	a.modal.Title = title
	a.modal.BorderStyle.Fg = borderFg
	a.fitModalContent()

	a.lm.ClearLayer(ui.LayerModal)
	_ = a.lm.AddLayer(ui.LayerModal, a.modal)
}

func (a *app) fitModalContent() {
	if a.modal == nil || a.modal.Content == nil {
		return
	}
	inner := a.modal.GetDialogInnerRect()
	a.modal.Content.SetRect(0, 0, inner.Dx(), inner.Dy())
}

func (a *app) resizeModal(width, height int) {
	if a.modal == nil {
		return
	}
	a.modal.CenterModal(width, height)
	a.fitModalContent()
}

func (a *app) showMessage(title, text string) {
	a.form = nil
	a.pickerList = nil
	a.pendingGhUsername = ""

	a.screen = screenMessage
	a.message = widgets.NewParagraph()
	a.message.Text = text
	a.message.TextStyle.Fg = ui.ColorWhite
	a.message.WrapText = true
	a.message.Border = false

	a.openModal(a.message, modalMessageW, modalMessageH, " "+title+" ", ui.ColorCyan)
}

func (a *app) closeModal() {
	a.lm.ClearLayer(ui.LayerModal)
	a.screen = screenMain
	a.modal = nil
	a.form = nil
	a.message = nil
	a.pickerList = nil
	a.pendingGhUsername = ""
}

func (a *app) openProfilePicker(ghUsername string) {
	a.screen = screenPicker
	a.pendingGhUsername = ghUsername

	a.pickerList = widgets.NewList()
	names := a.cfg.ProfileNames()
	rows := make([]string, 0, len(names))
	for _, name := range names {
		p, _ := a.cfg.GetProfile(name)
		gitName := p.GitName
		if gitName == "" {
			gitName = "(not set)"
		}
		rows = append(rows, fmt.Sprintf("%s  git-name: %s", name, gitName))
	}
	a.pickerList.Rows = rows
	a.pickerList.SelectedRow = 0
	a.pickerList.TextStyle = ui.NewStyle(ui.ColorWhite)
	a.pickerList.SelectedRowStyle = ui.NewStyle(ui.ColorYellow, ui.ColorBlue, ui.ModifierBold)
	a.pickerList.WrapText = false

	title := fmt.Sprintf(" Owner: %s ", ghUsername)
	a.openModal(a.pickerList, modalPickerW, modalPickerH, title, ui.ColorYellow)
}

func (a *app) configureRepo() {
	if len(a.cfg.ProfileNames()) == 0 {
		a.showMessage("No profiles", "Add a profile first (press a).")
		return
	}
	if a.busy {
		return
	}

	a.busy = true
	a.statusMsg = "Checking repository..."
	a.render()

	repoInfo, err := git.GetGitHubRepoInfo()
	if err != nil {
		a.busy = false
		a.statusMsg = ""
		title, msg := ghsetup.UserFacingError(err)
		a.showMessage(title, msg)
		return
	}

	ghUsername, err := git.ParseGitHubUsername(repoInfo.URL)
	if err != nil {
		a.busy = false
		a.statusMsg = ""
		title, msg := ghsetup.UserFacingError(err)
		a.showMessage(title, msg)
		return
	}

	preferred := a.selectedProfileName()
	profileName, _ := ghsetup.ResolveProfile(a.cfg, ghUsername, preferred)

	a.busy = false
	a.statusMsg = ""

	if profileName == "" {
		a.openProfilePicker(ghUsername)
		return
	}

	if profileName != preferred {
		a.selectProfileByName(profileName)
	}

	a.runRepoSetup(profileName)
}

func (a *app) runRepoSetup(profileName string) {
	if a.busy {
		return
	}

	a.busy = true
	a.statusMsg = "Configuring repository..."
	a.render()

	result, err := ghsetup.ConfigureRepoWithProfile(a.cfg, profileName)
	a.busy = false

	if err != nil {
		a.statusMsg = ""
		title, msg := ghsetup.UserFacingError(err)
		a.showMessage(title, msg)
		return
	}

	if err := profile.Save(a.configMgr, a.cfg); err != nil {
		a.statusMsg = ""
		a.showMessage("Error", err.Error())
		return
	}

	a.setStatus(fmt.Sprintf("Configured %s with profile '%s'", result.RepoName, result.ProfileName))
	a.showMessage("Repo configured",
		fmt.Sprintf("Repository: %s\nRoot: %s\nProfile: %s\nUser: %s\nEmail: %s\n\nRemote verified successfully.",
			result.RepoName, result.RepoRoot, result.ProfileName, result.UserName, result.Email))
}

func (a *app) render() {
	a.refresh()
	ui.RenderLayers(a.lm)
}
