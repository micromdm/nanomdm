package diskv

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/peterbourgon/diskv/v3"
)

func TestSplit2X2Transform(t *testing.T) {
	for _, tc := range []struct {
		key  string
		want []string
	}{
		{"00008030-001A2B3C4D5E6F70.auth", []string{"00", "00"}},
		{"ABCD1234-5678-90EF-ABCD-1234567890EF", []string{"AB", "CD"}},
		{"a", []string{"00", "0a"}},
		{"....repro.auth", []string{"__", "__"}},
		{"..ab.auth", []string{"__", "ab"}},
		{"ab...auth", []string{"ab", "__"}},
		{"..", []string{"00", "__"}},
		{".a.b", []string{".a", ".b"}},
	} {
		t.Run(tc.key, func(t *testing.T) {
			got := Split2X2Transform(tc.key)
			if len(got) != len(tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("got %q, want %q", got, tc.want)
					break
				}
			}
		})
	}
}

func TestSplit2X2TransformDotDotKey(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "dbkv")
	d := diskv.New(diskv.Options{
		BasePath:  filepath.Join(root, "users"),
		Transform: Split2X2Transform,
	})

	key := "....repro.auth"
	value := []byte("hello")

	if err := d.Write(key, value); err != nil {
		t.Fatal(err)
	}
	got, err := d.Read(key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, value) {
		t.Errorf("got %q, want %q", got, value)
	}

	err = filepath.Walk(tmp, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			t.Errorf("file written outside storage root %s: %s", root, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
