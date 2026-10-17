package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"pvfcli/internal/pvf"
)

// Raw data is base64 in JSON. Text is never implicitly compiled into tokens.
type patchPlan struct {
	Schema      int         `json:"schema"`
	InputSHA256 string      `json:"input_sha256"`
	Edits       []patchEdit `json:"edits"`
}
type patchEdit struct {
	Path         string          `json:"path"`
	DataType     int             `json:"data_type"`
	BeforeSHA256 string          `json:"before_sha256"`
	AfterSHA256  string          `json:"after_sha256"`
	Raw          []byte          `json:"raw_base64,omitempty"`
	Tokens       []pvf.EditToken `json:"tokens,omitempty"`
}

func hash(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func runPatch(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := fs.String("pvf", "", "original archive")
	planFile := fs.String("plan", "", "strict raw replacement JSON")
	out := fs.String("out", "", "new archive (patch) or existing candidate (verify)")
	max := fs.Int64("max-bytes", pvf.DefaultMaxBytes, "maximum input/output bytes")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *input == "" || *planFile == "" || *out == "" || *max <= 0 || fs.NArg() != 0 {
		fmt.Fprintln(stderr, "patch/verify require --pvf, --plan, --out")
		return 2
	}
	err := applyPatch(ctx, *input, *planFile, *out, *max, args[0] == "verify", stdout)
	if err != nil {
		fmt.Fprintln(stderr, err)
		if ctx.Err() != nil {
			return 130
		}
		return 1
	}
	return 0
}
func applyPatch(ctx context.Context, input, planFile, out string, max int64, verify bool, stdout io.Writer) error {
	f, e := os.Open(planFile)
	if e != nil {
		return e
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 16*1024*1024))
	d.DisallowUnknownFields()
	var plan patchPlan
	if e = d.Decode(&plan); e != nil {
		return e
	}
	var extra any
	if e = d.Decode(&extra); e != io.EOF {
		return fmt.Errorf("trailing plan data")
	}
	if (plan.Schema != 1 && plan.Schema != 2) || len(plan.Edits) == 0 {
		return fmt.Errorf("unsupported or empty plan")
	}
	inInfo, e := os.Stat(input)
	if e != nil {
		return e
	}
	if o, e := os.Stat(out); e == nil {
		if os.SameFile(inInfo, o) {
			return fmt.Errorf("output aliases input")
		}
		if !verify {
			return fmt.Errorf("output already exists")
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	a, e := pvf.LoadArchive(pvf.Options{Path: input, MaxBytes: max})
	if e != nil {
		return e
	}
	if a.Snapshot().Checksum != plan.InputSHA256 {
		return fmt.Errorf("input SHA256 mismatch")
	}
	seen := map[int]bool{}
	replacements := make([]pvf.RawReplacement, 0, len(plan.Edits))
	var tokenEdits []pvf.TokenReplacement
	for _, edit := range plan.Edits {
		entry, ok := a.FindFile(edit.Path)
		if !ok || entry.ArchivePath != edit.Path || seen[entry.Index] || entry.DataType != edit.DataType {
			return fmt.Errorf("invalid, duplicate or mismatched entry %q", edit.Path)
		}
		seen[entry.Index] = true
		b, e := a.ReadRaw(edit.Path)
		if e != nil {
			return e
		}
		if hash(b) != edit.BeforeSHA256 {
			return fmt.Errorf("entry before hash mismatch: %s", edit.Path)
		}
		if len(edit.Tokens) > 0 {
			if plan.Schema != 2 || edit.Raw != nil || edit.AfterSHA256 != "" || edit.DataType != 1 {
				return fmt.Errorf("invalid token edit mode: %s", edit.Path)
			}
			tokenEdits = append(tokenEdits, pvf.TokenReplacement{Path: edit.Path, Tokens: edit.Tokens})
			continue
		}
		if hash(edit.Raw) != edit.AfterSHA256 || edit.BeforeSHA256 == edit.AfterSHA256 {
			return fmt.Errorf("entry hashes mismatch or no change: %s", edit.Path)
		}
		replacements = append(replacements, pvf.RawReplacement{Path: edit.Path, Data: edit.Raw})
	}
	var added []string
	if len(tokenEdits) > 0 {
		var compiled []pvf.RawReplacement
		compiled, added, e = a.CompileTokenReplacements(tokenEdits)
		if e != nil {
			return e
		}
		for _, edit := range compiled {
			before, _ := a.ReadRaw(edit.Path)
			if hash(before) == hash(edit.Data) {
				return fmt.Errorf("token edit has no change: %s", edit.Path)
			}
		}
		replacements = append(replacements, compiled...)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	target := out
	if !verify {
		b, e := a.RepackRawWithStrings(replacements, added)
		if e != nil {
			return e
		}
		if int64(len(b)) > max {
			return fmt.Errorf("output exceeds maximum bytes")
		}
		temp, e := os.CreateTemp(filepath.Dir(out), ".pvf-verified-*.tmp")
		if e != nil {
			return e
		}
		target = temp.Name()
		defer os.Remove(target)
		_, e = temp.Write(b)
		if e == nil {
			e = temp.Sync()
		}
		closeErr := temp.Close()
		if e != nil {
			return e
		}
		if closeErr != nil {
			return closeErr
		}
	}
	next, e := pvf.LoadArchive(pvf.Options{Path: target, MaxBytes: max})
	if e != nil {
		return e
	}
	n, e := a.VerifyRawReplacementWithStrings(ctx, next, replacements, added)
	if e != nil {
		return e
	}
	// Hard-link install is atomic and refuses an existing path, including races.
	// Source and destination are in one directory/on one filesystem.
	if !verify {
		if e = os.Link(target, out); e != nil {
			return e
		}
	}
	digests := map[string]string{}
	for _, edit := range replacements {
		digests[edit.Path] = hash(edit.Data)
	}
	return json.NewEncoder(stdout).Encode(map[string]any{"schema": plan.Schema, "input_sha256": plan.InputSHA256, "output_sha256": next.Snapshot().Checksum, "output_bytes": next.Snapshot().Size, "changed_entries": len(replacements), "entry_sha256": digests, "appended_strings": len(added), "unchanged_entries_verified": n, "verification": "all entry identities; exact old pools, approved appended strings and HASH; encrypted untouched groups; raw touched entries", "out": out})
}
