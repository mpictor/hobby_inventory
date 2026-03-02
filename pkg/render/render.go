package render

import (
	"iter"

	"charm.land/bubbles/v2/table"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Rows interface {
	ColumnHeaders() ([]string, error)
	Len() int
	All() iter.Seq[[]string]
}

// much of this copied from https://github.com/charmbracelet/bubbletea/blob/main/examples/table/main.go

func Render(rows Rows) error {
	var columns []table.Column
	tr := make([]table.Row, 0, rows.Len())
	chs, err := rows.ColumnHeaders()
	if err != nil {
		return err
	}
	for _, ch := range chs {
		columns = append(columns, table.Column{Title: ch /*TODO width*/})
	}
	for r := range rows.All() {
		tr = append(tr, r)
	}
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(tr),
		table.WithFocused(true),
		// table.WithHeight(7),
		// table.WithWidth(42),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	m := model{t}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return err
	}
	return nil
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	table table.Model
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m, tea.Batch(
				tea.Printf("Let's go to %s!", m.table.SelectedRow()[1]),
			)
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	return tea.NewView(baseStyle.Render(m.table.View()) + "\n  " + m.table.HelpView() + "\n")
}
