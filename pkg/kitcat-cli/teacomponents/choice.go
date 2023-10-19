package teacomponents

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
	"strings"
)

var (
	choiceItemHintColor = lipgloss.NewStyle().Foreground(lipgloss.Color("#767676"))
)

type ChoiceItem struct {
	Value string
	Hint  string
}

type Choice struct {
	Question string
	Choices  []ChoiceItem

	cursor int
	Choice string
}

func (m *Choice) Init() tea.Cmd {
	_, i, ok := lo.FindIndexOf(m.Choices, func(item ChoiceItem) bool {
		return item.Value == m.Choice
	})

	if ok {
		m.cursor = i
	}
	return nil
}

func (m *Choice) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "enter", tea.KeySpace.String():
			// Send the Choice on the channel and exit.
			m.Choice = m.Choices[m.cursor].Value
			return m, tea.Quit

		case "down", "j":
			m.cursor++
			if m.cursor >= len(m.Choices) {
				m.cursor = 0
			}

		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.Choices) - 1
			}
		}
	}

	return m, nil
}

func (m *Choice) View() string {
	s := strings.Builder{}
	s.WriteString(m.Question + "\n\n")

	for i := 0; i < len(m.Choices); i++ {
		if m.cursor == i {
			s.WriteString("(•) ")
		} else {
			s.WriteString("( ) ")
		}
		s.WriteString(m.Choices[i].Value)
		s.WriteString(" ")
		if m.Choices[i].Hint != "" {
			s.WriteString(choiceItemHintColor.Render(m.Choices[i].Hint))
		}
		s.WriteString("\n")
	}
	s.WriteString("\n(press q to quit, enter to select)\n")

	return s.String()
}
