package myelin

import (
	"os"
	"strings"
	"testing"

	"github.com/neuroplastio/myelin/myelinkey"
)

// kubecom's default.kdl at bc2fae3 (internal/tui/keymap), as the Doc its
// parser reads from it.
var kubeDefaults = Doc{Layer: "default", Source: "default.kdl", Binds: cat(
	[]Bind{
		b("k", "nav.up"), b("up", "nav.up"),
		b("j", "nav.down"), b("down", "nav.down"),
		b("h", "nav.left"), b("left", "nav.left"),
		b("l", "nav.right"), b("right", "nav.right"),
		b("enter", "nav.drillIn"), b("esc", "nav.back"),
		b("g g", "nav.top"), b("home", "nav.top"),
		b("G", "nav.bottom"), b("end", "nav.bottom"),
		b("ctrl+d", "nav.halfPageDown"), b("pgdn", "nav.halfPageDown"),
		b("ctrl+u", "nav.halfPageUp"), b("pgup", "nav.halfPageUp"),
		b("space", "nav.pageDown"), b("ctrl+b", "nav.pageUp"),
		b("/", "app.filter"), b("n", "app.searchNext"), b("#", "app.searchPrev"),
		b("H", "app.unhealthy"), b("U", "app.unhealthyScan"),
		b("?", "app.help"), b("q", "app.quit"), b("ctrl+c", "app.quit"),
		b(":", "app.palette"),
		b("N", "ns.switch"), b("R", "resources.switch"), b("C", "ctx.switch"),
		b("T", "theme.switch"), b("M", "mouse.toggle"),
		b("S", "sort.column"), b("x", "sort.clear"), b("m", "menu.toggle"),
		b("d", "res.describe"), b("E", "res.events"), b("L", "res.logs"),
		b("e", "res.edit"), b("D", "res.delete"), b("P", "res.children"),
		b("g r", "res.relations"),
		b("F", "forwards.panel"), b("ctrl+f", "search.cluster"),
		b("*", "menu.pin"),
	},
	in("logs",
		b("f", "logs.follow"), b("ctrl+r", "logs.regex"), b("w", "logs.wrap"),
		b("t", "logs.timestamps"), b("o", "logs.previous"), b("ctrl+p", "logs.previous"),
		b("v", "logs.select"), b("V", "logs.select"), b("y", "logs.yank")),
	in("viewer.content == 'secret'", b("r", "secret.reveal"), b("c", "secret.copy")),
	in("forwards", b("X", "forwards.stopAll")),
	in("picker.id == 'ports'", b("p", "forwards.localPort"), b("0", "forwards.freeLocal")),
	in("search", b("ctrl+a", "search.allKinds"), b("ctrl+w", "search.allNamespaces")),
	in("focus == 'table'", b("enter", "actions.menu")),
	in("confirm",
		b("y", "confirm.accept"), b("enter", "confirm.accept"),
		b("n", "confirm.decline"), b("esc", "confirm.decline")),
)}

// kubecom's port to contexts kept every key, and so does myelin's:
// testdata/accept/kubecom-oldkeys.golden is kubecom's own golden at bc2fae3,
// what its old flat keymap resolved on each surface, one line per scenario,
// key and action. myelin resolves kubecom's default.kdl to the same, with
// kubecom's one allowed difference (KEYS-04): an action bound on its surface
// only resolves to nothing off it.
func TestKubecomKeepsItsOldKeys(t *testing.T) {
	e := kubeEnv(t)
	km := build(t, e, kubeDefaults)
	surfaces := map[string]*Context{}
	for a, when := range map[string]string{
		"logs.follow": "logs", "logs.regex": "logs", "logs.wrap": "logs", "logs.timestamps": "logs",
		"logs.previous": "logs", "logs.select": "logs", "logs.yank": "logs",
		"secret.reveal": "viewer.content == 'secret'", "secret.copy": "viewer.content == 'secret'",
		"forwards.stopAll":   "forwards",
		"forwards.localPort": "picker.id == 'ports'", "forwards.freeLocal": "picker.id == 'ports'",
		"search.allKinds": "search", "search.allNamespaces": "search",
	} {
		c, err := e.Compile(when)
		if err != nil {
			t.Fatal(err)
		}
		surfaces[a] = c
	}
	c, pods := cluster("kind-dev"), table("Pod")
	search := Step{Name: "search", Attrs: map[string]any{"all_kinds": false, "all_namespaces": false}}
	viewer := Step{Name: "viewer", Attrs: map[string]any{"content": "secret"}}
	field := node("field")
	paths := map[string]Path{
		"menu":           {c, node("menu")},
		"menu filter":    {c, node("menu"), field},
		"table":          {c, pods},
		"table filter":   {c, pods, field},
		"sort":           {c, pods, node("sort")},
		"logs":           {c, pods, logsStep(true)},
		"logs grep":      {c, pods, logsStep(false), field},
		"viewer":         {c, pods, viewer},
		"search results": {c, pods, search},
		"search query":   {c, node("menu"), search, field},
		"unhealthy":      {c, node("menu"), node("unhealthy")},
		"forwards":       {c, pods, node("forwards")},
		"help":           {c, node("menu"), node("help")},
		"picker":         {c, pods, picker("ports")},
		"picker query":   {c, pods, picker("palette"), field},
		"prompt":         {c, pods, node("prompt"), field},
		"confirm":        {c, pods, node("confirm")},
		"no cluster":     {},
	}
	src, err := os.ReadFile("testdata/accept/kubecom-oldkeys.golden")
	if err != nil {
		t.Fatal(err)
	}
	lines := 0
	for _, line := range strings.Split(strings.TrimSuffix(string(src), "\n"), "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 3 {
			t.Fatalf("golden line %q", line)
		}
		p, ok := paths[f[0]]
		if !ok {
			t.Fatalf("golden scenario %q has no path", f[0])
		}
		keys := keysOf(t, f[1])
		lines++
		m := km.Match(keys, p)
		var got string
		if m.Exact != nil {
			got = m.Exact.Action
		}
		want := f[2]
		if c, ok := surfaces[want]; ok && !c.Holds(p) {
			want = ""
		}
		if got != want {
			t.Errorf("%s: %s resolves to %q, want %q (the old keymap: %q)", f[0], f[1], got, want, f[2])
		}
		if len(keys) > 1 {
			if m.Prefix {
				t.Errorf("%s: %s is a prefix of a longer binding", f[0], f[1])
			}
			if !km.Match(keys[:1], p).Prefix {
				t.Errorf("%s: %s does not start a sequence", f[0], keys[0])
			}
		}
	}
	if lines < 3000 {
		t.Fatalf("the golden has %d lines", lines)
	}
}

// kubecom's menu is Menu with no keys typed: every first key on the path,
// `g` a group.
func TestKubecomMenu(t *testing.T) {
	km := build(t, kubeEnv(t), kubeDefaults)
	m := km.Menu(nil, Path{cluster("kind-dev"), table("Pod"), logsStep(true)}, nil)
	var keys []string
	groups := map[string]bool{}
	for _, it := range m.Items {
		keys = append(keys, myelinkey.Display(it.Key))
		groups[myelinkey.Display(it.Key)] = it.Group
	}
	if got := strings.Join(keys[:6], " "); got != "k ↑ j ↓ h ←" {
		t.Errorf("the menu starts %s", got)
	}
	if !groups["g"] || groups["f"] || !strings.Contains(strings.Join(keys, " "), " f ") {
		t.Errorf("g is a group %v, f %v, in %s", groups["g"], groups["f"], keys)
	}
	g := km.Menu(keysOf(t, "g"), Path{cluster("kind-dev"), table("Pod")}, nil)
	if len(g.Items) != 2 || g.Items[0].Binding.Action != "nav.top" || g.Items[1].Binding.Action != "res.relations" {
		t.Errorf("g's level is %+v", g.Items)
	}
}
