package myelin

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/tinylib/msgp/msgp"

	"github.com/neuroplastio/myelin/myelinkey"
)

// codec is what every data type has: msgp's generated streaming codec.
type codec interface {
	msgp.Encodable
	msgp.Decodable
}

// roundTrip writes v with an encoder and reads it into out with a decoder,
// as a program sends myelin's data over a connection, and returns the bytes.
func roundTrip(t *testing.T, v, out codec) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := msgp.NewWriter(&buf)
	if err := v.EncodeMsg(w); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	wire := bytes.Clone(buf.Bytes())
	r := msgp.NewReader(&buf)
	r.SetMaxRecursionDepth(64)
	if err := out.DecodeMsg(r); err != nil {
		t.Fatal(err)
	}
	return wire
}

func TestRoundTrip(t *testing.T) {
	deadline := time.Date(2026, 10, 10, 21, 30, 0, 123456789, time.UTC)
	bind := Bind{Keys: "<leader> p n", Action: "pane.new", Args: []string{"right"}, Label: "new pane",
		When: "pane && !pane.zoomed", Source: "config.kdl:12", Props: map[string]string{"steal": "true"}}
	binding := Binding{Bind: bind, Seq: []myelinkey.Key{"ctrl+b", "p", "n"}, Layer: "user"}
	for _, tc := range []struct {
		name    string
		in, out codec
	}{
		{"schema", &Schema{
			Nodes: []NodeType{
				{Name: "workspace"},
				{Name: "pane", Attrs: map[string]Type{"zoomed": "bool", "cols": "int", "env": "map<string,string>"}},
				{Name: "input", Text: true},
			},
			Layers:  []string{"plugin", "core", "user"},
			Aliases: map[string]string{"leader": "ctrl+b", "window": "ctrl+w"}, StrictKeys: true, ShiftedCapitals: true,
			Timeout: 5 * time.Second,
		}, &Schema{}},
		{"doc", &Doc{Layer: "user", Aliases: map[string]string{"leader": "ctrl+a"}, Drops: []string{"core"}, Source: "config.kdl:3",
			Binds:  []Bind{bind, {Keys: "g g", Unbind: true}},
			Groups: []Group{{Keys: "<leader> p", Label: "+pane", When: "pane", Source: "config.kdl:9"}},
			Fences: []Fence{{When: `workspace.mode == "locked"`, Shadow: true, Swallow: true, Source: "config.kdl:20"}},
		}, &Doc{}},
		{"path", &Path{{Name: "workspace"}, {Name: "pane", Attrs: map[string]any{"zoomed": true, "title": "logs"}}}, &Path{}},
		{"match", &Match{Exact: &binding, Prefix: true}, &Match{}},
		{"candidate", &Candidate{Binding: binding, Holds: true, Depth: 2, Wins: true, Fenced: true}, &Candidate{}},
		{"problem", &Problem{Source: "config.kdl:12", Text: "unknown action", Warning: true}, &Problem{}},
		{"menu", &Menu{Seq: []myelinkey.Key{"ctrl+b"}, Label: "+leader", Items: []Item{
			{Key: "p", Label: "+pane", Group: true, Enabled: true, Items: []Item{
				{Key: "n", Binding: &binding, Enabled: true},
				{Key: "x", Label: "close", Binding: &Binding{Bind: Bind{Keys: "<leader> p x", Action: "pane.close"}, Seq: []myelinkey.Key{"ctrl+b", "p", "x"}, Layer: "core"}},
			}},
		}}, &Menu{}},
		{"pending", &Pending{Seq: []myelinkey.Key{"g"}, Path: Path{{Name: "list"}}, Deadline: deadline}, &Pending{}},
		{"result", &Result{Run: &binding, Deadline: deadline, Dropped: []myelinkey.Key{"g", "x"}, Swallowed: []myelinkey.Key{"q"}}, &Result{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			roundTrip(t, tc.in, tc.out)
			if !reflect.DeepEqual(tc.in, tc.out) {
				t.Errorf("round trip:\n got %#v\nwant %#v", tc.out, tc.in)
			}
		})
	}
}

// Args absent means no values, and [""] means run with an empty one.
func TestArgsAbsentAndEmpty(t *testing.T) {
	var none Bind
	roundTrip(t, &Bind{Keys: "x", Action: "palette"}, &none)
	if none.Args != nil {
		t.Errorf("absent args came back %q", none.Args)
	}
	var empty Bind
	roundTrip(t, &Bind{Keys: "x", Action: "palette", Args: []string{""}}, &empty)
	if len(empty.Args) != 1 || empty.Args[0] != "" {
		t.Errorf(`[""] came back %q`, empty.Args)
	}
}

// A path's attribute values keep their types: msgpack keeps integers and
// floats apart, so an int comes back an integer (int64, CEL's int) and a
// float a float, even a whole one.
func TestPathAttrTypes(t *testing.T) {
	in := Path{{Name: "pane", Attrs: map[string]any{
		"cols": 80, "big": int64(1) << 40, "neg": -3, "ratio": 2.0, "zoomed": false, "title": "logs",
		"env":  map[string]any{"TERM": "xterm"},
		"tags": []any{"a", "b"},
	}}}
	var out Path
	roundTrip(t, &in, &out)
	got := out[0].Attrs
	for k, want := range map[string]any{
		"cols": int64(80), "big": int64(1) << 40, "neg": int64(-3), "ratio": 2.0, "zoomed": false, "title": "logs",
		"env":  map[string]any{"TERM": "xterm"},
		"tags": []any{"a", "b"},
	} {
		if !reflect.DeepEqual(got[k], want) {
			t.Errorf("%s came back %#v (%T), want %#v (%T)", k, got[k], got[k], want, want)
		}
	}
}

// A deadline is a msgpack timestamp, extension −1, not msgp's own
// extension 5, and comes back in UTC.
func TestDeadlineIsATimestamp(t *testing.T) {
	local := time.FixedZone("CEST", 2*60*60)
	deadline := time.Date(2026, 10, 10, 23, 30, 5, 500, local)
	var out Pending
	wire := roundTrip(t, &Pending{Seq: []myelinkey.Key{"g"}, Deadline: deadline}, &out)
	if !out.Deadline.Equal(deadline) || out.Deadline.Location() != time.UTC {
		t.Errorf("deadline came back %v, want %v in UTC", out.Deadline, deadline)
	}
	// The value after the "deadline" key is an extension of type −1.
	i := bytes.Index(wire, append([]byte{0xa8}, "deadline"...))
	if i < 0 {
		t.Fatalf("no deadline key in % x", wire)
	}
	ext := wire[i+9:]
	var typ byte
	switch ext[0] {
	case 0xd6, 0xd7: // fixext 4, 8
		typ = ext[1]
	case 0xc7: // ext 8: length, type
		typ = ext[2]
	default:
		t.Fatalf("deadline is % x, not an extension", ext[:min(len(ext), 4)])
	}
	if int8(typ) != -1 {
		t.Errorf("deadline is extension %d, want -1", int8(typ))
	}
	// A zero deadline is left out.
	wire = roundTrip(t, &Pending{Seq: []myelinkey.Key{"g"}}, &Pending{})
	if bytes.Contains(wire, []byte("deadline")) {
		t.Errorf("a zero deadline is written: % x", wire)
	}
}

func TestMount(t *testing.T) {
	d := Doc{Layer: "default", Drops: []string{"x"},
		Binds: []Bind{
			{Keys: "j", Action: "down", Args: []string{""}, Props: map[string]string{"p": "1"}},
			{Keys: "q", Action: "quit", When: "a || b"},
		},
		Groups: []Group{{Keys: "g", Label: "+go"}, {Keys: "z", Label: "+fold", When: "c"}},
		Fences: []Fence{{Swallow: true}, {When: "d", Shadow: true}},
	}
	m := Mount(d, "logs || pager")
	if got := m.Binds[0].When; got != "logs || pager" {
		t.Errorf("a bind with no context mounts as %q", got)
	}
	if got := m.Binds[1].When; got != "(logs || pager) && (a || b)" {
		t.Errorf("a bind with a context mounts as %q", got)
	}
	if got := m.Groups[0].When; got != "logs || pager" {
		t.Errorf("a group with no context mounts as %q", got)
	}
	if got := m.Groups[1].When; got != "(logs || pager) && (c)" {
		t.Errorf("a group with a context mounts as %q", got)
	}
	if got := m.Fences[0].When + " | " + m.Fences[1].When; got != "logs || pager | (logs || pager) && (d)" {
		t.Errorf("the fences mount as %q", got)
	}
	// A copy: the original is untouched by the mount and by edits to it.
	m.Binds[0].Args[0], m.Binds[0].Props["p"], m.Drops[0] = "changed", "changed", "changed"
	if d.Binds[0].When != "" || d.Groups[1].When != "c" || d.Fences[1].When != "d" || d.Binds[0].Args[0] != "" || d.Binds[0].Props["p"] != "1" || d.Drops[0] != "x" {
		t.Errorf("Mount changed its document: %#v", d)
	}
	if m.Binds[0].Args == nil || len(Mount(Doc{Binds: []Bind{{Keys: "x"}}}, "y").Binds[0].Args) != 0 {
		t.Errorf("Mount lost the difference between no args and [\"\"]")
	}
	if !reflect.DeepEqual(Mount(d, ""), d) {
		t.Errorf("an empty when changed the document")
	}
}
