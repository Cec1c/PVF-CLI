package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"pvfcli/internal/pvf"
	"strings"
	"testing"
)

func invoke(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, err bytes.Buffer
	code := Run(context.Background(), args, &out, &err)
	return code, out.String(), err.String()
}

func TestReadSearchAndJSON(t *testing.T) {
	for _, input := range []string{"testdata/nkpi.bin", "testdata/protected_nkpi.bin"} {
		for _, args := range [][]string{{"info", "--json"}, {"list", "--limit", "1", "--json"}, {"read", "--path", "dir/a.stk", "--json"}, {"read", "--path", "dir/a.stk", "--format", "tokens"}, {"search", "--query", "测试", "--section", "name", "--json"}} {
			code, out, err := invoke(t, append(args, "--pvf", input)...)
			if code != 0 || !json.Valid([]byte(out)) {
				t.Fatalf("%v: %d %s %s", args, code, out, err)
			}
		}
		code, out, err := invoke(t, "read", "--pvf", input, "--path", "dir/c.bin", "--format", "raw")
		if code != 0 || out != "\x01\x02\x03" || err != "" {
			t.Fatal(code, out, err)
		}
		code, _, _ = invoke(t, "read", "--pvf", input, "--path", "dir/c.bin")
		if code != 1 {
			t.Fatal("opaque text", code)
		}
		code, out, _ = invoke(t, "search", "--pvf", input, "--query", "absent", "--json")
		if code != 3 || !json.Valid([]byte(out)) {
			t.Fatal(code, out)
		}
	}
}

func TestExportAndReadCannotClobber(t *testing.T) {
	dir := t.TempDir()
	input := "testdata/protected_nkpi.bin"
	code, _, err := invoke(t, "export", "--pvf", input, "--path", "dir/a.stk", "--path", "dir/b.txt", "--out", dir)
	if code != 0 {
		t.Fatal(err)
	}
	code, _, _ = invoke(t, "export", "--pvf", input, "--path", "dir/a.stk", "--out", dir)
	if code != 1 {
		t.Fatal("overwrite accepted")
	}
	code, _, _ = invoke(t, "read", "--pvf", input, "--path", "dir/a.stk", "--format", "raw", "--out", input)
	if code != 1 {
		t.Fatal("input overwritten")
	}
	code, _, _ = invoke(t, "export", "--pvf", input, "--prefix", "dir", "--format", "raw", "--out", t.TempDir(), "--limit", "1")
	if code != 1 {
		t.Fatal("partial limit export accepted")
	}
	code, _, _ = invoke(t, "export", "--pvf", "testdata/unsafe.bin", "--prefix", "..", "--format", "raw", "--out", t.TempDir())
	if code != 1 {
		t.Fatal("traversal accepted", code)
	}
}

func TestExportRejectsEscapingSymlink(t *testing.T) {
	dir, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "dir")); err != nil {
		t.Skip("symlink privilege unavailable")
	}
	code, _, _ := invoke(t, "export", "--pvf", "testdata/nkpi.bin", "--path", "dir/a.stk", "--out", dir)
	if code != 1 {
		t.Fatal("symlink escaped")
	}
	entries, _ := os.ReadDir(outside)
	if len(entries) != 0 {
		t.Fatal("wrote outside export root")
	}
}

func TestSafePathsAndUsage(t *testing.T) {
	for _, dir := range []string{"/absolute", "C:/drive", "dir/../hidden"} {
		if _, e := safeEntryPath(pvf.File{Path: dir, Name: "a.stk", ArchivePath: "normalized/a.stk"}); e == nil {
			t.Fatal("normalized away unsafe path", dir)
		}
	}
	for _, name := range []string{"../x", "/abs", "a/../b", "C:/x", "a\\b", "a//b", "a:stream", ""} {
		if _, e := safePath(name); e == nil {
			t.Fatal(name)
		}
	}
	for _, args := range [][]string{{"unknown"}, {"read"}, {"search", "--pvf", "testdata/nkpi.bin"}, {"read", "--pvf", "testdata/nkpi.bin", "--path", "dir/a.stk", "--format", "raw", "--json"}} {
		code, _, _ := invoke(t, args...)
		if code != 2 {
			t.Fatal(args, code)
		}
	}
	if got := sectionText("\n[name]\n`test`\n[explain]\n`other`", "name"); strings.Contains(got, "other") || !strings.Contains(got, "test") {
		t.Fatal(got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out, err bytes.Buffer
	if code := Run(ctx, []string{"info", "--pvf", "testdata/nkpi.bin"}, &out, &err); code != 130 {
		t.Fatal(code)
	}
}
