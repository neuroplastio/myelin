package myelin

import (
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/tinylib/msgp/msgp"

	"github.com/neuroplastio/myelin/myelinkey"
)

// What myelin aims to cost: a key no binding starts with is one map lookup;
// matching a key against a binding with a context takes a few microseconds;
// compiling a context, once at load, some tens. Run them with make bench,
// and compare two runs with benchstat.

// benchKeymap is a keymap the size of a program's defaults: a hundred
// bindings, half under contexts, and a group per leader key.
func benchKeymap(b testing.TB) (*Keymap, Path) {
	e, err := NewEnv(appSchema)
	if err != nil {
		b.Fatal(err)
	}
	km, problems := e.Build(benchDocs(25), nil)
	if km == nil {
		b.Fatal(problems)
	}
	return km, benchPath
}

// benchDocs is n letters' bindings: four for each, two of them in
// contexts, and a group under the leader.
func benchDocs(n int) []Doc {
	var binds []Bind
	var groups []Group
	for i := range n {
		k := string(rune('a' + i%26))
		if i >= 26 {
			k = fmt.Sprintf("f%d", i%24+1)
		}
		binds = append(binds,
			Bind{Keys: "<leader> " + k, Action: "core." + k, Label: "Core " + k},
			Bind{Keys: "<leader> " + k + " " + k, Action: "core.twice." + k},
			Bind{Keys: "ctrl+" + k, Action: "panel." + k, When: "sidebar && panel.ext == 'git' && !input"},
			Bind{Keys: "alt+" + k, Action: "buffer." + k, When: "buffer.tags['format'] == 'gofmt'"},
		)
		groups = append(groups, Group{Keys: "<leader> " + k, Label: "+" + k})
	}
	return []Doc{{Layer: "core", Binds: binds, Groups: groups}}
}

// benchPath is a git panel in the left sidebar: three steps deep.
var benchPath = Path{
	{Name: "window", Attrs: map[string]any{"name": "main"}},
	{Name: "sidebar", Attrs: map[string]any{"side": "left"}},
	{Name: "panel", Attrs: map[string]any{"id": "changes", "ext": "git"}},
}

// deepPath is a palette's input over a buffer: six steps deep.
var deepPath = Path{
	{Name: "window", Attrs: map[string]any{"name": "main"}},
	{Name: "buffer", Attrs: map[string]any{"path": "main.go", "lang": "go", "modified": true, "readonly": false,
		"tags": map[string]string{"format": "gofmt"}}},
	{Name: "sidebar", Attrs: map[string]any{"side": "left"}},
	{Name: "panel", Attrs: map[string]any{"id": "changes", "ext": "git"}},
	{Name: "palette", Attrs: map[string]any{"stage": "pick"}},
	{Name: "popup"},
}

var (
	leader = myelinkey.Key("ctrl+space")
	now    = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
)

func BenchmarkFeedUnboundKey(b *testing.B) {
	km, p := benchKeymap(b)
	b.ReportAllocs()
	for b.Loop() {
		Feed(km, Pending{}, "z", p, now)
	}
}

func BenchmarkFeedRunsContextual(b *testing.B) {
	km, p := benchKeymap(b)
	b.ReportAllocs()
	for b.Loop() {
		if _, r := Feed(km, Pending{}, "ctrl+k", p, now); r.Run == nil {
			b.Fatal("ctrl+k ran nothing")
		}
	}
}

// The leader, then k: a buffered key, then a prefix that waits for k k.
func BenchmarkFeedSequence(b *testing.B) {
	km, p := benchKeymap(b)
	b.ReportAllocs()
	for b.Loop() {
		st, _ := Feed(km, Pending{}, leader, p, now)
		if st, _ = Feed(km, st, "k", p, now); len(st.Seq) != 2 {
			b.Fatal("leader k does not wait")
		}
	}
}

func BenchmarkMatchContextual(b *testing.B) {
	km, p := benchKeymap(b)
	seq := []myelinkey.Key{"ctrl+k"}
	b.ReportAllocs()
	for b.Loop() {
		if km.Match(seq, p).Exact == nil {
			b.Fatal("no match")
		}
	}
}

func BenchmarkMatchDeepPath(b *testing.B) {
	km, _ := benchKeymap(b)
	seq := []myelinkey.Key{"alt+k"}
	b.ReportAllocs()
	for b.Loop() {
		if km.Match(seq, deepPath).Exact == nil {
			b.Fatal("no match")
		}
	}
}

func BenchmarkMatchLeaderPrefix(b *testing.B) {
	km, p := benchKeymap(b)
	seq := []myelinkey.Key{leader, "k"}
	b.ReportAllocs()
	for b.Loop() {
		if !km.Match(seq, p).Prefix {
			b.Fatal("no prefix")
		}
	}
}

func BenchmarkMenuRoot(b *testing.B) {
	km, p := benchKeymap(b)
	b.ReportAllocs()
	for b.Loop() {
		km.Menu(nil, p, nil)
	}
}

func BenchmarkMenuLeader(b *testing.B) {
	km, p := benchKeymap(b)
	seq := []myelinkey.Key{leader}
	b.ReportAllocs()
	for b.Loop() {
		km.Menu(seq, p, nil)
	}
}

func BenchmarkTreeLeader(b *testing.B) {
	km, p := benchKeymap(b)
	seq := []myelinkey.Key{leader}
	b.ReportAllocs()
	for b.Loop() {
		km.Tree(seq, p, nil)
	}
}

func BenchmarkEffective(b *testing.B) {
	km, _ := benchKeymap(b)
	b.ReportAllocs()
	for b.Loop() {
		km.Effective(deepPath)
	}
}

func BenchmarkExplain(b *testing.B) {
	km, p := benchKeymap(b)
	seq := []myelinkey.Key{"ctrl+k"}
	b.ReportAllocs()
	for b.Loop() {
		km.Explain(seq, p)
	}
}

// The key column: one KeyFor per row of a menu.
func BenchmarkKeyFor(b *testing.B) {
	km, p := benchKeymap(b)
	b.ReportAllocs()
	for b.Loop() {
		if km.KeyFor("panel.k", p) == nil {
			b.Fatal("no key for panel.k")
		}
	}
}

func BenchmarkCompile(b *testing.B) {
	e, err := NewEnv(appSchema)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := e.Compile("sidebar && panel.ext == 'git' && !input"); err != nil {
			b.Fatal(err)
		}
	}
}

// Building a keymap: a hundred bindings, then a thousand. Their contexts
// repeat, as a program's do, so each is compiled once.
func BenchmarkBuild(b *testing.B) {
	for _, n := range []int{25, 250} {
		b.Run(fmt.Sprint(n*4), func(b *testing.B) {
			e, err := NewEnv(appSchema)
			if err != nil {
				b.Fatal(err)
			}
			docs := benchDocs(n)
			b.ReportAllocs()
			for b.Loop() {
				if km, problems := e.Build(docs, nil); km == nil {
					b.Fatal(problems)
				}
			}
		})
	}
}

// The data crossing a process: the leader's menu and a sequencer state,
// streamed through a msgp Writer and Reader that are reused, as a program
// pools them, into a value that is reused.
func BenchmarkCodec(b *testing.B) {
	km, p := benchKeymap(b)
	menu := km.Tree([]myelinkey.Key{leader}, p, nil)
	pending := Pending{Seq: []myelinkey.Key{leader, "k"}, Path: deepPath, Deadline: now}
	for _, tc := range []struct {
		name    string
		in, out codec
	}{
		{"menu", &menu, &Menu{}},
		{"pending", &pending, &Pending{}},
	} {
		var buf bytes.Buffer
		w := msgp.NewWriter(&buf)
		if err := tc.in.EncodeMsg(w); err != nil || w.Flush() != nil {
			b.Fatal(err)
		}
		data := buf.Bytes()
		b.Run(tc.name+"/encode", func(b *testing.B) {
			w := msgp.NewWriter(io.Discard)
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for b.Loop() {
				if err := tc.in.EncodeMsg(w); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(tc.name+"/decode", func(b *testing.B) {
			src := bytes.NewReader(data)
			r := msgp.NewReader(src)
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for b.Loop() {
				src.Reset(data)
				r.Reset(src)
				if err := tc.out.DecodeMsg(r); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
