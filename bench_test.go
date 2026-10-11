package myelin

import (
	"testing"
	"time"

	"github.com/neuroplastio/myelin/myelinkey"
)

// What myelin aims to cost: a key no binding starts with is one map lookup;
// matching a key against a binding with a context takes a few microseconds;
// compiling a context, once at load, some tens.

// benchKeymap is a keymap the size of a program's defaults: a hundred
// bindings, half under contexts.
func benchKeymap(b *testing.B) (*Keymap, Path) {
	e, err := NewEnv(appSchema)
	if err != nil {
		b.Fatal(err)
	}
	var binds []Bind
	for _, c := range "abcdefghijklmnopqrstuvwxy" {
		k := string(c)
		binds = append(binds,
			Bind{Keys: "<leader> " + k, Action: "core." + k},
			Bind{Keys: "<leader> " + k + " " + k, Action: "core.twice." + k},
			Bind{Keys: "ctrl+" + k, Action: "panel." + k, When: "sidebar && panel.ext == 'git' && !input"},
			Bind{Keys: "alt+" + k, Action: "buffer." + k, When: "buffer.tags['format'] == 'gofmt'"},
		)
	}
	km, problems := e.Build([]Doc{{Layer: "core", Binds: binds}}, nil)
	if km == nil {
		b.Fatal(problems)
	}
	p := Path{
		{Name: "window", Attrs: map[string]any{"name": "main"}},
		{Name: "sidebar", Attrs: map[string]any{"side": "left"}},
		{Name: "panel", Attrs: map[string]any{"id": "changes", "ext": "git"}},
	}
	return km, p
}

func BenchmarkFeedUnboundKey(b *testing.B) {
	km, p := benchKeymap(b)
	now := time.Now()
	b.ReportAllocs()
	for b.Loop() {
		Feed(km, Pending{}, "z", p, now)
	}
}

func BenchmarkMatchContextual(b *testing.B) {
	km, p := benchKeymap(b)
	seq := []myelinkey.Key{"Control+k"}
	b.ReportAllocs()
	for b.Loop() {
		if km.Match(seq, p).Exact == nil {
			b.Fatal("no match")
		}
	}
}

func BenchmarkMatchLeaderPrefix(b *testing.B) {
	km, p := benchKeymap(b)
	seq := []myelinkey.Key{"Control+Space", "k"}
	b.ReportAllocs()
	for b.Loop() {
		if !km.Match(seq, p).Prefix {
			b.Fatal("no prefix")
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

func BenchmarkMenuLeader(b *testing.B) {
	km, p := benchKeymap(b)
	seq := []myelinkey.Key{"Control+Space"}
	b.ReportAllocs()
	for b.Loop() {
		km.Menu(seq, p, nil)
	}
}
