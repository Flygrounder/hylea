package main

import (
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type item string

func (i item) FilterValue() string {
	return string(i)
}

var _ list.Item = item("")

type itemDelegate struct{}

func (i itemDelegate) Height() int {
	return 1
}

func (i itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it := listItem.(item)
	var res string
	if m.Index() == index {
		res = ">" + string(it)
	} else {
		res = string(it)
	}
	_, _ = w.Write([]byte(res))
}

func (i itemDelegate) Spacing() int {
	return 0
}

func (i itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

var _ list.ItemDelegate = itemDelegate{}
