package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/calghar/gas-cli/internal/ghsetup"
	"github.com/calghar/gas-cli/internal/git"
)

func (m model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	main := m.renderMain()

	switch {
	case m.messageOpen:
		return m.placeDialog(main, m.renderMessageDialog(), dialogStyle)
	case m.screen == screenForm:
		return m.placeDialog(main, m.renderFormDialog(), dialogWarnStyle)
	case m.screen == screenConfirm:
		return m.placeDialog(main, m.renderConfirmDialog(), dialogDangerStyle)
	case m.screen == screenPicker:
		return m.placeDialog(main, m.renderPickerDialog(), dialogWarnStyle)
	default:
		return main
	}
}

func (m model) placeDialog(base, dialog string, style lipgloss.Style) string {
	box := style.Render(dialog)
	// Overlay dialog on the main screen so content stays visible around the edges.
	return overlayCenter(base, box, m.width, m.height)
}

func overlayCenter(background, box string, width, height int) string {
	if width <= 0 || height <= 0 {
		return background
	}

	bgLines := strings.Split(background, "\n")
	for len(bgLines) < height {
		bgLines = append(bgLines, "")
	}
	if len(bgLines) > height {
		bgLines = bgLines[:height]
	}

	boxLines := strings.Split(box, "\n")
	boxW, boxH := 0, len(boxLines)
	for _, line := range boxLines {
		if w := lipgloss.Width(line); w > boxW {
			boxW = w
		}
	}

	startY := max(0, (height-boxH)/2)
	startX := max(0, (width-boxW)/2)

	for i, bl := range boxLines {
		y := startY + i
		if y >= height {
			break
		}
		bgLines[y] = spliceLine(bgLines[y], bl, startX, width)
	}
	return strings.Join(bgLines, "\n")
}

func spliceLine(baseLine, insert string, startX, totalWidth int) string {
	baseLine = padLine(baseLine, totalWidth)
	insertW := lipgloss.Width(insert)
	if startX >= totalWidth {
		return baseLine
	}
	left := padLine(truncateANSI(baseLine, startX), startX)
	remaining := totalWidth - startX - insertW
	if remaining < 0 {
		remaining = 0
	}
	right := padLine(truncateANSI(baseLine, startX+insertW), remaining)
	return left + insert + right
}

func padLine(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func truncateANSI(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	// Walk runes until display width exceeds maxWidth.
	var b strings.Builder
	width := 0
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			b.WriteRune(r)
			continue
		}
		if inEscape {
			b.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		rw := lipgloss.Width(string(r))
		if width+rw > maxWidth {
			break
		}
		width += rw
		b.WriteRune(r)
	}
	return b.String()
}

func (m model) renderMain() string {
	headerH := 5
	if m.height < 22 {
		headerH = 4
	}
	footerH := 1
	bodyH := max(1, m.height-headerH-footerH)

	listW := m.width / 3
	if listW < 28 {
		listW = 28
	}
	if listW > m.width-20 {
		listW = m.width / 2
	}
	detailsW := max(10, m.width-listW)

	header := headerStyle.Width(m.width).Height(headerH).Render(m.buildHeaderText())
	list := panelStyle.Width(listW).Height(bodyH).Render(
		panelTitleStyle.Render("Profiles") + "\n" + m.buildProfileList(),
	)
	details := panelStyle.Width(detailsW).Height(bodyH).Render(
		panelTitleStyle.Render("Details") + "\n" + m.buildDetailsText(),
	)
	body := lipgloss.JoinHorizontal(lipgloss.Top, list, details)
	footer := footerStyle.Width(m.width).Render(m.buildFooterText())

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m model) buildHeaderText() string {
	var b strings.Builder
	selected := m.selectedProfileName()
	if selected != "" {
		fmt.Fprintf(&b, "gas-cli — Repo Identity\n\nProfile: %s", selected)
		if m.cfg.CurrentProfile == selected {
			b.WriteString("  " + lipgloss.NewStyle().Foreground(colorGreen).Render("applied"))
		}
	} else {
		b.WriteString("gas-cli — Repo Identity\n\nProfile: (none selected)")
	}

	if !m.hasGitRepo {
		b.WriteString("\n.git/config: not found in current directory")
		return b.String()
	}

	local, err := git.ReadLocalRepoConfig()
	if err != nil {
		fmt.Fprintf(&b, "\n.git/config: %s", err.Error())
		return b.String()
	}

	if repo, err := git.GetRemoteOriginURL(); err == nil {
		if ghUser, err := git.ParseGitHubUsername(repo.URL); err == nil {
			resolved, _ := ghsetup.ResolveProfile(m.cfg, ghUser, selected)
			if resolved != "" && resolved != selected {
				fmt.Fprintf(&b, "  → will use: %s", resolved)
			}
		}
	}

	b.WriteString("\n")
	if local.UserName != "" {
		fmt.Fprintf(&b, "user.name: %s   ", local.UserName)
	} else {
		b.WriteString("user.name: (not set)   ")
	}
	if local.UserEmail != "" {
		fmt.Fprintf(&b, "user.email: %s", local.UserEmail)
	} else {
		b.WriteString("user.email: (not set)")
	}
	if local.OriginURL != "" {
		fmt.Fprintf(&b, "\norigin: %s", local.OriginURL)
	} else {
		b.WriteString("\norigin: (not set)")
	}
	return b.String()
}

func (m model) buildProfileList() string {
	names := m.cfg.ProfileNames()
	if len(names) == 0 {
		return lipgloss.NewStyle().Foreground(colorGray).Render("(no profiles — press a to add)")
	}

	idx := m.profileIdx
	if idx >= len(names) {
		idx = len(names) - 1
	}
	if idx < 0 {
		idx = 0
	}

	var b strings.Builder
	for i, name := range names {
		marker := " "
		if name == m.cfg.CurrentProfile {
			marker = "*"
		}
		p, _ := m.cfg.GetProfile(name)
		row := fmt.Sprintf("%s %s  %s", marker, name, patStatus(p))
		if i == idx {
			row = selectedStyle.Render("› " + row)
		} else {
			row = normalRowStyle.Render("  " + row)
		}
		b.WriteString(row)
		if i < len(names)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m model) buildDetailsText() string {
	var b strings.Builder

	if m.cwd != "" {
		fmt.Fprintf(&b, "Directory:   %s\n", m.cwd)
	}
	if m.hasGitRepo {
		b.WriteString("Git repo:    " + lipgloss.NewStyle().Foreground(colorGreen).Render("yes — r enabled") + "\n")
	} else {
		b.WriteString("Git repo:    " + lipgloss.NewStyle().Foreground(colorRed).Render("no .git here — r disabled") + "\n")
	}
	if m.busy {
		b.WriteString(lipgloss.NewStyle().Foreground(colorYellow).Render("Working...") + "\n")
	}
	if m.cwd != "" || m.busy {
		b.WriteString("\n")
	}

	p := m.selectedProfile()
	if p == nil {
		b.WriteString("No profiles yet.\n\nPress a to add your first GitHub identity.\nEach profile stores a PAT for HTTPS repo access.")
		if !m.hasGitRepo {
			b.WriteString("\n\nCd to a repository root with .git before using r.")
		}
		if m.statusMsg != "" {
			fmt.Fprintf(&b, "\n\nStatus: %s", m.statusMsg)
		}
		return b.String()
	}

	fmt.Fprintf(&b, "Name:       %s\n", p.Name)
	if p.GitName != "" {
		fmt.Fprintf(&b, "Git name:   %s\n", p.GitName)
	} else {
		b.WriteString("Git name:   (not set)\n")
	}
	if p.PrimaryEmail != "" {
		fmt.Fprintf(&b, "Email:      %s\n", p.PrimaryEmail)
	} else {
		b.WriteString("Email:      (not set)\n")
	}
	fmt.Fprintf(&b, "PAT:        %s\n", patStatus(p))
	if p.GPGKey != "" {
		fmt.Fprintf(&b, "GPG key:    %s\n", p.GPGKey)
	}
	if m.cfg.CurrentProfile == p.Name {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(colorGreen).Render("★ Current profile"))
	}

	b.WriteString("\n\nRepo setup\n")
	if m.hasGitRepo {
		b.WriteString("Press r or Enter to configure this repo:\n")
		b.WriteString("  • set user.name / user.email\n")
		b.WriteString("  • rewrite origin with PAT (HTTPS)\n")
		b.WriteString("  • verify remote access")
	} else {
		b.WriteString("Cd to a repository root with .git, then press r.")
	}
	if m.statusMsg != "" {
		fmt.Fprintf(&b, "\n\nStatus: %s", m.statusMsg)
	}
	return b.String()
}

func (m model) buildFooterText() string {
	if m.busy && m.screen == screenMain {
		return " Working... please wait "
	}
	switch m.screen {
	case screenForm:
		return " ↑↓/Tab field  Enter save  Ctrl+S save  Esc cancel "
	case screenConfirm:
		return " y confirm delete  n/Esc cancel "
	case screenMessage:
		return " any key back to main "
	case screenPicker:
		return " j/k select  Enter confirm  Esc cancel "
	default:
		return " a add  e edit  p PAT  r repo  d delete  j/k nav  Esc/q quit "
	}
}

func (m model) renderMessageDialog() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(colorCyan).Render(m.messageTitle)
	return title + "\n\n" + m.messageText
}

func (m model) renderFormDialog() string {
	title := "Add Profile"
	switch m.formKind {
	case formEdit:
		title = "Edit Profile"
	case formPAT:
		title = "Set PAT"
	}
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorYellow).Render(title))
	b.WriteString("\n\n")
	for i, field := range m.formFields {
		line := field.View()
		if i == m.formFocus {
			line = focusedFieldStyle.Render(line)
		} else {
			line = blurredFieldStyle.Render(line)
		}
		b.WriteString(line)
		if i < len(m.formFields)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m model) renderConfirmDialog() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(colorRed).Render("Confirm Delete")
	body := fmt.Sprintf("Delete profile '%s'?\n\nThis removes the profile and its directory rules.\nPAT and credentials are cleared.", m.confirmName)
	return title + "\n\n" + body
}

func (m model) renderPickerDialog() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(colorYellow).Render(
		fmt.Sprintf("Owner: %s", m.pendingGhUsername),
	)
	names := m.cfg.ProfileNames()
	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\nSelect a profile:\n")
	for i, name := range names {
		p, _ := m.cfg.GetProfile(name)
		gitName := p.GitName
		if gitName == "" {
			gitName = "(not set)"
		}
		row := fmt.Sprintf("%s  git-name: %s", name, gitName)
		if i == m.pickerIdx {
			row = selectedStyle.Render("› " + row)
		} else {
			row = normalRowStyle.Render("  " + row)
		}
		b.WriteString(row)
		if i < len(names)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
