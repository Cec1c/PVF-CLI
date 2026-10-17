package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"pvfcli/internal/pvf"
)

const help = `pvf info|list|read|search|export|patch|verify --pvf <file> [options]
  info:   --json
  list:   --prefix <directory> --query <path substring> --limit 1000 --json
  read:   --path <entry> --format text|raw|tokens --out <new file> --json
  search: --prefix <directory> --query <text> [--section name] [--regex] [--ignore-case] --limit 100 --json
  export: --prefix <directory> OR --path <entry> (repeatable), --format text|raw|tokens --out <directory>
  patch:  --plan <strict raw JSON> --out <new PVF>; verifies every unchanged entry before installation
  verify: --plan <strict raw JSON> --out <existing candidate PVF>; verifies the exact approved changes
Common: --max-bytes 536870912. Limits: 0 means unlimited; truncated results are reported.
Export creates new files only. Text is for reading; raw/tokens preserve payload bits.
Exit: 0 success, 1 input/I/O error, 2 usage, 3 no match, 130 cancelled.
`

var errLimit = errors.New("limit reached")
var errNoMatch = errors.New("no matching entries")

type paths []string

func (p *paths) String() string     { return strings.Join(*p, ",") }
func (p *paths) Set(v string) error { *p = append(*p, v); return nil }

type result struct {
	Schema             int  `json:"schema"`
	Results            any  `json:"results"`
	Truncated          bool `json:"truncated"`
	SkippedUnsupported int  `json:"skipped_unsupported,omitempty"`
}

type match struct {
	Path  string   `json:"path"`
	Lines []string `json:"lines"`
}

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprint(stdout, help)
		return 0
	}
	cmd := args[0]
	if cmd == "patch" || cmd == "verify" {
		return runPatch(ctx, args, stdout, stderr)
	}
	switch cmd {
	case "info", "list", "read", "search", "export":
	default:
		fmt.Fprintln(stderr, "unknown command:", cmd)
		return 2
	}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := fs.String("pvf", "", "input PVF")
	prefix := fs.String("prefix", "", "directory prefix")
	query := fs.String("query", "", "literal substring or regex")
	section := fs.String("section", "", "search within one script section, e.g. name")
	format := fs.String("format", "text", "text, raw, or tokens")
	out := fs.String("out", "", "new output file or export directory")
	jsonOut := fs.Bool("json", false, "JSON result")
	regex := fs.Bool("regex", false, "query is a regular expression")
	fold := fs.Bool("ignore-case", false, "ignore query case")
	max := fs.Int64("max-bytes", pvf.DefaultMaxBytes, "maximum archive bytes")
	defaultLimit := 1000
	if cmd == "search" {
		defaultLimit = 100
	}
	limit := fs.Int("limit", defaultLimit, "maximum matches (0 unlimited)")
	var selected paths
	fs.Var(&selected, "path", "entry path; export allows repetition")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, help)
			return 0
		}
		return 2
	}
	usage := func(s string) int { fmt.Fprintln(stderr, s); return 2 }
	if *input == "" || fs.NArg() != 0 || *limit < 0 || *max <= 0 {
		return usage("--pvf is required; no positional arguments; limits must be nonnegative")
	}
	if *format != "text" && *format != "raw" && *format != "tokens" {
		return usage("--format must be text, raw, or tokens")
	}
	if cmd == "read" && len(selected) != 1 {
		return usage("read requires exactly one --path")
	}
	if cmd == "search" && *query == "" {
		return usage("search requires --query")
	}
	if cmd == "export" && (*out == "" || (len(selected) == 0) == (*prefix == "")) {
		return usage("export requires --out and either --prefix or --path")
	}
	if cmd == "read" && *format == "raw" && *jsonOut {
		return usage("raw cannot be combined with --json")
	}
	var expression *regexp.Regexp
	if *regex {
		q := *query
		if *fold {
			q = "(?i)" + q
		}
		var err error
		expression, err = regexp.Compile(q)
		if err != nil {
			return usage("invalid query regex: " + err.Error())
		}
	}
	matches := func(text string) bool {
		if expression != nil {
			return expression.MatchString(text)
		}
		q := *query
		if *fold {
			text = strings.ToLower(text)
			q = strings.ToLower(q)
		}
		return strings.Contains(text, q)
	}
	archive, err := pvf.LoadArchive(pvf.Options{Path: *input, MaxBytes: *max})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if ctx.Err() != nil {
		return 130
	}
	var data []byte
	var value any
	truncated := false
	switch cmd {
	case "info":
		value = archive.Snapshot()
	case "list":
		files := archive.Files()
		rows := make([]pvf.File, 0)
		for _, f := range files {
			if ctx.Err() != nil {
				err = ctx.Err()
				break
			}
			if !within(f.ArchivePath, *prefix) || !matches(f.ArchivePath) {
				continue
			}
			rows = append(rows, f)
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].ArchivePath < rows[j].ArchivePath })
		if *limit > 0 && len(rows) > *limit {
			truncated = true
			rows = rows[:*limit]
		}
		value = result{Schema: 1, Results: rows, Truncated: truncated}
		if len(rows) == 0 {
			err = errNoMatch
		}
		if !*jsonOut {
			var b strings.Builder
			for _, f := range rows {
				fmt.Fprintf(&b, "%s\t%d\t%d\n", f.ArchivePath, f.DataType, f.Size)
			}
			data = []byte(b.String())
		}
	case "read":
		f, ok := archive.FindFile(selected[0])
		if !ok {
			err = errNoMatch
			break
		}
		var raw []byte
		raw, err = archive.ReadRaw(f.ArchivePath)
		if err != nil {
			break
		}
		data, err = render(archive, f, raw, *format)
		if *jsonOut && *format == "text" {
			value = map[string]any{"schema": 1, "path": f.ArchivePath, "data_type": f.DataType, "text": string(data)}
			data = nil
		}
	case "search":
		rows := make([]match, 0)
		skipped := 0
		err = archive.WalkRaw(ctx, *prefix, func(f pvf.File, raw []byte) error {
			if f.DataType != 1 && f.DataType != 3 {
				skipped++
				return nil
			}
			text, e := archive.Text(f, raw)
			if e != nil {
				return e
			}
			if *section != "" {
				text = sectionText(text, *section)
			}
			if !matches(text) {
				return nil
			}
			if *limit > 0 && len(rows) >= *limit {
				truncated = true
				return errLimit
			}
			row := match{Path: f.ArchivePath, Lines: make([]string, 0)}
			for _, line := range strings.Split(text, "\n") {
				if matches(line) {
					row.Lines = append(row.Lines, strings.TrimSpace(line))
				}
			}
			rows = append(rows, row)
			return nil
		})
		if errors.Is(err, errLimit) {
			err = nil
		}
		value = result{Schema: 1, Results: rows, Truncated: truncated, SkippedUnsupported: skipped}
		if err == nil && len(rows) == 0 {
			err = errNoMatch
		}
		if !*jsonOut {
			var b strings.Builder
			for _, row := range rows {
				fmt.Fprintf(&b, "%s\n  %s\n", row.Path, strings.Join(row.Lines, "\n  "))
			}
			data = []byte(b.String())
		}
	case "export":
		value, err = export(ctx, archive, selected, *prefix, *format, *out, *limit)
		out = new(string) // export already wrote its individual files
	}
	if err != nil && !errors.Is(err, errNoMatch) {
		fmt.Fprintln(stderr, err)
		if errors.Is(err, context.Canceled) {
			return 130
		}
		return 1
	}
	if data == nil && value != nil {
		data, _ = json.MarshalIndent(value, "", "  ")
		data = append(data, '\n')
	}
	if *out != "" {
		if e := writeNew(*out, data); e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
	} else {
		if _, e := stdout.Write(data); e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
	}
	if truncated {
		fmt.Fprintln(stderr, "results truncated; increase --limit or use 0")
	}
	if errors.Is(err, errNoMatch) {
		return 3
	}
	return 0
}

func within(name, prefix string) bool {
	name = strings.ToLower(strings.ReplaceAll(name, "\\", "/"))
	prefix = strings.ToLower(strings.Trim(strings.ReplaceAll(prefix, "\\", "/"), "/"))
	return prefix == "" || name == prefix || strings.HasPrefix(name, prefix+"/")
}

func sectionText(text, section string) string {
	label := "\n[" + strings.Trim(section, "[]") + "]\n"
	_, body, ok := strings.Cut("\n"+text, label)
	if !ok {
		return ""
	}
	if end := strings.Index(body, "\n["); end >= 0 {
		body = body[:end]
	}
	return body
}

func render(a *pvf.Archive, f pvf.File, raw []byte, format string) ([]byte, error) {
	switch format {
	case "raw":
		return raw, nil
	case "text":
		text, err := a.Text(f, raw)
		return []byte(text), err
	case "tokens":
		d, err := a.InspectTokens(f, raw)
		if err != nil {
			return nil, err
		}
		b, err := json.MarshalIndent(d, "", "  ")
		return append(b, '\n'), err
	}
	return nil, fmt.Errorf("unsupported format")
}

func writeNew(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func safePath(name string) (string, error) {
	if strings.ContainsAny(name, "\\:\x00") || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("unsafe export path %q", name)
	}
	for _, p := range strings.Split(name, "/") {
		if p == "" || p == "." || p == ".." {
			return "", fmt.Errorf("unsafe export path %q", name)
		}
	}
	local := filepath.FromSlash(name)
	if !filepath.IsLocal(local) {
		return "", fmt.Errorf("unsafe export path %q", name)
	}
	return local, nil
}

func safeEntryPath(f pvf.File) (string, error) {
	// The reader normalizes ArchivePath; validate the original table strings too
	// so normalization cannot hide a drive, absolute path, or '..' component.
	name := f.Name
	if f.Path != "" {
		name = f.Path + "/" + name
	}
	if _, err := safePath(name); err != nil {
		return "", err
	}
	return safePath(f.ArchivePath)
}

func export(ctx context.Context, a *pvf.Archive, selected []string, prefix, format, out string, limit int) (any, error) {
	files := make(map[string]pvf.File)
	if len(selected) > 0 {
		for _, p := range selected {
			f, ok := a.FindFile(p)
			if !ok {
				return nil, fmt.Errorf("entry not found: %s", p)
			}
			files[f.ArchivePath] = f
		}
	} else {
		for _, f := range a.Files() {
			if within(f.ArchivePath, prefix) {
				files[f.ArchivePath] = f
			}
		}
	}
	if len(files) == 0 {
		return result{Schema: 1, Results: []string{}}, errNoMatch
	}
	if limit > 0 && len(files) > limit {
		return nil, fmt.Errorf("export has %d files, exceeds --limit %d; no files written", len(files), limit)
	}
	for name, f := range files {
		if _, err := safeEntryPath(f); err != nil {
			return nil, err
		}
		if format == "text" && f.DataType != 1 && f.DataType != 3 {
			return nil, fmt.Errorf("%s: unsupported text data type %d", name, f.DataType)
		}
		if format == "tokens" && f.DataType != 1 {
			return nil, fmt.Errorf("%s: tokens require data type 1", name)
		}
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(out)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	for name := range files {
		p, _ := safePath(name)
		if format == "tokens" {
			p += ".tokens.json"
		}
		if _, err := root.Lstat(p); !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("output exists or is inaccessible: %s", p)
		}
	}
	written := make([]string, 0, len(files))
	write := func(f pvf.File, raw []byte) error {
		if _, ok := files[f.ArchivePath]; !ok {
			return nil
		}
		data, err := render(a, f, raw, format)
		if err != nil {
			return err
		}
		p, _ := safePath(f.ArchivePath)
		if format == "tokens" {
			p += ".tokens.json"
		}
		if err := root.MkdirAll(filepath.Dir(p), 0700); err != nil {
			return err
		}
		file, err := root.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, err = file.Write(data)
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		written = append(written, filepath.ToSlash(p))
		return nil
	}
	if len(selected) > 0 {
		for _, name := range selected {
			if err = ctx.Err(); err != nil {
				break
			}
			f, _ := a.FindFile(name)
			if _, ok := files[f.ArchivePath]; !ok {
				continue
			}
			var raw []byte
			raw, err = a.ReadRaw(f.ArchivePath)
			if err == nil {
				err = write(f, raw)
			}
			if err != nil {
				break
			}
			delete(files, f.ArchivePath)
		}
	} else {
		err = a.WalkRaw(ctx, prefix, write)
	}
	if err != nil {
		return nil, fmt.Errorf("export stopped after %d files: %w", len(written), err)
	}
	sort.Strings(written)
	return result{Schema: 1, Results: written}, nil
}
