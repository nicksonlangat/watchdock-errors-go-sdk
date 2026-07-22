package watchdock

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSourceContext(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	content := "package sample\n" + // line 1
		"\n" + // line 2
		"func boom() {\n" + // line 3
		"\tx := 1\n" + // line 4
		"\tpanic(\"boom\")\n" + // line 5 (failing line)
		"\treturn x\n" + // line 6
		"}\n" // line 7
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp source: %v", err)
	}

	cache := map[string][]string{}
	ctx, pre, post := sourceContext(file, 5, cache)

	if ctx != `panic("boom")` {
		t.Errorf("context line = %q, want %q", ctx, `panic("boom")`)
	}
	if len(pre) != contextLines || len(post) != contextLines {
		t.Fatalf("pre/post lengths = %d/%d, want %d/%d", len(pre), len(post), contextLines, contextLines)
	}
	if pre[1] != "\tx := 1" {
		t.Errorf("pre[1] = %q, want %q", pre[1], "\tx := 1")
	}
	if post[0] != "\treturn x" {
		t.Errorf("post[0] = %q, want %q", post[0], "\treturn x")
	}

	// A second lookup for the same file must hit the cache, not re-read disk.
	if _, ok := cache[file]; !ok {
		t.Errorf("expected file to be cached after first read")
	}
}

func TestSourceContextMissingFile(t *testing.T) {
	cache := map[string][]string{}
	ctx, pre, post := sourceContext("/no/such/file.go", 10, cache)
	if ctx != "" || pre != nil || post != nil {
		t.Errorf("missing file should yield empty context, got %q / %v / %v", ctx, pre, post)
	}
	// The miss should be remembered so a repeat lookup doesn't stat again.
	if lines, ok := cache["/no/such/file.go"]; !ok || lines != nil {
		t.Errorf("expected miss to be cached as nil, got ok=%v lines=%v", ok, lines)
	}
}
