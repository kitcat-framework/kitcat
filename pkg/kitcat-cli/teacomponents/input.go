package teacomponents

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type (
	errMsg error
)

type Input struct {
	Question  string
	TextInput textinput.Model
	err       error
}

func (m Input) Init() tea.Cmd {
	return textinput.Blink
}

func (m Input) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.TextInput, cmd = m.TextInput.Update(msg)
	return m, cmd
}

func (m Input) View() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		m.Question,
		m.TextInput.View(),
		"(esc to quit)",
	) + "\n"
}
