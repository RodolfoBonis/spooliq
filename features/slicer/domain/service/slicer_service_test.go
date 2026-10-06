package service

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/suggest"
)

// emptyCatalog yields no candidates.
type emptyCatalog struct{}

func (emptyCatalog) LoadCandidates(_ context.Context, _ string) ([]suggest.Candidate, error) {
	return nil, nil
}

// newTestService builds a service with shrunk tunables and an injected parse seam
// so the safety layer can be exercised without real files.
func newTestService(parse func(io.ReaderAt, int64, string) (*entities.Analysis, error)) *slicerService {
	return &slicerService{
		catalog:        emptyCatalog{},
		logger:         &logger.NoopLogger{},
		analyzeTimeout: 50 * time.Millisecond,
		acquireTimeout: 50 * time.Millisecond,
		sem:            make(chan struct{}, 1),
		parse:          parse,
	}
}

func okAnalysis() *entities.Analysis {
	return &entities.Analysis{Source: entities.SourceGCode, Warnings: []string{}}
}

func TestService_Timeout(t *testing.T) {
	// A parser that blocks far longer than the analyze timeout.
	parse := func(io.ReaderAt, int64, string) (*entities.Analysis, error) {
		time.Sleep(2 * time.Second)
		return okAnalysis(), nil
	}
	s := newTestService(parse)

	start := time.Now()
	_, err := s.Analyze(context.Background(), nil, 0, "part.gcode")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected a timeout error")
	}
	// Must return promptly (well under the 2s parser sleep).
	if elapsed > 500*time.Millisecond {
		t.Errorf("timeout returned after %v, want ~50ms", elapsed)
	}
}

func TestService_PanicRecovered(t *testing.T) {
	parse := func(io.ReaderAt, int64, string) (*entities.Analysis, error) {
		panic("boom")
	}
	s := newTestService(parse)

	a, err := s.Analyze(context.Background(), nil, 0, "part.gcode")
	if err == nil {
		t.Fatal("expected an error from a panicking parser")
	}
	if a != nil {
		t.Errorf("expected nil analysis, got %+v", a)
	}
}

func TestService_AcquireTimeout(t *testing.T) {
	// Occupy the single slot with a parser that never returns, then a second call
	// must give up at the acquire timeout rather than block.
	release := make(chan struct{})
	parse := func(io.ReaderAt, int64, string) (*entities.Analysis, error) {
		<-release
		return okAnalysis(), nil
	}
	s := newTestService(parse)
	// Give the first analysis a generous analyze timeout so it holds the slot.
	s.analyzeTimeout = 5 * time.Second

	go func() { _, _ = s.Analyze(context.Background(), nil, 0, "a.gcode") }()
	// Wait for the first call to grab the slot.
	time.Sleep(20 * time.Millisecond)

	start := time.Now()
	_, err := s.Analyze(context.Background(), nil, 0, "b.gcode")
	elapsed := time.Since(start)
	close(release)

	if err == nil {
		t.Fatal("expected ErrBusy when no slot is available")
	}
	if elapsed > 400*time.Millisecond {
		t.Errorf("acquire returned after %v, want ~50ms", elapsed)
	}
}

func TestService_Success(t *testing.T) {
	parse := func(io.ReaderAt, int64, string) (*entities.Analysis, error) {
		return okAnalysis(), nil
	}
	s := newTestService(parse)
	a, err := s.Analyze(context.Background(), nil, 0, "part.gcode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a == nil || a.Source != entities.SourceGCode {
		t.Errorf("unexpected analysis: %+v", a)
	}

	// The slot must be released and reusable.
	if _, err := s.Analyze(context.Background(), nil, 0, "part.gcode"); err != nil {
		t.Errorf("slot not released: %v", err)
	}
}

func TestService_UnsupportedShortCircuits(t *testing.T) {
	called := false
	parse := func(io.ReaderAt, int64, string) (*entities.Analysis, error) {
		called = true
		return okAnalysis(), nil
	}
	s := newTestService(parse)
	if _, err := s.Analyze(context.Background(), nil, 0, "model.stl"); err == nil {
		t.Fatal("expected unsupported error")
	}
	if called {
		t.Error("parser should not be called for an unsupported extension")
	}
}
