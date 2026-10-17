package myelin

import (
	"bytes"
	"errors"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/tinylib/msgp/msgp"
)

// A header that claims more than the limits fails before DecodeMsg
// allocates for it: five bytes from a peer don't cost hundreds of
// megabytes.
func TestDecodeHostileHeader(t *testing.T) {
	arr := []byte{0xdd, 0xff, 0xff, 0xff, 0xff} // an array32 of 2³²−1 elements
	mp := []byte{0xdf, 0xff, 0xff, 0xff, 0xff}  // a map32 of as many entries
	field := func(name string, header []byte) []byte {
		b := msgp.AppendMapHeader(nil, 1)
		return append(msgp.AppendString(b, name), header...)
	}
	for _, tc := range []struct {
		name string
		out  codec
		in   []byte
	}{
		{"schema", &Schema{}, mp},
		{"schema nodes", &Schema{}, field("nodes", arr)},
		{"schema layers", &Schema{}, field("layers", arr)},
		{"schema aliases", &Schema{}, field("aliases", mp)},
		{"node attrs", &NodeType{}, field("attrs", mp)},
		{"doc", &Doc{}, mp},
		{"doc binds", &Doc{}, field("binds", arr)},
		{"doc groups", &Doc{}, field("groups", arr)},
		{"doc drops", &Doc{}, field("drops", arr)},
		{"doc fences", &Doc{}, field("fences", arr)},
		{"doc aliases", &Doc{}, field("aliases", mp)},
		{"bind args", &Bind{}, field("args", arr)},
		{"bind props", &Bind{}, field("props", mp)},
		{"binding seq", &Binding{}, field("seq", arr)},
		{"path", &Path{}, arr},
		{"step attrs", &Step{}, field("attrs", mp)},
		{"pending seq", &Pending{}, field("seq", arr)},
		{"pending path", &Pending{}, field("path", arr)},
		{"menu items", &Menu{}, field("items", arr)},
		{"item items", &Item{}, field("items", arr)},
		{"result dropped", &Result{}, field("dropped", arr)},
		{"result swallowed", &Result{}, field("swallowed", arr)},
	} {
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		err := tc.out.DecodeMsg(msgp.NewReader(bytes.NewReader(tc.in)))
		runtime.ReadMemStats(&after)
		if !errors.Is(err, msgp.ErrLimitExceeded) {
			t.Errorf("%s: %v, want msgp.ErrLimitExceeded", tc.name, err)
		}
		if n := after.TotalAlloc - before.TotalAlloc; n > 1<<20 {
			t.Errorf("%s: allocated %d bytes", tc.name, n)
		}
	}
}

// Every make in the generated codec has a count check in front of it, so a
// type added later can't decode unbounded: msgp sizes a slice or a map from
// its header, whatever the reader's SetMaxElements says.
func TestGeneratedMakesBounded(t *testing.T) {
	src, err := os.ReadFile("data_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(src), "\n")
	sized := regexp.MustCompile(`make\([^,]+, (zb\d+)\)`)
	for i, l := range lines {
		if !strings.Contains(l, "make(") {
			continue
		}
		m := sized.FindStringSubmatch(l)
		if m == nil {
			t.Errorf("data_gen.go:%d: a make not sized by a header: %s", i+1, strings.TrimSpace(l))
			continue
		}
		checked := false
		for _, before := range lines[max(0, i-12):i] {
			checked = checked || strings.Contains(before, "if "+m[1]+" > ")
		}
		if !checked {
			t.Errorf("data_gen.go:%d: %s has no limit in front of it (//msgp:limit or limit=n)", i+1, m[1])
		}
	}
}
