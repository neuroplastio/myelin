package myelin

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/neuroplastio/myelin/myelinkey"
)

// kubeSchema is kubecom's path, declared to myelin. Its keymaps write keys
// as vim does, G for shift+g.
var kubeSchema = Schema{
	Nodes: []NodeType{
		{Name: "cluster", Attrs: map[string]Type{"context": "string", "namespace": "string"}},
		{Name: "menu"},
		{Name: "table", Attrs: map[string]Type{
			"resource": "string", "group": "string", "kind": "string",
			"namespaced": "bool", "drilled": "bool", "filtered": "bool", "unhealthy_only": "bool",
		}},
		{Name: "sort"},
		{Name: "logs", Attrs: map[string]Type{"follow": "bool", "wrap": "bool", "timestamps": "bool", "previous": "bool", "selecting": "bool"}},
		{Name: "viewer", Attrs: map[string]Type{"content": "string"}},
		{Name: "search", Attrs: map[string]Type{"all_kinds": "bool", "all_namespaces": "bool"}},
		{Name: "unhealthy"}, {Name: "forwards"}, {Name: "help"},
		{Name: "picker", Attrs: map[string]Type{"id": "string", "stage": "string"}},
		{Name: "prompt"}, {Name: "confirm"},
		{Name: "field", Text: true},
	},
	Layers:          []string{"default", "user"},
	ShiftedCapitals: true,
	Timeout:         500 * time.Millisecond,
}

// appSchema is an imagined editor's path, for what kubecom's doesn't
// exercise: three layers, a map attribute, and nodes whose attributes a
// program may not know.
var appSchema = Schema{
	Nodes: []NodeType{
		{Name: "window", Attrs: map[string]Type{"name": "string"}},
		{Name: "buffer", Attrs: map[string]Type{
			"path": "string", "lang": "string", "modified": "bool", "readonly": "bool",
			"tags": "map<string,string>",
		}},
		{Name: "sidebar", Attrs: map[string]Type{"side": "string"}},
		{Name: "panel", Attrs: map[string]Type{"id": "string", "ext": "string"}},
		{Name: "input"}, {Name: "popup"}, {Name: "menu"},
		{Name: "palette", Attrs: map[string]Type{"stage": "string"}},
	},
	Layers:  []string{"ext", "core", "user"},
	Aliases: map[string]string{"leader": "ctrl+space"},
	Timeout: time.Second,
}

func kubeEnv(t testing.TB) *Env {
	t.Helper()
	e, err := NewEnv(kubeSchema)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// The steps of kubecom's paths, every attribute given.
func cluster(ctx string) Step {
	return Step{Name: "cluster", Attrs: map[string]any{"context": ctx, "namespace": "default"}}
}

func table(kind string) Step {
	return Step{Name: "table", Attrs: map[string]any{
		"resource": strings.ToLower(kind) + "s", "group": "", "kind": kind,
		"namespaced": true, "drilled": false, "filtered": false, "unhealthy_only": false,
	}}
}

func logsStep(follow bool) Step {
	return Step{Name: "logs", Attrs: map[string]any{"follow": follow, "wrap": false, "timestamps": false, "previous": false, "selecting": false}}
}

func picker(id string) Step {
	return Step{Name: "picker", Attrs: map[string]any{"id": id, "stage": ""}}
}

func node(name string) Step { return Step{Name: name} }

// doc is a document of a layer, its binds in order, each written at the
// next line of test.kdl.
func doc(layer string, binds ...Bind) Doc {
	d := Doc{Layer: layer}
	for i, b := range binds {
		if b.Source == "" {
			b.Source = "test.kdl:" + string(rune('1'+i))
		}
		d.Binds = append(d.Binds, b)
	}
	return d
}

func b(keys, action string) Bind { return Bind{Keys: keys, Action: action} }

func in(when string, bs ...Bind) []Bind {
	for i := range bs {
		bs[i].When = when
	}
	return bs
}

func cat(bs ...[]Bind) []Bind {
	var out []Bind
	for _, x := range bs {
		out = append(out, x...)
	}
	return out
}

func unbind(keys string) Bind { return Bind{Keys: keys, Unbind: true} }

func build(t testing.TB, e *Env, docs ...Doc) *Keymap {
	t.Helper()
	km, problems := e.Build(docs, nil)
	if km == nil {
		t.Fatalf("build: %+v", problems)
	}
	return km
}

// keysOf reads a sequence as kubecom's schema does, G as shift+g.
func keysOf(t testing.TB, s string) []myelinkey.Key {
	t.Helper()
	var ks []myelinkey.Key
	for _, w := range myelinkey.SplitSeq(s) {
		k, err := myelinkey.ParseWith(w, myelinkey.Options{ShiftedCapitals: true})
		if err != nil {
			t.Fatal(err)
		}
		ks = append(ks, k)
	}
	return ks
}

func exact(t testing.TB, km *Keymap, seq string, p Path) string {
	t.Helper()
	m := km.Match(keysOf(t, seq), p)
	if m.Exact == nil {
		return ""
	}
	return m.Exact.Action
}

// Each kind of mistake in a context is a load error, not a key that does
// nothing.
func TestContextsCompileOrFail(t *testing.T) {
	e := kubeEnv(t)
	for src, says := range map[string]string{
		`table &&`:          "Syntax error",
		`tabel.kind == 'x'`: "a context reads focus, cluster, menu",
		`table.kind == 1`:   "no matching overload",
		`table.kind`:        "not a boolean",
	} {
		if _, err := e.Compile(src); err == nil || !strings.Contains(err.Error(), says) {
			t.Errorf("%s: %v, want it to say %q", src, err, says)
		}
	}
	for _, src := range []string{`table`, `table.kind == 'Pod' && !field`, `focus == 'logs'`, `cluster.context.startsWith('prod-')`} {
		if _, err := e.Compile(src); err != nil {
			t.Errorf("%s: %v", src, err)
		}
	}
	if c, err := e.Compile("  "); c != nil || err != nil {
		t.Errorf("an empty context is %v, %v; want nil, which holds everywhere", c, err)
	}
}

// The depth is how far into the path a context reaches: a view's is deeper
// than its table's, saying what is not there adds none, and focus reaches
// exactly as far as the node it names.
func TestDepthIsHowFarTheContextReaches(t *testing.T) {
	e := kubeEnv(t)
	p := Path{cluster("prod-eu"), table("Pod"), logsStep(true)}
	for src, want := range map[string]int{
		`cluster.context == 'prod-eu'`:  1,
		`table.kind == 'Pod'`:           2,
		`logs.follow`:                   3,
		`focus == 'logs'`:               3,
		`focus == 'table'`:              -1, // the logs view holds the keys, not the table
		`!confirm`:                      0,
		`table.kind == 'Deployment'`:    -1,
		`viewer.content == 'secret'`:    -1, // no viewer: absent, does not hold
		`!(viewer.content == 'secret')`: -1, // and its negation does not either
	} {
		c, err := e.Compile(src)
		if err != nil {
			t.Fatal(err)
		}
		if got := c.Depth(p); got != want {
			t.Errorf("%s: depth %d, want %d", src, got, want)
		}
		if got := c.Holds(p); got != (want >= 0) {
			t.Errorf("%s: holds %v", src, got)
		}
	}
	var everywhere *Context
	if everywhere.Depth(p) != 0 || !everywhere.Holds(p) {
		t.Error("no context is not everywhere at depth 0")
	}
	// An attribute the step does not give is absent.
	c, _ := e.Compile(`table.kind == 'Pod'`)
	if c.Holds(Path{node("table")}) {
		t.Error("an attribute the path does not give held")
	}
}

// Precedence: the deepest context, then the layer (user over default), then
// the later bind.
func TestPrecedence(t *testing.T) {
	e := kubeEnv(t)
	km := build(t, e,
		doc("default", cat(
			[]Bind{b("a", "default.shallow")},
			in("logs", b("a", "default.deep")),
			[]Bind{b("b", "default.first"), b("b", "default.second")},
		)...),
		doc("user", b("a", "user.shallow")),
	)
	inLogs := Path{table("Pod"), logsStep(false)}
	inTable := Path{table("Pod")}
	for _, tc := range []struct {
		seq  string
		path Path
		want string
	}{
		{"a", inLogs, "default.deep"},    // depth beats the user's layer
		{"a", inTable, "user.shallow"},   // equal depth: user beats default
		{"b", inTable, "default.second"}, // same layer: the later bind
	} {
		if got := exact(t, km, tc.seq, tc.path); got != tc.want {
			t.Errorf("%s: %s, want %s", tc.seq, got, tc.want)
		}
	}
}

// An exact binding that is also a prefix waits: Match says both.
func TestExactAndPrefix(t *testing.T) {
	km := build(t, kubeEnv(t), doc("default", cat(
		[]Bind{b("g", "go.short"), b("g g", "nav.top")},
		in("logs", b("g x", "go.logs")),
	)...))
	m := km.Match(keysOf(t, "g"), nil)
	if m.Exact == nil || m.Exact.Action != "go.short" || !m.Prefix {
		t.Fatalf("g: %+v", m)
	}
	if m := km.Match(keysOf(t, "g g"), nil); m.Exact == nil || m.Prefix {
		t.Fatalf("g g: %+v", m)
	}
	// A continuation whose context does not hold is no prefix.
	if m := km.Match(keysOf(t, "g x"), nil); m.Exact != nil || m.Prefix {
		t.Fatalf("g x outside the logs: %+v", m)
	}
	if !km.Starts("g") || km.Starts("x") {
		t.Fatal("Starts does not index the first keys")
	}
}

// A shallower continuation does not hold a deeper exact binding up.
func TestAShallowerPrefixDoesNotWait(t *testing.T) {
	km := build(t, kubeEnv(t), doc("default", cat(
		[]Bind{b("enter x", "long")},
		in("focus == 'table'", b("enter", "actions.menu")),
	)...))
	m := km.Match(keysOf(t, "enter"), Path{table("Pod")})
	if m.Exact == nil || m.Exact.Action != "actions.menu" || m.Prefix {
		t.Fatalf("enter: %+v", m)
	}
}

// focus is the innermost node: a binding in `focus == 'table'` is the
// table's own and does not reach an overlay opened over it.
func TestFocusIsTheInnermostNode(t *testing.T) {
	km := build(t, kubeEnv(t), doc("default", cat(
		[]Bind{b("enter", "nav.drillIn")},
		in("focus == 'table'", b("enter", "actions.menu")),
		in("table", b("x", "table.anywhere")),
		in("confirm", b("enter", "confirm.accept")),
	)...))
	c := cluster("kind")
	for _, tc := range []struct {
		seq  string
		path Path
		want string
	}{
		{"enter", Path{c, table("Pod")}, "actions.menu"},
		{"enter", Path{c, table("Pod"), node("forwards")}, "nav.drillIn"},
		{"enter", Path{c, table("Pod"), node("confirm")}, "confirm.accept"},
		{"x", Path{c, table("Pod"), node("forwards")}, "table.anywhere"},
		{"enter", Path{c, node("menu")}, "nav.drillIn"},
	} {
		if got := exact(t, km, tc.seq, tc.path); got != tc.want {
			t.Errorf("%s in %s: %s, want %s", tc.seq, FormatPath(tc.path), got, tc.want)
		}
	}
}

// A path writes its steps outside in, each with the attributes it sets.
func TestFormatPath(t *testing.T) {
	p := Path{
		{Name: "cluster", Attrs: map[string]any{"context": "kind-dev", "namespace": ""}},
		{Name: "table", Attrs: map[string]any{"resource": "pods", "kind": "Pod", "namespaced": true, "filtered": false}},
		{Name: "pane", Attrs: map[string]any{"cols": 80, "annotations": map[string]string{}, "tags": []string{"a"}}},
		picker("palette"),
		node("field"),
	}
	want := `cluster(context="kind-dev") > table(kind="Pod", namespaced, resource="pods") > pane(cols=80, tags=[a]) > picker(id="palette") > field`
	if got := FormatPath(p); got != want {
		t.Errorf("FormatPath = %s\nwant         %s", got, want)
	}
	if got := FormatPath(nil); got != "" {
		t.Errorf("the empty path writes %q", got)
	}
}

// A text field keeps its text: a key that types goes to the field unless a
// binding at least as deep as the field takes it; a key that does not type
// resolves as anywhere.
func TestFieldsKeepTheirText(t *testing.T) {
	km := build(t, kubeEnv(t),
		doc("default", cat(
			[]Bind{b("j", "nav.down"), b("down", "nav.down"), b("g g", "nav.top"), b("ctrl+a", "search.allKinds")},
			in("picker", b("x", "picker.x")),
			in("field", b("y", "field.y")),
		)...),
		doc("user", b("q", "user.q")),
	)
	field := Path{table("Pod"), picker("palette"), node("field")}
	for _, tc := range []struct{ seq, want string }{
		{"j", ""},                     // text: the field's
		{"q", ""},                     // a user's binding everywhere cannot take it either
		{"x", ""},                     // nor a context shallower than the field
		{"y", "field.y"},              // one at the field's depth can
		{"down", "nav.down"},          // no text: resolves
		{"ctrl+a", "search.allKinds"}, // a ctrl chord types nothing
	} {
		if got := exact(t, km, tc.seq, field); got != tc.want {
			t.Errorf("%s in a field: %q, want %q", tc.seq, got, tc.want)
		}
	}
	if m := km.Match(keysOf(t, "g"), field); m.Prefix {
		t.Error("g still starts a sequence in a field")
	}
	noField := Path{table("Pod"), picker("palette")}
	if exact(t, km, "j", noField) != "nav.down" || exact(t, km, "x", noField) != "picker.x" {
		t.Error("with the field closed the text keys do not resolve")
	}
	var next []string
	for _, it := range km.Menu(nil, field, nil).Items {
		next = append(next, string(it.Key))
	}
	if got := strings.Join(next, " "); got != "arrow_down ctrl+a y" {
		t.Errorf("the menu in a field = %s", got)
	}
}

// unbind binds keys to nothing where it wins; unbinding a prefix unbinds the
// group under it.
func TestUnbind(t *testing.T) {
	km := build(t, kubeEnv(t),
		doc("default", b("D", "res.delete"), b("g g", "nav.top"), b("g r", "res.relations")),
		doc("user", cat(
			in("cluster.context.startsWith('prod-')", unbind("D")),
			[]Bind{unbind("g")},
		)...),
	)
	prod := Path{cluster("prod-eu"), table("Pod")}
	dev := Path{cluster("kind"), table("Pod")}
	if exact(t, km, "D", prod) != "" || exact(t, km, "D", dev) != "res.delete" {
		t.Error("the prod-only unbind did not hold where, and only where, it says")
	}
	if m := km.Match(keysOf(t, "g"), dev); m.Prefix || m.Exact != nil {
		t.Errorf("g is still a prefix: %+v", m)
	}
	if exact(t, km, "g g", dev) != "" {
		t.Error("g g survived its group's unbind")
	}
}

// An unbind with no context does not reach a key a lower layer binds in one,
// so it loads with a warning naming the context to unbind it in (kubecom
// KEYS-04).
func TestShallowUnbindWarns(t *testing.T) {
	km, problems := kubeEnv(t).Build([]Doc{
		doc("default", cat([]Bind{b("q", "app.quit")}, in("forwards", Bind{Keys: "X", Action: "forwards.stopAll", Source: "default.kdl:9"}))...),
		doc("user", cat([]Bind{unbind("q"), unbind("X")}, in("forwards", unbind("X")))...),
	}, nil)
	if km == nil {
		t.Fatalf("a warning refused the keymap: %+v", problems)
	}
	if len(problems) != 1 || !problems[0].Warning || problems[0].Source != "test.kdl:2" ||
		problems[0].Text != `unbind "shift+x" leaves shift+x → forwards.stopAll in context "forwards" (default.kdl:9), which wins there; unbind it in that context` {
		t.Fatalf("problems = %+v", problems)
	}
	if exact(t, km, "X", Path{node("forwards")}) != "" {
		t.Error("the unbind in the forwards context did not take X away there")
	}
}

// An alias names a sequence, which bindings write as <name>: the highest
// layer's, the later document's within a layer, else the schema's. A
// binding that names no alias is an error, and an alias's keys name none.
func TestAliases(t *testing.T) {
	e := kubeEnv(t)
	def := doc("default", b("<leader> n", "ns.switch"), b("<go> p", "pods"))
	user := Doc{Layer: "user", Aliases: map[string]string{"leader": "space", "go": "g o"}}
	km := build(t, e, def, user)
	if l, ok := km.Alias("leader"); !ok || len(l) != 1 || l[0] != "space" {
		t.Fatalf("leader = %q, %v", l, ok)
	}
	if exact(t, km, "space n", nil) != "ns.switch" || exact(t, km, "g o p", nil) != "pods" {
		t.Fatal("the aliases did not resolve to the user's keys")
	}
	// A lower layer's alias loses to a higher one's, and within a layer the
	// later document's wins.
	low := Doc{Layer: "default", Aliases: map[string]string{"leader": "ctrl+b", "go": "x"}, Binds: def.Binds}
	km = build(t, e, low, user)
	if l, _ := km.Alias("leader"); l[0] != "space" {
		t.Fatalf("leader = %q", l)
	}
	km = build(t, e, user, Doc{Layer: "user", Aliases: map[string]string{"leader": "ctrl+x"}}, low)
	if l, _ := km.Alias("leader"); l[0] != "ctrl+x" {
		t.Fatalf("the later document's leader lost: %q", l)
	}
	// A dropped layer's aliases go with it.
	km, problems := e.Build([]Doc{low, {Layer: "user", Drops: []string{"default"}, Binds: []Bind{{Keys: "<go>", Action: "x", Source: "u.kdl:1"}}}}, nil)
	if km != nil || len(problems) != 1 || problems[0].Text != `"<go>": no alias <go>` {
		t.Fatalf("a dropped alias: %v, %+v", km, problems)
	}
	km, problems = e.Build([]Doc{def}, nil)
	if km != nil || len(problems) != 2 || problems[0].Source != "test.kdl:1" || problems[0].Text != `"<leader> n": no alias <leader>` {
		t.Fatalf("no leader: %v, %+v", km, problems)
	}
	// The schema's alias stands when no document names one.
	s := kubeSchema
	s.Aliases = map[string]string{"leader": "ctrl+b"}
	e2, err := NewEnv(s)
	if err != nil {
		t.Fatal(err)
	}
	if exact(t, build(t, e2, def, user), "space n", nil) != "ns.switch" ||
		exact(t, build(t, e2, doc("default", b("<leader> n", "ns.switch"))), "ctrl+b n", nil) != "ns.switch" {
		t.Fatal("the schema's leader did not stand")
	}
	for name, keys := range map[string]string{"leader": "<go> x", "lead er": "x", "": "x", "x": " ", "y": "meta+x"} {
		s.Aliases = map[string]string{name: keys}
		if _, err := NewEnv(s); err == nil {
			t.Errorf("alias %q = %q was declared", name, keys)
		}
	}
}

// Drops leave lower layers out, as a `base "none"` in a program's file
// might.
func TestDrops(t *testing.T) {
	e := kubeEnv(t)
	km := build(t, e, doc("default", b("j", "nav.down")), Doc{Layer: "user", Drops: []string{"default"}, Binds: []Bind{b("x", "app.quit")}})
	if exact(t, km, "j", nil) != "" || exact(t, km, "x", nil) != "app.quit" {
		t.Fatal("dropping the defaults kept them")
	}
}

// Every problem names where it was written; any error refuses the keymap,
// and a warning loads it.
func TestBuildProblems(t *testing.T) {
	e := kubeEnv(t)
	check := func(b Binding) error {
		switch b.Action {
		case "nope":
			return errors.New(`"nope" is not an action`)
		case "old":
			return Warn(errors.New(`"old" is deprecated`))
		}
		return nil
	}
	for _, tc := range []struct {
		name    string
		doc     Doc
		source  string
		says    string
		warning bool
	}{
		{"a bad key", doc("user", b("meta+x", "x")), "test.kdl:1", `meta is no modifier`, false},
		{"a bad context", doc("user", Bind{Keys: "x", Action: "x", When: "tabel", Source: "c.kdl:4"}), "c.kdl:4", "undeclared reference", false},
		{"an unknown action", doc("user", b("x", "nope")), "test.kdl:1", `"nope" is not an action`, false},
		{"a deprecated action", doc("user", b("x", "old")), "test.kdl:1", `"old" is deprecated`, true},
		{"an unbind with an action", doc("user", Bind{Keys: "x", Unbind: true, Action: "a"}), "test.kdl:1", "an unbind binds its keys to nothing", false},
		{"a bind with no action", doc("user", Bind{Keys: "x"}), "test.kdl:1", "names no action", false},
		{"an empty sequence", doc("user", b(" ", "x")), "test.kdl:1", "an empty key sequence", false},
		{"an unknown layer", Doc{Layer: "plugin", Source: "p.kdl"}, "p.kdl", `layer "plugin" is not one of default, user`, false},
		{"a drop upward", Doc{Layer: "default", Drops: []string{"user"}, Source: "d.kdl"}, "d.kdl", "not below it", false},
		{"a bad alias", Doc{Layer: "user", Aliases: map[string]string{"leader": "hyper+b"}, Source: "l.kdl"}, "l.kdl", "alias <leader>: ", false},
		{"an alias's bad name", Doc{Layer: "user", Aliases: map[string]string{"a.b": "x"}, Source: "l.kdl"}, "l.kdl", "an alias's name is", false},
		{"Shift with no case", doc("user", b("shift+1", "x")), "test.kdl:1", "shift+1: Shift with a character that has no case", true},
		{"a bad group", Doc{Layer: "user", Groups: []Group{{Keys: "g", Label: "+go", When: "1", Source: "g.kdl:2"}}}, "g.kdl:2", "not a boolean", false},
	} {
		km, problems := e.Build([]Doc{tc.doc}, check)
		if len(problems) != 1 {
			t.Errorf("%s: problems %+v", tc.name, problems)
			continue
		}
		p := problems[0]
		if p.Source != tc.source || !strings.Contains(p.Text, tc.says) || p.Warning != tc.warning {
			t.Errorf("%s: %+v; want %s: %q (warning %v)", tc.name, p, tc.source, tc.says, tc.warning)
		}
		if (km != nil) != tc.warning {
			t.Errorf("%s: keymap %v with warning %v", tc.name, km != nil, tc.warning)
		}
	}
}

// Props pass through untouched; Args nil and [""] stay apart.
func TestBindsPassThrough(t *testing.T) {
	km := build(t, kubeEnv(t), doc("user",
		Bind{Keys: "x", Action: "palette", Props: map[string]string{"steal": "true"}},
		Bind{Keys: "y", Action: "palette", Args: []string{""}},
	))
	all := km.All()
	if all[0].Props["steal"] != "true" || all[0].Args != nil || all[1].Args == nil || all[0].Layer != "user" ||
		len(all[0].Seq) != 1 || all[0].Seq[0] != "x" {
		t.Fatalf("all = %+v", all)
	}
}

// The menu lists one item per next key, in the order the keymap first binds
// it; a key with longer bindings behind it is a group with its label.
func TestMenu(t *testing.T) {
	d := doc("default", cat(
		[]Bind{b("g g", "nav.top"), {Keys: "g r", Action: "res.relations", Label: "relations"}, b("G", "nav.bottom")},
		in("logs", b("g l", "logs.only")),
	)...)
	d.Groups = []Group{{Keys: "g", Label: "+goto"}}
	km := build(t, kubeEnv(t), d)
	menu := km.Menu(nil, nil, nil)
	var got []string
	for _, it := range menu.Items {
		if it.Group {
			got = append(got, string(it.Key)+"="+it.Label)
		} else {
			got = append(got, string(it.Key)+"="+it.Binding.Action)
		}
	}
	if s := strings.Join(got, " "); s != "g=+goto shift+g=nav.bottom" {
		t.Fatalf("menu = %s", s)
	}
	after := km.Menu(keysOf(t, "g"), nil, nil)
	got = nil
	for _, it := range after.Items {
		got = append(got, string(it.Key)+"="+it.Label)
	}
	if s := strings.Join(got, " "); s != "g= r=relations" || after.Label != "+goto" || len(after.Seq) != 1 {
		t.Fatalf("menu after g = %s (%q)", s, after.Label)
	}

	// The program says which bindings apply; a group is enabled when
	// anything under it is.
	applies := func(b Binding) bool { return b.Action == "res.relations" }
	tree := km.Tree(nil, nil, applies)
	if g := tree.Items[0]; !g.Group || !g.Enabled || len(g.Items) != 2 || g.Items[0].Enabled || !g.Items[1].Enabled {
		t.Fatalf("tree's group = %+v", g)
	}
	if tree.Items[1].Enabled || tree.Items[1].Items != nil {
		t.Fatalf("tree's leaf = %+v", tree.Items[1])
	}
	if lvl := km.Menu(nil, nil, func(Binding) bool { return false }); lvl.Items[0].Enabled || lvl.Items[0].Items != nil {
		t.Fatalf("a level's group = %+v", lvl.Items[0])
	}
}

// KeysFor is the sequences that run an action with exactly its values, in
// load order; KeyFor is the key column's one.
func TestKeysFor(t *testing.T) {
	km := build(t, kubeEnv(t),
		doc("default", b("k", "nav.up"), b("up", "nav.up"), Bind{Keys: "g p", Action: "resources.switch", Args: []string{"pods"}},
			b("R", "resources.switch"), b("g g", "nav.top"), b("home", "nav.top")),
		doc("user", unbind("up")),
	)
	var got []string
	for _, ks := range km.KeysFor("nav.up", nil, nil) {
		got = append(got, myelinkey.FormatSeq(ks))
	}
	if s := strings.Join(got, ","); s != "k" {
		t.Fatalf("nav.up = %s", s)
	}
	if ks := km.KeysFor("resources.switch", []string{"pods"}, nil); len(ks) != 1 || myelinkey.FormatSeq(ks[0]) != "g p" {
		t.Fatalf("resources.switch pods = %v", ks)
	}
	if ks := km.KeysFor("nav.top", nil, nil); len(ks) != 2 || myelinkey.FormatSeq(ks[0]) != "g g" {
		t.Fatalf("nav.top is not in load order: %v", ks)
	}
	// The key column: as it is before with values, then the shorter.
	if b := km.KeyFor("resources.switch", nil); b == nil || myelinkey.FormatSeq(b.Seq) != "shift+r" {
		t.Fatalf("key column for resources.switch = %+v", b)
	}
	if b := km.KeyFor("nav.top", nil); b == nil || myelinkey.FormatSeq(b.Seq) != "home" {
		t.Fatalf("key column for nav.top = %+v", b)
	}
	if km.KeyFor("nav.none", nil) != nil {
		t.Fatal("a key for an unbound action")
	}
}

// Explain ranks every candidate: the winner, the losers, the longer ones,
// then those whose context does not hold.
func TestExplain(t *testing.T) {
	km := build(t, kubeEnv(t),
		doc("default", cat(
			[]Bind{b("a", "shallow")},
			in("logs", b("a", "deep")),
			in("viewer", b("a", "elsewhere")),
			[]Bind{b("a b", "longer")},
		)...),
		doc("user", b("a", "user")),
	)
	cs := km.Explain(keysOf(t, "a"), Path{table("Pod"), logsStep(false)})
	var got []string
	for _, c := range cs {
		s := c.Binding.Action
		if c.Wins {
			s += "!"
		}
		if !c.Holds {
			s += "?"
		}
		got = append(got, s)
	}
	if s := strings.Join(got, " "); s != "deep! user shallow longer elsewhere?" {
		t.Fatalf("explain = %s", s)
	}
	if cs[0].Depth != 2 || cs[1].Depth != 0 {
		t.Fatalf("depths: %+v", cs)
	}
}

// Shadowed finds a higher layer's binding a deeper lower one hides on a path
// the program names.
func TestShadowed(t *testing.T) {
	km := build(t, kubeEnv(t),
		doc("default", in("logs", Bind{Keys: "w", Action: "logs.wrap", Source: "default.kdl:3"})...),
		doc("user", b("w", "user.w"), b("z", "user.z")),
	)
	ps := km.Shadowed(Path{table("Pod")}, Path{table("Pod"), logsStep(false)})
	if len(ps) != 1 || ps[0].Source != "test.kdl:1" || !ps[0].Warning ||
		!strings.Contains(ps[0].Text, "w → user.w is shadowed where table(") ||
		!strings.Contains(ps[0].Text, "default's w → logs.wrap (default.kdl:3) has a deeper context there") {
		t.Fatalf("shadowed = %+v", ps)
	}
}

// The sequencer: a prefix buffers until a deadline, a finished binding runs,
// and keys that match nothing drop; Expire runs the exact binding of the
// pending keys after the deadline and does nothing before it.
func TestSequencer(t *testing.T) {
	km := build(t, kubeEnv(t), doc("default", b("g", "go.short"), b("g g", "nav.top"), b("j", "nav.down"), b("z z", "fold")))
	t0 := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

	st, res := Feed(km, Pending{}, "x", nil, t0)
	if len(st.Seq) != 0 || res.Run != nil || len(res.Dropped) != 1 || res.Dropped[0] != "x" {
		t.Fatalf("an unbound key: %+v %+v", st, res)
	}
	st, res = Feed(km, Pending{}, "j", nil, t0)
	if res.Run == nil || res.Run.Action != "nav.down" || len(st.Seq) != 0 {
		t.Fatalf("j: %+v", res)
	}
	st, res = Feed(km, Pending{}, "g", nil, t0)
	if want := t0.Add(500 * time.Millisecond); !res.Deadline.Equal(want) || !st.Deadline.Equal(want) || res.Run != nil {
		t.Fatalf("g buffers: %+v %+v", st, res)
	}
	if st2, res := Expire(km, st, t0.Add(100*time.Millisecond)); len(st2.Seq) != 1 || res.Run != nil {
		t.Fatalf("Expire before the deadline did something: %+v", res)
	}
	if _, res := Expire(km, st, t0.Add(time.Second)); res.Run == nil || res.Run.Action != "go.short" {
		t.Fatalf("g timed out: %+v", res)
	}
	if _, res := Feed(km, st, "g", nil, t0.Add(100*time.Millisecond)); res.Run == nil || res.Run.Action != "nav.top" {
		t.Fatalf("g g: %+v", res)
	}
	st, _ = Feed(km, Pending{}, "z", nil, t0)
	if st2, res := Feed(km, st, "q", nil, t0); len(st2.Seq) != 0 || len(res.Dropped) != 2 {
		t.Fatalf("z q drops both: %+v", res)
	}
	if _, res := Expire(km, st, t0.Add(time.Second)); res.Run != nil || len(res.Dropped) != 1 {
		t.Fatalf("z timed out: %+v", res)
	}

	// Arm is an alias as if typed.
	e := kubeEnv(t)
	km = build(t, e, Doc{Layer: "default", Aliases: map[string]string{"leader": "ctrl+b"}, Binds: []Bind{b("<leader> n", "ns.switch")}})
	st = Arm(km, "leader", nil, t0)
	if len(st.Seq) != 1 || st.Seq[0] != "ctrl+b" {
		t.Fatalf("Arm = %+v", st)
	}
	if _, res := Feed(km, st, "n", nil, t0); res.Run == nil || res.Run.Action != "ns.switch" {
		t.Fatalf("armed n: %+v", res)
	}
	if st := Arm(build(t, e, doc("default", b("x", "y"))), "leader", nil, t0); len(st.Seq) != 0 {
		t.Fatalf("armed with no leader: %+v", st)
	}
}

// A path's attributes compare the same whether built in Go or decoded from
// another encoder's msgpack: an int is an int of any Go kind.
func TestIntAttributes(t *testing.T) {
	e, err := NewEnv(Schema{Nodes: []NodeType{{Name: "pane", Attrs: map[string]Type{"cols": "int", "ratio": "double", "tags": "list<string>", "ann": "map<string, string>"}}}, Layers: []string{"core"}})
	if err != nil {
		t.Fatal(err)
	}
	c, err := e.Compile(`pane.cols == 80 && pane.ratio > 1.5 && 'x' in pane.tags && pane.ann['k'] == 'v'`)
	if err != nil {
		t.Fatal(err)
	}
	for _, cols := range []any{80, int8(80), int64(80), uint64(80), uint(80)} {
		p := Path{{Name: "pane", Attrs: map[string]any{"cols": cols, "ratio": 2.0, "tags": []any{"x"}, "ann": map[string]any{"k": "v"}}}}
		if !c.Holds(p) {
			t.Errorf("cols as %T does not hold", cols)
		}
	}
	if _, err := NewEnv(Schema{Nodes: []NodeType{{Name: "pane", Attrs: map[string]Type{"x": "list<"}}}, Layers: []string{"core"}}); err == nil {
		t.Error("a bad type was declared")
	}
}

// MayHold and MayDepth evaluate with the attributes the path does not give
// unknown: "can this hold wherever a buffer has the focus?", and how deep
// a context may reach in one extension's panel.
func TestPartialEvaluation(t *testing.T) {
	e, err := NewEnv(appSchema)
	if err != nil {
		t.Fatal(err)
	}
	overBuffer := Path{node("window"), node("buffer")}
	for src, want := range map[string]bool{
		``:                        true,
		`buffer`:                  true,
		`buffer.modified`:         true, // unknown: it might
		`buffer.tags['x'] == 'y'`: true,
		`!buffer`:                 false,
		`panel.id == 'files'`:     false, // no panel there
		`palette`:                 false,
		`!palette && !input`:      true,
	} {
		c, err := e.Compile(src)
		if err != nil {
			t.Fatal(err)
		}
		if got := c.MayHold(overBuffer); got != want {
			t.Errorf("%q may hold over a buffer: %v, want %v", src, got, want)
		}
	}
	// The panel's attributes given, the rest unknown.
	files := Path{node("window"), node("sidebar"), {Name: "panel", Attrs: map[string]any{"id": "files", "ext": "git"}}}
	for src, want := range map[string]int{
		`panel.ext == 'git'`:                       3,
		`panel.ext == 'other'`:                     -1,
		`sidebar.side == 'left'`:                   2, // unknown: it may hold, at the sidebar
		`panel.id == 'files' && !input`:            3,
		`sidebar.side == 'left' && panel.id == ''`: -1,
	} {
		c, err := e.Compile(src)
		if err != nil {
			t.Fatal(err)
		}
		if got := c.MayDepth(files); got != want {
			t.Errorf("%q may reach %d, want %d", src, got, want)
		}
	}
}

// CheckSeq reads a Bind's sequence as Build does, an alias on trust.
func TestCheckSeq(t *testing.T) {
	strict := kubeSchema
	strict.StrictKeys = true
	se, err := NewEnv(strict)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		e       *Env
		s, says string
	}{
		{kubeEnv(t), "<leader> p n", ""},
		{kubeEnv(t), "g g", ""},
		{kubeEnv(t), "<any_name> G", ""},
		{kubeEnv(t), "Ctrl+X esc", ""},
		{kubeEnv(t), "", "an empty key sequence"},
		{kubeEnv(t), "g meta+x", "meta is no modifier"},
		{kubeEnv(t), "<a.b> n", "an alias's name is"},
		{se, "<leader> G ctrl+x", ""},
		{se, "g Escape", `"Escape": write escape`},
		{se, "shift+G", `"shift+G": write shift+g`},
	} {
		err := tc.e.CheckSeq(tc.s)
		if (err == nil) != (tc.says == "") || (err != nil && !strings.Contains(err.Error(), tc.says)) {
			t.Errorf("CheckSeq(%q) = %v, want %q", tc.s, err, tc.says)
		}
	}
}
