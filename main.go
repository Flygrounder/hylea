package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const gap = "\n\n"

type model struct {
	client *http.Client
	ready  bool

	request  textinput.Model
	response viewport.Model
}

func initialModel() model {
	request := textinput.New()
	request.Prompt = "URL > "
	request.Focus()

	return model{
		request: request,
		client:  http.DefaultClient,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var reqCmd, respCmd tea.Cmd
	m.request, reqCmd = m.request.Update(msg)
	m.response, respCmd = m.response.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.response = viewport.New(msg.Width, msg.Height-lipgloss.Height(m.request.View())-lipgloss.Height(gap))
		m.ready = true
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+n":
			m.response.ScrollDown(1)
		case "ctrl+p":
			m.response.ScrollUp(1)
		case "enter":
			resp, err := m.client.Get(m.request.Value())
			if err != nil {
				break
			}
			res, err := io.ReadAll(resp.Body)
			if err != nil {
				break
			}
			m.response.SetContent(string(res))
		}
	}
	return m, tea.Batch(reqCmd, respCmd)
}

func (m model) View() string {
	return fmt.Sprintf("%s%s%s", m.request.View(), gap, m.response.View())
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	p.Run()
}
