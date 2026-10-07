package services

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPutObjectSizing(t *testing.T) {
	t.Run("bytes reader uses exact size", func(t *testing.T) {
		r := bytes.NewReader(make([]byte, 27800))
		size, opts := putObjectSizing(r)
		if size != 27800 || opts.PartSize != 0 {
			t.Fatalf("got size=%d part=%d", size, opts.PartSize)
		}
	})
	t.Run("partially read reader reports remaining bytes", func(t *testing.T) {
		r := strings.NewReader("hello world")
		_, _ = r.Read(make([]byte, 6))
		if size, _ := putObjectSizing(r); size != 5 {
			t.Fatalf("got %d, want 5", size)
		}
	})
	t.Run("seeker keeps position", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "x")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f.WriteString("0123456789")
		_, _ = f.Seek(2, io.SeekStart)
		size, _ := putObjectSizing(f)
		pos, _ := f.Seek(0, io.SeekCurrent)
		if size != 8 || pos != 2 {
			t.Fatalf("size=%d pos=%d", size, pos)
		}
	})
	t.Run("unknown length bounds the part size", func(t *testing.T) {
		size, opts := putObjectSizing(io.MultiReader(strings.NewReader("a")))
		if size != -1 || opts.PartSize != uploadPartSize {
			t.Fatalf("size=%d part=%d", size, opts.PartSize)
		}
	})
}
