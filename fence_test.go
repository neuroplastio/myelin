package myelin

import (
	"strings"
	"testing"
	"time"

	"github.com/neuroplastio/myelin/myelinkey"
)

// modeSchema is an imagined multiplexer's path: a window in a mode, its
// pane, and a palette over it with a field.
var modeSchema = Schema{
	Nodes: []NodeType{
		{Name: "window", Attrs: map[string]Type{"mode": "string"}},
		{Name: "pane"}, {Name: "palette"},
		{Name: "field", Text: true},
	},
	Layers:  []string{"default", "user"},
	Aliases: map[string]string{"leader": "ctrl+b"},
	Timeout: time.Second,
}

func modeEnv(t testing.TB) *Env {
	t.Helper()
	e, err := NewEnv(modeSchema)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func window(mode string) Step { return Step{Name: "window", Attrs: map[string]any{"mode": mode}} }

const locked = `window.mode == "locked"`

// modeBinds are a binding everywhere, one in the locked mode, and one in a
// pane.
func modeBinds() []Bind {
	return cat(
		[]Bind{b("ctrl+q", "quit"), b("<leader> n", "pane.new")},
		in(locked, b("ctrl+g", "unlock")),
		in("pane", b("j", "pane.down")),
	)
}

// fenced is a document of modeBinds with fences.
func fenced(fs ...Fence) Doc {
	d := doc("default", modeBinds()...)
	d.Fences = fs
	return d
}

// fed is what Feed makes of one key on a path with nothing pending: the
// action it runs, or "dropped" or "swallowed".
func fed(t *testing.T, km *Keymap, key string, p Path) string {
	t.Helper()
	st, res := Feed(km, Pending{}, keysOf(t, key)[0], p, time.Time{})
	switch {
	case len(st.Seq) > 0:
		return "pending"
	case res.Run != nil:
		return res.Run.Action
	case len(res.Dropped) > 0 && len(res.Swallowed) == 0:
		return "dropped"
	case len(res.Swallowed) > 0 && len(res.Dropped) == 0:
		return "swallowed"
	}
	t.Fatalf("%s: %+v", key, res)
	return ""
}

// A fence's two options make four modes: keys fall through or not, and what
// matches nothing goes to the program or nowhere. Off the fence's context
// nothing changes.
func TestFenceModes(t *testing.T) {
	e := modeEnv(t)
	on, off := Path{window("locked"), node("pane")}, Path{window("normal"), node("pane")}
	for _, tc := range []struct {
		name            string
		shadow, swallow bool
		want            string // ctrl+q, ctrl+g, j, x in the mode
	}{
		{"fall through, to the program", false, false, "quit unlock pane.down dropped"},
		{"fall through, nowhere", false, true, "quit unlock pane.down swallowed"},
		{"shadow, to the program", true, false, "dropped unlock pane.down dropped"},
		{"shadow, nowhere", true, true, "swallowed unlock pane.down swallowed"},
	} {
		km := build(t, e, fenced(Fence{When: locked, Shadow: tc.shadow, Swallow: tc.swallow, Source: "test.kdl:9"}))
		var got, outside []string
		for _, k := range []string{"ctrl+q", "ctrl+g", "j", "x"} {
			got = append(got, fed(t, km, k, on))
			outside = append(outside, fed(t, km, k, off))
		}
		if s := strings.Join(got, " "); s != tc.want {
			t.Errorf("%s: %s, want %s", tc.name, s, tc.want)
		}
		if s := strings.Join(outside, " "); s != "quit dropped pane.down dropped" {
			t.Errorf("%s, off the mode: %s", tc.name, s)
		}
		if f := km.Fence(on); f == nil || f.Source != "test.kdl:9" {
			t.Errorf("%s: Fence = %+v", tc.name, f)
		}
		if f := km.Fence(off); f != nil {
			t.Errorf("%s: Fence off the mode = %+v", tc.name, f)
		}
	}
}

// A mode that lets nothing past still lets a field deeper than it take its
// text: a key that types text drops to the program, which types it, and the
// field rule still holds against the mode's own bindings.
func TestFenceKeepsAFieldsText(t *testing.T) {
	km := build(t, modeEnv(t), fenced(Fence{When: locked, Shadow: true, Swallow: true}))
	field := Path{window("locked"), node("pane"), node("palette"), node("field")}
	for _, tc := range []struct{ key, want string }{
		{"a", "dropped"},        // the field's text
		{"j", "dropped"},        // the pane's binding, shallower than the field
		{"ctrl+x", "swallowed"}, // types nothing: the mode's
		{"ctrl+g", "unlock"},
	} {
		if got := fed(t, km, tc.key, field); got != tc.want {
			t.Errorf("%s in the mode's field: %s, want %s", tc.key, got, tc.want)
		}
	}
	// A field that is the fence's own node is not deeper than it.
	km = build(t, modeEnv(t), Doc{Layer: "default", Fences: []Fence{{When: "field", Swallow: true}}})
	if got := fed(t, km, "a", field); got != "swallowed" {
		t.Errorf("a in a field that fences itself: %s", got)
	}
}

// A sequence that dies is decided on the path its first key was typed on,
// in Feed and in Expire.
func TestFenceDecidesADyingSequence(t *testing.T) {
	km := build(t, modeEnv(t), fenced(Fence{When: locked, Swallow: true}))
	on, off := Path{window("locked"), node("pane")}, Path{window("normal"), node("pane")}
	t0 := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	leader := myelinkey.Key("ctrl+b")
	for _, tc := range []struct {
		name     string
		typed, q Path
		want     string
	}{
		{"in the mode", on, on, "swallowed"},
		{"leaving the mode", on, off, "swallowed"},
		{"entering the mode", off, on, "dropped"},
	} {
		st, _ := Feed(km, Pending{}, leader, tc.typed, t0)
		_, res := Feed(km, st, "q", tc.q, t0)
		got := "dropped"
		if len(res.Swallowed) == 2 && len(res.Dropped) == 0 {
			got = "swallowed"
		} else if len(res.Dropped) != 2 {
			got = "neither"
		}
		if got != tc.want {
			t.Errorf("<leader> q %s: %+v, want %s", tc.name, res, tc.want)
		}
	}
	st, _ := Feed(km, Pending{}, leader, on, t0)
	if _, res := Expire(km, st, t0.Add(2*time.Second)); len(res.Swallowed) != 1 || len(res.Dropped) != 0 {
		t.Errorf("<leader> timing out in the mode: %+v", res)
	}
	st, _ = Feed(km, Pending{}, leader, off, t0)
	if _, res := Expire(km, st, t0.Add(2*time.Second)); len(res.Dropped) != 1 || len(res.Swallowed) != 0 {
		t.Errorf("<leader> timing out off the mode: %+v", res)
	}
}

// One fence decides on a path: the deepest that holds, then the higher
// layer, then the later. A deeper fence with neither option gives the
// default back, as a palette opened in a mode does.
func TestFencePrecedence(t *testing.T) {
	e := modeEnv(t)
	mode := Fence{When: locked, Shadow: true, Swallow: true, Source: "mode"}
	palette := Path{window("locked"), node("pane"), node("palette")}
	km := build(t, e, fenced(mode, Fence{When: "palette", Source: "palette"}))
	if f := km.Fence(palette); f == nil || f.Source != "palette" {
		t.Fatalf("the deeper fence does not decide: %+v", f)
	}
	if got := fed(t, km, "ctrl+q", palette) + " " + fed(t, km, "x", palette); got != "quit dropped" {
		t.Errorf("in a palette in the mode: %s", got)
	}
	if got := fed(t, km, "ctrl+q", palette[:2]); got != "swallowed" {
		t.Errorf("in the mode out of the palette: %s", got)
	}

	open := Fence{When: locked, Source: "open"}
	for _, tc := range []struct {
		name string
		docs []Doc
		want string
	}{
		{"the higher layer", []Doc{fenced(mode), {Layer: "user", Fences: []Fence{open}}}, "open"},
		{"the higher layer, loaded first", []Doc{{Layer: "user", Fences: []Fence{open}}, fenced(mode)}, "open"},
		{"the later fence", []Doc{fenced(mode, open)}, "open"},
		{"the later fence, the other way", []Doc{fenced(open, mode)}, "mode"},
		{"a dropped layer's fence", []Doc{fenced(mode), {Layer: "user", Drops: []string{"default"}}}, ""},
	} {
		km := build(t, e, tc.docs...)
		got := ""
		if f := km.Fence(palette[:2]); f != nil {
			got = f.Source
		}
		if got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

// A shadowing fence hides the bindings it shadows from menus, and Explain
// says which they are.
func TestFenceMenusAndExplain(t *testing.T) {
	d := fenced(Fence{When: locked, Shadow: true})
	d.Groups = []Group{{Keys: "<leader>", Label: "+leader"}}
	km := build(t, modeEnv(t), d)
	on := Path{window("locked"), node("pane")}
	keys := func(m Menu) string {
		var ks []string
		for _, it := range m.Items {
			ks = append(ks, string(it.Key))
		}
		return strings.Join(ks, " ")
	}
	if got := keys(km.Menu(nil, on, nil)); got != "ctrl+g j" {
		t.Errorf("the menu in the mode: %s", got)
	}
	if got := keys(km.Tree(nil, on, nil)); got != "ctrl+g j" {
		t.Errorf("the tree in the mode: %s", got)
	}
	if got := keys(km.Menu(nil, Path{window("normal"), node("pane")}, nil)); got != "ctrl+q ctrl+b j" {
		t.Errorf("the menu off the mode: %s", got)
	}
	cs := km.Explain(keysOf(t, "ctrl+q"), on)
	if len(cs) != 1 || !cs[0].Holds || !cs[0].Fenced || cs[0].Wins {
		t.Errorf("explain ctrl+q in the mode: %+v", cs)
	}
	if cs := km.Explain(keysOf(t, "ctrl+g"), on); len(cs) != 1 || cs[0].Fenced || !cs[0].Wins {
		t.Errorf("explain ctrl+g in the mode: %+v", cs)
	}
}
