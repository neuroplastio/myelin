package myelin_test

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/neuroplastio/myelin"

	"github.com/neuroplastio/myelin/key"
)

// A file manager's keymap: its defaults and a user's config, matched on the
// paths the keyboard takes.
func Example() {
	env, err := myelin.NewEnv(myelin.Schema{
		Nodes: []myelin.NodeType{
			{Name: "window"},
			{Name: "list", Attrs: map[string]myelin.Type{"dir": "string"}},
			{Name: "field", Text: true},
		},
		Layers:  []string{"defaults", "user"},
		Timeout: 500 * time.Millisecond,
	})
	if err != nil {
		panic(err)
	}
	defaults := myelin.Doc{Layer: "defaults", Binds: []myelin.Bind{
		{Keys: "j", Action: "down"},
		{Keys: "g g", Action: "top"},
		{Keys: "esc", Action: "back"},
		{Keys: "esc", Action: "close-filter", When: "field"},
		{Keys: "enter", Action: "open", When: "list"},
	}}
	user := myelin.Doc{Layer: "user", Binds: []myelin.Bind{
		{Keys: "enter", Action: "preview", When: "list.dir.startsWith('/tmp')"},
	}}
	km, problems := env.Build([]myelin.Doc{defaults, user}, nil)
	if km == nil {
		panic(fmt.Sprint(problems))
	}

	home := myelin.Path{{Name: "window"}, {Name: "list", Attrs: map[string]any{"dir": "/home/me"}}}
	tmp := myelin.Path{{Name: "window"}, {Name: "list", Attrs: map[string]any{"dir": "/tmp"}}}
	filter := append(slices.Clone(home), myelin.Step{Name: "field"})
	match := func(keys string, p myelin.Path) string {
		seq, err := key.ParseSeq(keys)
		if err != nil {
			return err.Error()
		}
		switch m := km.Match(seq, p); {
		case m.Prefix:
			return "wait for more"
		case m.Exact != nil:
			return m.Exact.Action
		}
		return "nothing"
	}
	fmt.Println("enter in a list:", match("enter", home))
	fmt.Println("enter in /tmp:", match("enter", tmp))
	fmt.Println("esc in the filter:", match("esc", filter))
	fmt.Println("j in the filter:", match("j", filter))
	fmt.Println("g:", match("g", home))
	// Output:
	// enter in a list: open
	// enter in /tmp: preview
	// esc in the filter: close-filter
	// j in the filter: nothing
	// g: wait for more
}

// A program checks its file's sequences while it parses it, before it has
// a keymap.
func ExampleCheckSeq() {
	for _, s := range []string{"<leader> f n", "g g", "ctrl+X"} {
		fmt.Printf("%s: %v\n", s, myelin.CheckSeq(s))
	}
	// Output:
	// <leader> f n: <nil>
	// g g: <nil>
	// ctrl+X: "ctrl+X": write ctrl+shift+x
}

// Build checks each binding with the program's own rules: an error refuses
// the keymap, and a warning loads it.
func ExampleEnv_Build() {
	env, err := myelin.NewEnv(myelin.Schema{
		Nodes:  []myelin.NodeType{{Name: "list"}},
		Layers: []string{"defaults", "user"},
	})
	if err != nil {
		panic(err)
	}
	actions := map[string]bool{"down": true, "up": true, "scroll": true}
	check := func(b myelin.Binding) error {
		switch {
		case !actions[b.Action]:
			return fmt.Errorf("%q is not an action", b.Action)
		case b.Action == "scroll":
			return myelin.Warn(errors.New(`"scroll" is going away: use "down"`))
		}
		return nil
	}
	user := myelin.Doc{Layer: "user", Binds: []myelin.Bind{
		{Keys: "J", Action: "scroll", Source: "config:2"},
		{Keys: "shift+k", Action: "up", Source: "config:3"},
		{Keys: "x", Action: "explode", Source: "config:4"},
		{Keys: "y", Action: "up", When: "lsit", Source: "config:5"},
	}}
	km, problems := env.Build([]myelin.Doc{user}, check)
	for _, p := range problems {
		kind := "error"
		if p.Warning {
			kind = "warning"
		}
		fmt.Printf("%s: %s: %s\n", p.Source, kind, p.Text)
	}
	fmt.Println("loaded:", km != nil)
	// Output:
	// config:2: warning: "scroll" is going away: use "down"
	// config:3: error: "shift+k": write K
	// config:4: error: "explode" is not an action
	// config:5: error: context "lsit": undeclared reference to 'lsit' (a context reads focus, list)
	// loaded: false
}

// A context's depth is how far into the path it reaches: the deepest wins.
func ExampleContext_Depth() {
	env, err := myelin.NewEnv(myelin.Schema{
		Nodes:  []myelin.NodeType{{Name: "window"}, {Name: "list"}, {Name: "preview"}, {Name: "field", Text: true}},
		Layers: []string{"defaults"},
	})
	if err != nil {
		panic(err)
	}
	p := myelin.Path{{Name: "window"}, {Name: "list"}, {Name: "field"}}
	for _, when := range []string{"", "window", "list", "field", "list && !preview", "!preview", "preview"} {
		c, err := env.Compile(when)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%q reaches %d\n", when, c.Depth(p))
	}
	// Output:
	// "" reaches 0
	// "window" reaches 1
	// "list" reaches 2
	// "field" reaches 3
	// "list && !preview" reaches 2
	// "!preview" reaches 0
	// "preview" reaches -1
}

// The sequencer holds no clock: the time goes in with each key, and a
// deadline comes out for the program's loop to wake at.
func ExampleFeed() {
	env, err := myelin.NewEnv(myelin.Schema{
		Nodes:   []myelin.NodeType{{Name: "list"}},
		Layers:  []string{"defaults"},
		Timeout: 500 * time.Millisecond,
	})
	if err != nil {
		panic(err)
	}
	km, problems := env.Build([]myelin.Doc{{Layer: "defaults", Binds: []myelin.Bind{
		{Keys: "g", Action: "go"},
		{Keys: "g g", Action: "top"},
	}}}, nil)
	if km == nil {
		panic(fmt.Sprint(problems))
	}
	p := myelin.Path{{Name: "list"}}
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

	var st myelin.Pending
	st, res := myelin.Feed(km, st, "g", p, now)
	fmt.Println("g waits until", res.Deadline.Format("15:04:05.000"))
	st, res = myelin.Feed(km, st, "g", p, now.Add(200*time.Millisecond))
	fmt.Println("g g runs", res.Run.Action)

	st, _ = myelin.Feed(km, st, "g", p, now)
	st, res = myelin.Expire(km, st, now.Add(100*time.Millisecond))
	fmt.Println("before the deadline, Expire does nothing:", res.Run == nil)
	st, res = myelin.Expire(km, st, now.Add(500*time.Millisecond))
	fmt.Println("at the deadline, g runs", res.Run.Action)

	_, res = myelin.Feed(km, st, "x", p, now)
	fmt.Println("x matches nothing:", res.Dropped)
	// Output:
	// g waits until 12:00:00.500
	// g g runs top
	// before the deadline, Expire does nothing: true
	// at the deadline, g runs go
	// x matches nothing: [x]
}

// The leader's menu: one item per key, a group where more keys follow, and
// what the program says does nothing now dimmed.
func ExampleKeymap_Menu() {
	env, err := myelin.NewEnv(myelin.Schema{
		Nodes:  []myelin.NodeType{{Name: "list"}},
		Layers: []string{"defaults"},
		Leader: "space",
	})
	if err != nil {
		panic(err)
	}
	km, problems := env.Build([]myelin.Doc{{Layer: "defaults",
		Groups: []myelin.Group{{Keys: "<leader> f", Label: "+file"}},
		Binds: []myelin.Bind{
			{Keys: "<leader> f n", Action: "file.new", Label: "New file"},
			{Keys: "<leader> f d", Action: "file.delete", Label: "Delete"},
			{Keys: "<leader> q", Action: "quit", Label: "Quit"},
			{Keys: "<leader> left", Action: "back"},
		},
	}}, nil)
	if km == nil {
		panic(fmt.Sprint(problems))
	}
	readOnly := func(b myelin.Binding) bool { return b.Action != "file.new" && b.Action != "file.delete" }

	leader, _ := km.Leader()
	show := func(m myelin.Menu) {
		fmt.Printf("%s (%s)\n", key.FormatSeq(m.Seq), m.Label)
		for _, it := range m.Items {
			label := it.Label
			if label == "" {
				label = it.Binding.Action // the program words a binding with no label
			}
			fmt.Printf("  %s  %s enabled=%v\n", key.Display(it.Key), label, it.Enabled)
		}
	}
	p := myelin.Path{{Name: "list"}}
	show(km.Menu([]key.Key{leader}, p, readOnly))
	show(km.Menu([]key.Key{leader, "f"}, p, readOnly))
	// Output:
	// space ()
	//   f  +file enabled=false
	//   q  Quit enabled=true
	//   ←  back enabled=true
	// space f (+file)
	//   n  New file enabled=false
	//   d  Delete enabled=false
}
