package myelintea_test

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/neuroplastio/myelin"
	"github.com/neuroplastio/myelin/myelintea"
)

// model is a Bubble Tea model whose keys go through a myelin keymap.
type model struct {
	km      *myelin.Keymap
	path    myelin.Path
	pending myelin.Pending
	ran     []string
}

// expire wakes the model at a sequence's deadline.
type expire struct{ at time.Time }

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var res myelin.Result
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		k, ok := myelintea.Key(msg.Key())
		if !ok {
			return m, nil
		}
		m.pending, res = myelin.Feed(m.km, m.pending, k, m.path, time.Now())
	case expire:
		m.pending, res = myelin.Expire(m.km, m.pending, msg.at)
	}
	switch {
	case res.Run != nil:
		m.ran = append(m.ran, res.Run.Action)
	case !res.Deadline.IsZero():
		// Wake at the deadline, with the tick's time as now.
		return m, tea.Tick(time.Until(res.Deadline), func(t time.Time) tea.Msg { return expire{t} })
	}
	return m, nil
}

func (m model) View() tea.View { return tea.NewView(strings.Join(m.ran, "\n")) }

func Example() {
	env, err := myelin.NewEnv(myelin.Schema{
		Nodes:   []myelin.NodeType{{Name: "list"}},
		Layers:  []string{"defaults"},
		Timeout: 500 * time.Millisecond,
	})
	if err != nil {
		panic(err)
	}
	km, problems := env.Build([]myelin.Doc{{Layer: "defaults", Binds: []myelin.Bind{
		{Keys: "g g", Action: "top"},
		{Keys: "ctrl+d", Action: "half-page-down"},
	}}}, nil)
	if km == nil {
		panic(fmt.Sprint(problems))
	}
	var m tea.Model = model{km: km, path: myelin.Path{{Name: "list"}}}
	for _, k := range []tea.KeyPressMsg{
		{Code: 'g', Text: "g"},
		{Code: 'g', Text: "g"},
		{Code: 'd', Mod: tea.ModCtrl},
	} {
		m, _ = m.Update(k)
	}
	fmt.Println(m.(model).ran)
	// Output: [top half-page-down]
}
