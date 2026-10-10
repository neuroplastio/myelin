package myelin

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
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
	row := func(k Key) {
		fmt.Fprintf(&b, "| `%s` | %s | `%s` |\n", FormatKey(k), k, DisplayKey(k))
	}
	shorts := make([]string, 0, len(shortNames))
	for s := range shortNames {
		shorts = append(shorts, s)
	}
	slices.Sort(shorts)
	for _, s := range shorts {
		row(Key(shortNames[s]))
	}
	row("Shift+Tab")
	b.WriteString("| `f1`, `f2`, … | F1, F2, … | `f1`, `f2`, … |\n\n")
	return b.String()
}

// The docs' code is the Examples' code, which go test compiles and runs: a
// fenced block after `<!-- example:NAME -->` is the body of the Example
// function NAME, or the whole of an example file named by its path. The
// gate fails when a block drifts; -update rewrites them.
func TestDocsExamples(t *testing.T) {
	examples := exampleSources(t, "example_test.go", "myelintea/example_test.go")
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
			i := strings.Index(rest, "<!-- example:")
			if i < 0 {
				out.WriteString(rest)
				break
			}
			j := strings.Index(rest[i:], " -->")
			name := rest[i+len("<!-- example:") : i+j]
			code, ok := examples[name]
			if !ok {
				t.Fatalf("%s: no example %q", file, name)
			}
			open := strings.Index(rest[i:], "```go\n")
			end := strings.Index(rest[i+open+len("```go\n"):], "```\n")
			if open < 0 || end < 0 {
				t.Fatalf("%s: example %q has no ```go block after it", file, name)
			}
			start := i + open + len("```go\n")
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
// (myelintea's prefixed "myelintea."), and each whole file by its path.
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
			out[prefix+fn.Name.Name] = strings.Join(lines, "\n") + "\n"
		}
	}
	return out
}
