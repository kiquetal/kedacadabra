package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	boxStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	valueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	activeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func (m Model) View() tea.View {
	var b strings.Builder

	b.WriteString(titleStyle.Render("⚡ kedacadabra — KEDA Schedule Manager"))
	b.WriteString("\n\n")

	// Status dashboard
	s := m.status
	rows := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %s",
		labelStyle.Render("Pause schedule:"), valueStyle.Render(s.PauseSchedule),
		labelStyle.Render("Resume schedule:"), valueStyle.Render(s.ResumeSchedule),
		labelStyle.Render("Paused replicas:"), valueStyle.Render(s.PausedReplicas),
		labelStyle.Render("Replicas:"), valueStyle.Render(s.Replicas),
	)
	if s.Error != "" {
		rows += "\n" + errStyle.Render(s.Error)
	}
	b.WriteString(boxStyle.Render(rows))
	b.WriteString("\n\n")

	// Mode tabs
	rel := " Relative (F1) "
	abs := " Absolute (F2) "
	if m.mode == modeRelative {
		b.WriteString(activeStyle.Render(rel) + " " + dimStyle.Render(abs))
	} else {
		b.WriteString(dimStyle.Render(rel) + " " + activeStyle.Render(abs))
	}
	b.WriteString("\n\n")

	// Editor
	if m.mode == modeRelative {
		fmt.Fprintf(&b, "%s  %s\n%s  %s",
			labelStyle.Render("Pause in (min):"), m.relInputs[0].View(),
			labelStyle.Render("Resume in (min):"), m.relInputs[1].View(),
		)
	} else {
		fmt.Fprintf(&b, "%s\n  %s %s  %s %s  %s %s\n%s\n  %s %s  %s %s  %s %s",
			labelStyle.Render("Pause:"),
			labelStyle.Render("day:"), m.absInputs[0].View(),
			labelStyle.Render("hour:"), m.absInputs[1].View(),
			labelStyle.Render("min:"), m.absInputs[2].View(),
			labelStyle.Render("Resume:"),
			labelStyle.Render("day:"), m.absInputs[3].View(),
			labelStyle.Render("hour:"), m.absInputs[4].View(),
			labelStyle.Render("min:"), m.absInputs[5].View(),
		)
	}
	b.WriteString("\n\n")

	// Message
	if m.loading {
		b.WriteString(m.spinner.View() + " Applying...")
	} else if m.msg != "" {
		if m.msgErr {
			b.WriteString(errStyle.Render(m.msg))
		} else {
			b.WriteString(okStyle.Render(m.msg))
		}
	}
	b.WriteString("\n\n")

	// Help
	b.WriteString(m.help.View(m.keys))
	b.WriteString("\n")

	return tea.NewView(b.String())
}
