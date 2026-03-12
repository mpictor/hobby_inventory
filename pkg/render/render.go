package render

import (
	"iter"
	"log"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var Verbose bool

type Rows interface {
	ColumnHeaders(cols []int) []string
	Len() int
	All(cols []int) iter.Seq[interface{ Strings() []string }]
}

// type RowStrings interface {
// 	Strings() []string
// }

// much of this copied from https://github.com/charmbracelet/bubbletea/blob/main/examples/table/main.go

func Render(rows Rows, ord []int) error {
	var columns []table.Column
	tr := make([]table.Row, 0, rows.Len())
	chs := rows.ColumnHeaders(ord)
	if Verbose {
		log.Print(chs)
	}
	// find widths
	widths := make([]int, len(chs))
	for r := range rows.All(ord) {
		for i, str := range r.Strings() {
			widths[i] = max(widths[i], len(str))
		}
	}
	totalWidth := 0
	// drop empty cols
	displayedCols := 0
	for i, ch := range chs {
		if widths[i] == 0 {
			if Verbose {
				log.Printf("drop col %d %s", i, ch)
			}
			continue
		}
		w := max(widths[i], len(ch))
		columns = append(columns, table.Column{Title: ch, Width: w})
		totalWidth += w + 2
		displayedCols++
	}
	for r := range rows.All(ord) {
		dispRow := make([]string, displayedCols)
		j := 0
		for i, w := range widths {
			if w > 0 {
				dispRow[j] = r.Strings()[i]
				j++
			}
		}
		if Verbose {
			log.Print(dispRow)
		}
		tr = append(tr, dispRow)
	}
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(tr),
		table.WithFocused(true),
		table.WithHeight(7),
		// NOTE undersize width affects data rows but not column headers!
		// not sure why it can't compute based on column width
		table.WithWidth(totalWidth),
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
