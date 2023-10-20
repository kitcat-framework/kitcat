package teacomponents

import "github.com/charmbracelet/bubbles/textinput"

func NewTextInput(def string) textinput.Model {
	ti := textinput.New()
	ti.SetValue(def)
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return ti
}
