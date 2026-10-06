// Package parser contains the pure slicer-file parsers (G-code comment metadata
// and sliced 3MF archives). It has NO gin or DB dependencies and is heavily
// unit-tested. Parsers read only the metadata they need and never load an entire
// large file into memory.
package parser

import (
	"bufio"
	"errors"
	"io"
	"math"
	"path/filepath"
	"strings"

	"github.com/RodolfoBonis/spooliq/features/slicer/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/material"
)

// Sentinel errors mapped by callers to specific HTTP responses.
var (
	// ErrNotSliced indicates a valid file that carries no slicing data (e.g. a
	// design-only PrusaSlicer 3MF). Callers map it to 422 file_not_sliced.
	ErrNotSliced = errors.New("slicer: file contains no slicing data")
	// ErrCorruptFile indicates a malformed/corrupt archive. Callers map it to 400
	// invalid_file.
	ErrCorruptFile = errors.New("slicer: corrupt or unreadable file")
	// ErrUnsupported indicates an unsupported extension.
	ErrUnsupported = errors.New("slicer: unsupported file format")
)

// Default filament diameter in millimeters, used when a slicer does not report it.
const defaultDiameterMM = 1.75

// headTailLimit is how many bytes to scan from the head and (separately) the tail
// of a G-code file. Many slicers place summary metadata at the very end, so we
// look at both ends without ever reading the middle.
const headTailLimit int64 = 2 << 20 // 2MB

// maxLineBytes caps a single scanned line. Lines longer than this (e.g. a huge
// embedded thumbnail comment) are skipped rather than buffered unbounded.
const maxLineBytes = 64 * 1024

// Analyze dispatches to the right parser based on the file name/extension.
// It reads only the metadata needed from ra (which must cover [0,size)).
func Analyze(ra io.ReaderAt, size int64, filename string) (*entities.Analysis, error) {
	switch classify(filename) {
	case sourceGCode:
		return ParseGCode(ra, size)
	case source3MF:
		return Parse3MF(ra, size)
	default:
		return nil, ErrUnsupported
	}
}

type sourceKind int

const (
	sourceUnknown sourceKind = iota
	sourceGCode
	source3MF
)

// classify maps a filename to a parser kind. ".gcode.3mf" is a 3MF archive, so it
// must be checked before the plain extension.
func classify(filename string) sourceKind {
	lower := strings.ToLower(filename)
	if strings.HasSuffix(lower, ".gcode.3mf") || strings.HasSuffix(lower, ".3mf") {
		return source3MF
	}
	if filepath.Ext(lower) == ".gcode" {
		return sourceGCode
	}
	return sourceUnknown
}

// IsSupported reports whether a filename is a format the analyzer can parse.
func IsSupported(filename string) bool {
	return classify(filename) != sourceUnknown
}

// scanComments calls fn for each metadata comment line (lines starting with ';')
// found in r. Non-comment lines (actual G-code moves) are ignored. Overly long
// lines are truncated to maxLineBytes. Scanning stops early if fn returns false.
func scanComments(r io.Reader, fn func(line string) bool) {
	reader := bufio.NewReaderSize(r, 32*1024)
	for {
		line, err := readCappedLine(reader)
		if len(line) > 0 {
			trimmed := strings.TrimLeft(line, " \t")
			if strings.HasPrefix(trimmed, ";") {
				if !fn(strings.TrimRight(trimmed, "\r\n")) {
					return
				}
			}
		}
		if err != nil {
			return
		}
	}
}

// readCappedLine reads one line, discarding anything beyond maxLineBytes so a
// pathological line cannot exhaust memory. It returns the (possibly truncated)
// line including its trailing newline when present, and io.EOF at end of input.
func readCappedLine(r *bufio.Reader) (string, error) {
	var b strings.Builder
	for {
		chunk, err := r.ReadString('\n')
		if b.Len() < maxLineBytes {
			remaining := maxLineBytes - b.Len()
			if len(chunk) > remaining {
				b.WriteString(chunk[:remaining])
			} else {
				b.WriteString(chunk)
			}
		}
		// A complete line ends with '\n' (or EOF).
		if err != nil || strings.HasSuffix(chunk, "\n") {
			return b.String(), err
		}
	}
}

// estimateGramsFromLengthMM converts a filament length (mm) to grams using the
// material density and filament diameter. Used for slicers (Cura) that report
// length but not weight.
func estimateGramsFromLengthMM(lengthMM, diameterMM float64, materialName string) float64 {
	if diameterMM <= 0 {
		diameterMM = defaultDiameterMM
	}
	radiusCM := (diameterMM / 2) / 10 // mm -> cm
	lengthCM := lengthMM / 10         // mm -> cm
	volumeCM3 := math.Pi * radiusCM * radiusCM * lengthCM
	return volumeCM3 * material.Density(materialName)
}
