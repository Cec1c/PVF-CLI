package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"pvfcli/internal/pvf"
	"strings"
	"testing"
)

func TestTokenPatchNewDescriptionPreservesOtherEntries(t *testing.T) {
	a, err := pvf.Open("testdata/protected_nkpi.bin")
	if err != nil {
		t.Fatal(err)
	}
	path := "dir/a.stk"
	raw, err := a.ReadRaw(path)
	if err != nil {
		t.Fatal(err)
	}
	f, _ := a.FindFile(path)
	doc, err := a.InspectTokens(f, raw)
	if err != nil {
		t.Fatal(err)
	}
	var tokens []pvf.EditToken
	for _, tok := range doc.Tokens {
		v := tok.Payload
		tokens = append(tokens, pvf.EditToken{Type: tok.Type, Payload: &v})
	}
	tag, description := "[explain]", "不要滥用！！！\n原说明保留"
	bits := uint32(0x3f99999a)
	unknown := uint32(0xfedcba98)
	tokens = append(tokens, pvf.EditToken{Type: 3, Text: &tag}, pvf.EditToken{Type: 6, Text: &description}, pvf.EditToken{Type: 2, Payload: &bits}, pvf.EditToken{Type: 201, Payload: &unknown})
	plan := patchPlan{Schema: 2, InputSHA256: a.Snapshot().Checksum, Edits: []patchEdit{{Path: path, DataType: 1, BeforeSHA256: hash(raw), Tokens: tokens}}}
	dir := t.TempDir()
	planPath := filepath.Join(dir, "plan.json")
	out := filepath.Join(dir, "patched.pvf")
	write := func() {
		b, _ := json.Marshal(plan)
		if err := os.WriteFile(planPath, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write()
	if code, _, err := invoke(t, "patch", "--pvf", "testdata/protected_nkpi.bin", "--plan", planPath, "--out", out); code != 0 {
		t.Fatal(err)
	}
	if code, _, err := invoke(t, "verify", "--pvf", "testdata/protected_nkpi.bin", "--plan", planPath, "--out", out); code != 0 {
		t.Fatal(err)
	}
	next, err := pvf.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	rawNext, _ := next.ReadRaw(path)
	nextFile, _ := next.FindFile(path)
	edited, err := next.InspectTokens(nextFile, rawNext)
	if err != nil {
		t.Fatal(err)
	}
	if edited.Tokens[len(doc.Tokens)+1].Text != description || edited.Tokens[len(edited.Tokens)-2].Payload != bits || edited.Tokens[len(edited.Tokens)-1].Payload != unknown {
		t.Fatal("description or payload changed")
	}
	for i, original := range doc.Tokens {
		if edited.Tokens[i] != original {
			t.Fatal("original token changed")
		}
	}
	// Rewriting an old shared string must be detected even if raw entries match.
	for _, token := range doc.Tokens {
		if (token.Type == 3 || token.Type == 6) && len(token.Text) > 0 {
			units := len([]rune(token.Text))
			replacement := strings.Repeat("X", units)
			bad, err := next.RepackStrings([]pvf.StringReplacement{{MagicOffset: int(token.Payload), Old: token.Text, New: replacement}})
			if err != nil {
				t.Fatal(err)
			}
			badPath := filepath.Join(dir, "bad.pvf")
			os.WriteFile(badPath, bad, 0600)
			if code, _, _ := invoke(t, "verify", "--pvf", "testdata/protected_nkpi.bin", "--plan", planPath, "--out", badPath); code != 1 {
				t.Fatal("old shared pool mutation accepted")
			}
			break
		}
	}
	plan.Schema = 1
	write()
	if code, _, _ := invoke(t, "patch", "--pvf", "testdata/protected_nkpi.bin", "--plan", planPath, "--out", filepath.Join(dir, "rejected.pvf")); code != 1 {
		t.Fatal("schema 1 accepted token editing")
	}
	plan.Schema = 2
	plan.Edits[0].Tokens[0].Text = &description
	write()
	if code, _, _ := invoke(t, "patch", "--pvf", "testdata/protected_nkpi.bin", "--plan", planPath, "--out", filepath.Join(dir, "rejected.pvf")); code != 1 {
		t.Fatal("ambiguous payload accepted")
	}
}
