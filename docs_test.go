package myelin

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// The docs are read on GitHub and on the website, which renders README.md,
// docs/*.md and the files docs/docs.yaml names as pages (gov's NEIO-14).
// These tests are the parts of its docs check that myelin's pages can break.

// docPages are the files the website renders: the README, every Markdown
// file under docs/, and the files docs.yaml's pages names.
func docPages(t *testing.T) []string {
	t.Helper()
	pages, err := filepath.Glob("docs/*.md")
	if err != nil {
		t.Fatal(err)
	}
	pages = append([]string{"README.md"}, pages...)
	for _, l := range strings.Split(readFile(t, "docs/docs.yaml"), "\n") {
		if list, ok := strings.CutPrefix(l, "pages:"); ok {
			list = strings.Trim(strings.TrimSpace(list), "[]")
			for _, f := range strings.Split(list, ",") {
				if f = strings.TrimSpace(f); f != "" {
					pages = append(pages, f)
				}
			}
		}
	}
	return pages
}

func readFile(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// splitFront is a page's front matter's lines, and the Markdown after it.
func splitFront(src string) (front []string, body string, ok bool) {
	rest, found := strings.CutPrefix(src, "---\n")
	if !found {
		return nil, src, false
	}
	fm, body, found := strings.Cut(rest, "\n---\n")
	if !found {
		return nil, src, false
	}
	return strings.Split(fm, "\n"), body, true
}

// prose is a page's lines outside fenced code blocks.
func prose(body string) []string {
	var out []string
	fenced := false
	for _, l := range strings.Split(body, "\n") {
		if strings.HasPrefix(l, "```") {
			fenced = !fenced
			continue
		}
		if !fenced {
			out = append(out, l)
		}
	}
	return out
}

// Every page but the README says where it goes in front matter the site
// reads strictly; each page has one title; docs.yaml's description is the
// README's tagline, which the site then shows once.
func TestDocsPages(t *testing.T) {
	keys := []string{"group", "order", "nav", "description", "path", "parent"}
	groups := []string{"Start", "Guides", "Reference", "Concepts"}
	paths := map[string]string{}
	for _, file := range docPages(t) {
		src := readFile(t, file)
		front, body, ok := splitFront(src)
		switch {
		case file == "README.md" && ok:
			t.Errorf("%s: the README has no front matter: it is the overview", file)
		case file != "README.md" && !ok:
			t.Errorf("%s: no front matter: a page says its group", file)
		}
		group, path := "", strings.TrimSuffix(strings.TrimPrefix(file, "docs/"), ".md")
		for _, l := range front {
			k, v, _ := strings.Cut(l, ":")
			v = strings.TrimSpace(v)
			switch {
			case !slices.Contains(keys, k):
				t.Errorf("%s: front matter: %q is none of %v (a library has no man pages)", file, k, keys)
			case k == "group":
				group = v
			case k == "path":
				path = v
			}
		}
		if file != "README.md" {
			if !slices.Contains(groups, group) {
				t.Errorf("%s: group %q is none of %v", file, group, groups)
			}
			if other, ok := paths[path]; ok {
				t.Errorf("%s: its path %q is %s's too", file, path, other)
			}
			paths[path] = file
		}
		titles := 0
		for _, l := range prose(body) {
			if strings.HasPrefix(l, "# ") {
				titles++
			}
		}
		if titles != 1 {
			t.Errorf("%s: %d first-level headings: a page has one, its title", file, titles)
		}
	}

	var description string
	for _, l := range strings.Split(readFile(t, "docs/docs.yaml"), "\n") {
		if v, ok := strings.CutPrefix(l, "description:"); ok {
			description = strings.TrimSpace(v)
		}
	}
	readme := strings.Split(readFile(t, "README.md"), "\n\n")
	if len(readme) < 2 || readme[1] != description {
		t.Errorf("docs.yaml's description %q is not the README's tagline", description)
	}
}

// Code lines in the docs stay within 80 columns, a tab counted as 4, as the
// website and a terminal show them. What an example prints, after its
// "// Output:", is the program's.
func TestDocsWidth(t *testing.T) {
	for _, file := range docPages(t) {
		fenced, output := false, false
		for i, l := range strings.Split(readFile(t, file), "\n") {
			switch {
			case strings.HasPrefix(l, "```"):
				fenced, output = !fenced, false
				continue
			case !fenced:
				continue
			case strings.TrimSpace(l) == "// Output:":
				output = true
			}
			if w := utf8.RuneCountInString(strings.ReplaceAll(l, "\t", "    ")); w > 80 && !output {
				t.Errorf("%s:%d: a code line %d columns wide", file, i+1, w)
			}
		}
	}
}

var (
	mdLink     = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	codeSpan   = regexp.MustCompile("`[^`]*`")
	headingFmt = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
)

// A relative link reaches a file in the repository, and a fragment names a
// heading of the page it links to, by GitHub's ids.
func TestDocsLinks(t *testing.T) {
	ids := map[string][]string{}
	headings := func(file string) []string {
		if h, ok := ids[file]; ok {
			return h
		}
		_, body, _ := splitFront(readFile(t, file))
		seen := map[string]int{}
		var out []string
		for _, l := range prose(body) {
			text, ok := strings.CutPrefix(strings.TrimLeft(l, "#"), " ")
			if !ok || !strings.HasPrefix(l, "#") {
				continue
			}
			id := githubID(text)
			if n := seen[id]; n > 0 {
				seen[id]++
				id = id + "-" + string(rune('0'+n))
			} else {
				seen[id] = 1
			}
			out = append(out, id)
		}
		ids[file] = out
		return out
	}
	for _, file := range docPages(t) {
		_, body, _ := splitFront(readFile(t, file))
		for _, l := range prose(body) {
			for _, m := range mdLink.FindAllStringSubmatch(codeSpan.ReplaceAllString(l, ""), -1) {
				target := m[1]
				if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
					continue
				}
				name, frag, _ := strings.Cut(target, "#")
				to := file
				if name != "" {
					to = filepath.Join(filepath.Dir(file), name)
					if _, err := os.Stat(to); err != nil {
						t.Errorf("%s: %s reaches no file", file, target)
						continue
					}
				}
				if frag != "" && strings.HasSuffix(to, ".md") && !slices.Contains(headings(to), frag) {
					t.Errorf("%s: %s names no heading of %s", file, target, to)
				}
			}
		}
	}
}

// githubID is the id GitHub gives a heading: its text without markup,
// lowercased, spaces as hyphens, and punctuation but - and _ dropped.
func githubID(heading string) string {
	heading = headingFmt.ReplaceAllString(heading, "$1")
	var b strings.Builder
	for _, r := range strings.ToLower(heading) {
		switch {
		case r == ' ':
			b.WriteByte('-')
		case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}
