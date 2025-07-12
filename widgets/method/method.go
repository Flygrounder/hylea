package method

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	hasFocus bool
	input    textinput.Model
}

func New() Model {
	input := textinput.New()
	input.SetValue("GET")
	input.Prompt = ""
	return Model{
		input: input,
	}
}

func (m Model) View() string {
	var color lipgloss.TerminalColor
	if m.hasFocus {
		color = lipgloss.Color("#ff0000")
	}
	return lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(color).Render(m.input.View())
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.hasFocus {
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.hasFocus = false
			m.input.Blur()
		}
	}
	return m, cmd
}

func (m Model) Value() string {
	return m.input.Value()
}

func (m *Model) Focus() tea.Cmd {
	m.hasFocus = true
	return m.input.Focus()
}

func (m Model) Focused() bool {
	return m.hasFocus
}
