package myelin

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neuroplastio/myelin/myelinkey"
)

var update = flag.Bool("update", false, "rewrite generated docs")

// docs/keymaps.md's table of named keys is generated from the spelling, and
// the gate fails when it drifts. Rewrite it with go test -run
// TestReferenceKeys -update.
func TestReferenceKeys(t *testing.T) {
	const file = "docs/keymaps.md"
	const begin, end = "<!-- keys:begin -->\n", "<!-- keys:end -->"
	src, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	page := string(src)
	i, j := strings.Index(page, begin), strings.Index(page, end)
	if i < 0 || j < i {
		t.Fatalf("%s has no %q … %q", file, begin, end)
	}
	got := page[:i+len(begin)] + keyTable() + page[j:]
	if *update {
		if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if got != page {
		t.Errorf("%s's key table is stale: run go test -run TestReferenceKeys -update", file)
	}
}

// keyTable is the named keys as the page lists them: each short name, its
// HOTTY name and how menus show it, then the ones spelled by rule.
func keyTable() string {
	var b strings.Builder
	b.WriteString("\n| Write | Key | Menus show |\n|-------|-----|------------|\n")
	row := func(k myelinkey.Key) {
		fmt.Fprintf(&b, "| `%s` | %s | `%s` |\n", myelinkey.Format(k), k, myelinkey.Display(k))
	}
	for _, k := range myelinkey.Named() {
		row(k)
	}
	row("Shift+Tab")
	b.WriteString("| `f1`, `f2`, … | F1, F2, … | `f1`, `f2`, … |\n\n")
	return b.String()
}

// The docs' code is the Examples' code, which go test compiles and runs. A
// fenced block after a marker is generated, and the gate fails when it
// drifts; -update rewrites them:
//   - `<!-- example:NAME -->`: the body of the Example function NAME, or the
//     whole of an example file named by its path;
//   - `<!-- example:NAME#N -->`: step N of that body, the code after its
//     `// N. …` comment and before the next step's;
//   - `<!-- file:PATH -->`: the file at PATH, such as a config an Example
//     reads.
func TestDocsExamples(t *testing.T) {
	examples := exampleSources(t, "example_test.go", "myelinkey/example_test.go")
	pages, err := filepath.Glob("docs/*.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range append(pages, "README.md") {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		page := string(src)
		var out strings.Builder
		rest := page
		for {
			i := strings.Index(rest, "<!-- ")
			if i < 0 {
				out.WriteString(rest)
				break
			}
			j := strings.Index(rest[i:], " -->")
			kind, name, _ := strings.Cut(rest[i+len("<!-- "):i+j], ":")
			var code string
			switch kind {
			case "example":
				c, ok := examples[name]
				if !ok {
					t.Fatalf("%s: no example %q", file, name)
				}
				code = c
			case "file":
				b, err := os.ReadFile(name)
				if err != nil {
					t.Fatalf("%s: %v", file, err)
				}
				code = string(b)
			default:
				out.WriteString(rest[:i+j])
				rest = rest[i+j:]
				continue
			}
			open := strings.Index(rest[i:], "```")
			nl := strings.IndexByte(rest[i+max(open, 0):], '\n')
			if open < 0 || nl < 0 {
				t.Fatalf("%s: %s has no fenced block after it", file, name)
			}
			start := i + open + nl + 1
			end := strings.Index(rest[start:], "```\n")
			if end < 0 {
				t.Fatalf("%s: %s's block doesn't end", file, name)
			}
			out.WriteString(rest[:start])
			out.WriteString(code)
			rest = rest[start+end:]
		}
		if out.String() == page {
			continue
		}
		if *update {
			if err := os.WriteFile(file, []byte(out.String()), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		t.Errorf("%s: an example's code is stale: run go test -run TestDocsExamples -update", file)
	}
}

// exampleSources are each file's Example functions' bodies by name
// (myelinkey's prefixed "myelinkey."), each body's steps as NAME#N, and each
// whole file by its path.
func exampleSources(t *testing.T, files ...string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		out[file] = string(src)
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		prefix := ""
		if dir := filepath.Dir(file); dir != "." {
			prefix = dir + "."
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(fn.Name.Name, "Example") {
				continue
			}
			body := string(src[fset.Position(fn.Body.Lbrace).Offset+1 : fset.Position(fn.Body.Rbrace).Offset])
			lines := strings.Split(strings.Trim(body, "\n"), "\n")
			for i, l := range lines {
				lines[i] = strings.TrimPrefix(l, "\t")
			}
			name := prefix + fn.Name.Name
			out[name] = strings.Join(lines, "\n") + "\n"
			for n, step := range steps(lines) {
				out[fmt.Sprintf("%s#%d", name, n)] = step
			}
		}
	}
	return out
}

// steps splits an Example's lines at its numbered step comments ("// 1.
// Declare …"): step N is the code after its comment, up to the next step.
func steps(lines []string) map[int]string {
	out := map[int]string{}
	n, from := 0, 0
	flush := func(to int) {
		if n == 0 {
			return
		}
		for from < to && strings.HasPrefix(lines[from], "//") {
			from++ // the rest of the step's comment
		}
		out[n] = strings.Trim(strings.Join(lines[from:to], "\n"), "\n") + "\n"
	}
	for i, l := range lines {
		var k int
		if _, err := fmt.Sscanf(l, "// %d. ", &k); err == nil && k == n+1 {
			flush(i)
			n, from = k, i+1
		}
	}
	flush(len(lines))
	return out
}
