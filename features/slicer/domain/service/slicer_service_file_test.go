package service

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/RodolfoBonis/spooliq/features/slicer/domain/entities"
)

func tempGCode(t *testing.T) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "svc-*.gcode")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	if _, err := f.Write([]byte("; test\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("seek: %v", err)
	}
	return f
}

func fileGone(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

func TestAnalyzeFile_RemovesTempOnSuccess(t *testing.T) {
	f := tempGCode(t)
	name := f.Name()
	s := newTestService(func(io.ReaderAt, int64, string) (*entities.Analysis, error) {
		return okAnalysis(), nil
	})

	if _, err := s.AnalyzeFile(context.Background(), f, "part.gcode"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The worker closes and removes the file; give it a moment to run its defers.
	waitUntil(t, func() bool { return fileGone(name) }, time.Second)
}

func TestAnalyzeFile_RemovesTempAfterTimeout(t *testing.T) {
	f := tempGCode(t)
	name := f.Name()

	proceed := make(chan struct{})
	s := newTestService(func(io.ReaderAt, int64, string) (*entities.Analysis, error) {
		<-proceed // block past the analyze timeout
		return okAnalysis(), nil
	})
	s.analyzeTimeout = 30 * time.Millisecond

	_, err := s.AnalyzeFile(context.Background(), f, "part.gcode")
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	// At this point the handler has returned but the worker still holds the file;
	// it must NOT be removed yet by the caller.
	if fileGone(name) {
		t.Fatal("temp file removed before the worker finished (ownership not transferred)")
	}
	// Let the worker finish; it then owns closing and removing the file.
	close(proceed)
	waitUntil(t, func() bool { return fileGone(name) }, 2*time.Second)
}

func TestAnalyzeFile_RemovesTempOnUnsupported(t *testing.T) {
	f := tempGCode(t)
	name := f.Name()
	s := newTestService(func(io.ReaderAt, int64, string) (*entities.Analysis, error) {
		return okAnalysis(), nil
	})
	if _, err := s.AnalyzeFile(context.Background(), f, "model.stl"); err == nil {
		t.Fatal("expected unsupported error")
	}
	waitUntil(t, func() bool { return fileGone(name) }, time.Second)
}

func waitUntil(t *testing.T, cond func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}
