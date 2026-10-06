// Package service provides the bounded slicer analysis service used by both the
// slicer HTTP endpoint and the model3d feature. It wraps the pure parsers with a
// wall-clock timeout, panic recovery and a process-wide concurrency gate (the
// same safety pattern as the thumbnail service), and enriches an analysis with
// per-slot filament suggestions.
package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/parser"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/suggest"
)

// Analysis safety limits. A malformed file could make a parser spin or allocate;
// these bounds keep one bad upload from pinning the process.
const (
	defaultAnalyzeTimeout = 20 * time.Second
	defaultAcquireTimeout = 2 * time.Second
	maxConcurrentAnalyses = 4
)

// analyzeSem is the process-wide gate bounding concurrent analyses.
var analyzeSem = make(chan struct{}, maxConcurrentAnalyses)

// Service analyzes sliced files and enriches results with filament suggestions.
type Service interface {
	// Analyze parses a sliced file into an Analysis WITHOUT suggestions. It is
	// bounded by a timeout, a concurrency gate and panic recovery. It returns the
	// parser sentinel errors (parser.ErrNotSliced, parser.ErrCorruptFile,
	// parser.ErrUnsupported) unwrapped so callers can map them to HTTP responses.
	Analyze(ctx context.Context, ra io.ReaderAt, size int64, filename string) (*entities.Analysis, error)

	// Suggest fills each filament slot's Suggestion in place, loading the
	// organization's catalog once. A nil analysis is a no-op.
	Suggest(ctx context.Context, organizationID string, analysis *entities.Analysis) error
}

type slicerService struct {
	catalog        repositories.FilamentCatalogRepository
	logger         logger.Logger
	analyzeTimeout time.Duration
	acquireTimeout time.Duration
	sem            chan struct{}
}

// NewService builds the slicer analysis service.
func NewService(catalog repositories.FilamentCatalogRepository, log logger.Logger) Service {
	return &slicerService{
		catalog:        catalog,
		logger:         log,
		analyzeTimeout: defaultAnalyzeTimeout,
		acquireTimeout: defaultAcquireTimeout,
		sem:            analyzeSem,
	}
}

type analyzeResult struct {
	analysis *entities.Analysis
	err      error
}

func (s *slicerService) Analyze(ctx context.Context, ra io.ReaderAt, size int64, filename string) (*entities.Analysis, error) {
	if !parser.IsSupported(filename) {
		return nil, parser.ErrUnsupported
	}

	// Acquire a slot; skip queuing behind a stuck analysis.
	select {
	case s.sem <- struct{}{}:
	case <-time.After(s.acquireTimeout):
		s.logger.Warning(ctx, "Slicer analysis skipped: no slot available", nil)
		return nil, fmt.Errorf("slicer: analysis busy")
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	analyzeCtx, cancel := context.WithTimeout(ctx, s.analyzeTimeout)
	defer cancel()

	ch := make(chan analyzeResult, 1)
	go func() {
		defer func() { <-s.sem }()
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error(context.Background(), "Slicer analysis panicked", map[string]interface{}{"panic": fmt.Sprintf("%v", r)})
				ch <- analyzeResult{err: fmt.Errorf("slicer: analysis panicked")}
			}
		}()
		a, err := parser.Analyze(ra, size, filename)
		ch <- analyzeResult{analysis: a, err: err}
	}()

	select {
	case res := <-ch:
		return res.analysis, res.err
	case <-analyzeCtx.Done():
		s.logger.Warning(ctx, "Slicer analysis timed out", map[string]interface{}{"timeout": s.analyzeTimeout.String()})
		return nil, fmt.Errorf("slicer: analysis timed out: %w", analyzeCtx.Err())
	}
}

func (s *slicerService) Suggest(ctx context.Context, organizationID string, analysis *entities.Analysis) error {
	if analysis == nil {
		return nil
	}

	candidates, err := s.catalog.LoadCandidates(ctx, organizationID)
	if err != nil {
		return err
	}

	for pi := range analysis.Plates {
		for fi := range analysis.Plates[pi].Filaments {
			fil := &analysis.Plates[pi].Filaments[fi]
			fil.Suggestion = suggest.Match(fil.ColorHex, fil.Material, candidates)
		}
	}
	return nil
}

// IsNotSliced reports whether err is the "not sliced" sentinel.
func IsNotSliced(err error) bool { return errors.Is(err, parser.ErrNotSliced) }

// IsCorruptFile reports whether err is the "corrupt file" sentinel.
func IsCorruptFile(err error) bool { return errors.Is(err, parser.ErrCorruptFile) }

// IsUnsupported reports whether err is the "unsupported format" sentinel.
func IsUnsupported(err error) bool { return errors.Is(err, parser.ErrUnsupported) }
