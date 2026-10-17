package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"pvfcli/internal/pvf"
	"testing"
)

func TestPatchAndVerifyPreservation(t *testing.T) {
	for _, input := range []string{"testdata/nkpi.bin", "testdata/protected_nkpi.bin"} {
		t.Run(input, func(t *testing.T) {
			a, e := pvf.Open(input)
			if e != nil {
				t.Fatal(e)
			}
			original, _ := os.ReadFile(input)
			before, _ := a.ReadRaw("dir/c.bin")
			plan := patchPlan{Schema: 1, InputSHA256: hash(original), Edits: []patchEdit{{Path: "dir/c.bin", DataType: 2, BeforeSHA256: hash(before), AfterSHA256: hash([]byte{4, 5, 6, 7, 8}), Raw: []byte{4, 5, 6, 7, 8}}}}
			entry, _ := a.FindFile("dir/c.bin")
			plan.Edits[0].DataType = entry.DataType
			dir := t.TempDir()
			input = filepath.Join(dir, "input.pvf")
			if e = os.WriteFile(input, original, 0600); e != nil {
				t.Fatal(e)
			}
			p := filepath.Join(dir, "plan.json")
			out := filepath.Join(dir, "new.pvf")
			write := func() {
				b, _ := json.Marshal(plan)
				if e := os.WriteFile(p, b, 0600); e != nil {
					t.Fatal(e)
				}
			}
			write()
			if code, _, err := invoke(t, "patch", "--pvf", input, "--plan", p, "--out", out); code != 0 {
				t.Fatal(err)
			}
			if code, _, err := invoke(t, "verify", "--pvf", input, "--plan", p, "--out", out); code != 0 {
				t.Fatal(err)
			}
			// Introduce an unapproved change in the same compressed group.
			next, e := pvf.Open(out)
			if e != nil {
				t.Fatal(e)
			}
			bad, e := next.RepackRaw([]pvf.RawReplacement{{Path: "dir/a.stk", Data: []byte{0, 0, 0, 0, 0}}})
			if e != nil {
				t.Fatal(e)
			}
			badPath := filepath.Join(dir, "bad.pvf")
			os.WriteFile(badPath, bad, 0600)
			if code, _, _ := invoke(t, "verify", "--pvf", input, "--plan", p, "--out", badPath); code != 1 {
				t.Fatal("unapproved change accepted")
			}
			// Existing output and input aliases must never be overwritten.
			alias := filepath.Join(dir, "alias.pvf")
			if e = os.Link(input, alias); e != nil {
				t.Fatal(e)
			}
			for _, dest := range []string{input, alias, out} {
				if code, _, _ := invoke(t, "patch", "--pvf", input, "--plan", p, "--out", dest); code != 1 {
					t.Fatal("overwrite accepted")
				}
			}
			plan.InputSHA256 = hash(nil)
			write()
			dest := filepath.Join(dir, "rejected.pvf")
			if code, _, _ := invoke(t, "patch", "--pvf", input, "--plan", p, "--out", dest); code != 1 {
				t.Fatal("wrong input hash accepted")
			}
			plan.InputSHA256 = hash(original)
			plan.Edits[0].BeforeSHA256 = hash(nil)
			write()
			if code, _, _ := invoke(t, "patch", "--pvf", input, "--plan", p, "--out", dest); code != 1 {
				t.Fatal("wrong entry hash accepted")
			}
			plan.Edits[0].BeforeSHA256 = hash(before)
			write()
			b, _ := os.ReadFile(p)
			b = append(b[:len(b)-1], []byte(",\"extra_edit\":true}")...)
			os.WriteFile(p, b, 0600)
			if code, _, _ := invoke(t, "patch", "--pvf", input, "--plan", p, "--out", dest); code != 1 {
				t.Fatal("unknown plan field accepted")
			}
			if _, e := os.Stat(dest); !os.IsNotExist(e) {
				t.Fatal("failed patch left output")
			}
			current, _ := os.ReadFile(input)
			if hash(current) != hash(original) {
				t.Fatal("input modified")
			}
		})
	}
}
