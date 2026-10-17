package pvf

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

// Public synthetic fixtures: no game files or strings are used here.
func inspectionFixture(t *testing.T, format ArchiveFormat, unsafe bool) []byte {
	t.Helper()
	a := &Archive{format: format}
	pool := []byte{0}
	put := func(s string) int {
		o := len(pool)
		pool = append(pool, []byte(s)...)
		pool = append(pool, 0)
		return o << 1
	}
	dir := put("dir")
	if unsafe {
		dir = put("../outside")
	}
	label, name, price := put("[name]"), put("测试材料"), put("[price]")
	script := make([]byte, 20)
	for i, v := range []struct {
		k byte
		v int
	}{{3, label}, {6, name}, {3, price}, {0, 42}} {
		script[i*5] = v.k
		binary.LittleEndian.PutUint32(script[i*5+1:], uint32(v.v))
	}
	wide := []byte{}
	for _, r := range utf16.Encode([]rune("中文文本")) {
		wide = binary.LittleEndian.AppendUint16(wide, r)
	}
	items := []fileItem{{nameOffset: put("a.stk"), pathOffset: dir, chunkIndex: 0, dataOffset: 0, dataSize: len(script), dataType: 1},
		{nameOffset: put("b.txt"), pathOffset: dir, chunkIndex: 0, dataOffset: len(script), dataSize: len(wide), dataType: 3},
		{nameOffset: put("c.bin"), pathOffset: dir, chunkIndex: 0, dataOffset: len(script) + len(wide), dataSize: 3, dataType: 9}}
	raw := append(append(script, wide...), 1, 2, 3)
	names := make([]byte, 8)
	if format == FormatProtectedNKPI {
		for _, v := range []struct {
			k   string
			x   uint32
			raw []byte
		}{{"StRa", 0xAA74472E, pool}, {"StRw", 0x9A82F037, []byte{0, 0}}} {
			b, e := encodeProtectedStringBuffer(v.k, v.x, v.raw)
			if e != nil {
				t.Fatal(e)
			}
			names = append(names, b...)
		}
	} else {
		for _, v := range []struct {
			k   string
			x   uint32
			raw []byte
		}{{"sTrA", 0xAA74472E, pool}, {"sTrW", 0x9A82F037, []byte{0, 0}}} {
			b, e := compressZlib(v.raw)
			if e != nil {
				t.Fatal(e)
			}
			decrypt2(v.k, b)
			names = binary.LittleEndian.AppendUint32(names, uint32(len(b))^v.x)
			names = binary.LittleEndian.AppendUint32(names, uint32(len(v.raw)))
			names = append(names, b...)
		}
	}
	body, e := a.encryptChunk(raw)
	if e != nil {
		t.Fatal(e)
	}
	var plain [headerSize]byte
	copy(plain[:], "nkpi")
	h := pvfHeader{plain: plain, fileCount: 3, bodySize: len(body), groupCount: 1, nameSize: len(names)}
	out := a.encodeHeader(h)
	out = append(out, encodeFileTable(items)...)
	out = append(out, names...)
	out = append(out, a.encodeGroups([]groupItem{{compressedSize: len(body), originalSize: len(raw)}})...)
	return append(out, body...)
}

func TestInspectionFormatsAndFixtures(t *testing.T) {
	for _, format := range []ArchiveFormat{FormatNKPI, FormatProtectedNKPI} {
		t.Run(string(format), func(t *testing.T) {
			raw := inspectionFixture(t, format, false)
			a, e := OpenBytes(raw)
			if e != nil {
				t.Fatal(e)
			}
			if a.Format() != format || a.FileCount() != 3 {
				t.Fatal(a.Snapshot())
			}
			count := 0
			e = a.WalkRaw(context.Background(), "DIR", func(f File, b []byte) error {
				count++
				if f.DataType == 9 {
					return nil
				}
				text, e := a.Text(f, b)
				if e != nil {
					return e
				}
				if !strings.Contains(text, "测试材料") && !strings.Contains(text, "中文文本") {
					t.Fatal(text)
				}
				return nil
			})
			if e != nil || count != 3 || a.Snapshot().CachedChunks != 0 || a.Snapshot().CachedTexts != 0 {
				t.Fatal(count, e, a.Snapshot())
			}
			if _, e = a.ReadText("dir/c.bin"); e == nil {
				t.Fatal("opaque data silently accepted as text")
			}
			if _, e = a.FileText(-1); e == nil {
				t.Fatal("bad index accepted")
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if e = a.WalkRaw(ctx, "", func(File, []byte) error { return nil }); !errors.Is(e, context.Canceled) {
				t.Fatal(e)
			}
			if target := os.Getenv("PVFCLI_GENERATE_FIXTURES"); target != "" {
				if e = os.MkdirAll(target, 0700); e != nil {
					t.Fatal(e)
				}
				if e = os.WriteFile(filepath.Join(target, string(format)+".bin"), raw, 0600); e != nil {
					t.Fatal(e)
				}
			}
		})
	}
	if target := os.Getenv("PVFCLI_GENERATE_FIXTURES"); target != "" {
		if e := os.WriteFile(filepath.Join(target, "unsafe.bin"), inspectionFixture(t, FormatProtectedNKPI, true), 0600); e != nil {
			t.Fatal(e)
		}
	}
}

func TestTokenRoundTripPreservesAllBitsAndTail(t *testing.T) {
	raw := []byte{}
	for _, v := range []struct {
		k byte
		v uint32
	}{{0, 0xffffffff}, {2, 0x7fc01234}, {2, 0x80000000}, {5, 123}, {7, 456}, {255, 0xdeadbeef}} {
		raw = append(raw, v.k)
		raw = binary.LittleEndian.AppendUint32(raw, v.v)
	}
	raw = append(raw, 0xaa, 0xbb)
	a := &Archive{}
	d, e := a.InspectTokens(File{DataType: 1}, raw)
	if e != nil {
		t.Fatal(e)
	}
	encoded, _ := json.Marshal(d)
	var copy TokenDocument
	if e = json.Unmarshal(encoded, &copy); e != nil {
		t.Fatal(e)
	}
	got, e := copy.Raw()
	if e != nil || !bytes.Equal(raw, got) {
		t.Fatal(e, got)
	}
	if _, e = a.Text(File{DataType: 1}, raw); e == nil {
		t.Fatal("partial token hidden")
	}
	if _, e = a.Text(File{DataType: 3}, []byte{1}); e == nil {
		t.Fatal("odd UTF16 hidden")
	}
}

func TestTruncatedArchivesNeverPanic(t *testing.T) {
	raw := inspectionFixture(t, FormatProtectedNKPI, false)
	for n := 0; n < len(raw); n++ {
		if _, e := OpenBytes(raw[:n]); e == nil {
			t.Fatalf("accepted truncated archive %d", n)
		}
	}
}
