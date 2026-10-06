package parser

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"strings"
	"testing"
)

// Verbatim slice_info.config from a real Bambu Studio 02.08.02.61 export (Bambu
// Lab A1, 4-color AMS + 1 spare): note the non-contiguous filament id 7.
const realBambuSliceInfo = `<?xml version="1.0" encoding="UTF-8"?>
<config>
  <header>
    <header_item key="X-BBL-Client-Type" value="slicer"/>
    <header_item key="X-BBL-Client-Version" value="02.08.02.61"/>
  </header>
  <plate>
    <metadata key="index" value="1"/>
    <metadata key="printer_model_id" value="N2S"/>
    <metadata key="prediction" value="129846"/>
    <metadata key="weight" value="543.83"/>
    <object identify_id="79" name="HULK NO BASE.glb" skipped="false" />
    <filament id="1" tray_info_idx="GFA01" type="PLA" color="#000000" used_m="10.70" used_g="33.99" group_id="0"/>
    <filament id="2" tray_info_idx="GFA00" type="PLA" color="#0ACC38" used_m="113.38" used_g="343.62" group_id="0"/>
    <filament id="3" tray_info_idx="GFA00" type="PLA" color="#5E43B7" used_m="46.54" used_g="141.05" group_id="0"/>
    <filament id="4" tray_info_idx="GFA01" type="PLA" color="#FFFFFF" used_m="5.38" used_g="17.09" group_id="0"/>
    <filament id="7" tray_info_idx="GFA01" type="PLA" color="#DE4343" used_m="2.54" used_g="8.08" group_id="0"/>
  </plate>
</config>`

const bambuModelHead = `<?xml version="1.0" encoding="UTF-8"?>
<model unit="millimeter" xml:lang="en-US" xmlns="http://schemas.microsoft.com/3dmanufacturing/core/2015/02">
 <metadata name="Application">BambuStudio-02.08.02.61</metadata>
 <metadata name="BambuStudio:3mfVersion">1</metadata>
</model>`

type zipEntry struct {
	name string
	data []byte
	// declaredSize, when > 0, writes a raw entry whose header claims this
	// uncompressed size (simulating a huge entry without allocating it).
	declaredSize uint64
}

func buildRawZip(t *testing.T, entries []zipEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		if e.declaredSize > 0 {
			w, err := zw.CreateRaw(&zip.FileHeader{
				Name:               e.name,
				Method:             zip.Store,
				CompressedSize64:   uint64(len(e.data)),
				UncompressedSize64: e.declaredSize,
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write(e.data); err != nil {
				t.Fatal(err)
			}
			continue
		}
		w, err := zw.Create(e.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(e.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// Regression: a real .gcode.3mf carries a >100MB Metadata/plate_1.gcode. The
// archive must not be rejected for the declared size of an entry that is never
// read, since slice_info.config already has the data.
func TestParse3MF_RealBambuWithHugeUnreadGCode(t *testing.T) {
	data := buildRawZip(t, []zipEntry{
		{name: "3D/3dmodel.model", data: []byte(bambuModelHead)},
		{name: sliceInfoPath, data: []byte(realBambuSliceInfo)},
		{name: "Metadata/plate_1.gcode", data: []byte("; tiny"), declaredSize: 123_871_340},
	})

	a, err := Parse3MF(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("Parse3MF: %v", err)
	}
	if a.Slicer.Name != "BambuStudio" || a.Slicer.Version != "02.08.02.61" {
		t.Errorf("slicer = %+v", a.Slicer)
	}
	if len(a.Plates) != 1 {
		t.Fatalf("plates = %d", len(a.Plates))
	}
	p := a.Plates[0]
	if p.PrintTimeSeconds != 129846 {
		t.Errorf("print time = %d", p.PrintTimeSeconds)
	}
	wantSlots := []int{1, 2, 3, 4, 7}
	if len(p.Filaments) != len(wantSlots) {
		t.Fatalf("filaments = %d", len(p.Filaments))
	}
	var total float64
	for i, f := range p.Filaments {
		if f.Slot != wantSlots[i] {
			t.Errorf("slot[%d] = %d, want %d", i, f.Slot, wantSlots[i])
		}
		if f.Material != "PLA" {
			t.Errorf("material[%d] = %q", i, f.Material)
		}
		total += f.Grams
	}
	if total < 543.82 || total > 543.84 {
		t.Errorf("total grams = %.2f, want 543.83", total)
	}
	if p.Filaments[4].ColorHex != "#DE4343" {
		t.Errorf("slot 7 color = %q", p.Filaments[4].ColorHex)
	}
}

// realisticGCode builds G-code with metadata at the head and tail and
// low-compressibility moves in between (real G-code compresses ~3-4x).
func realisticGCode(bodyBytes int) []byte {
	var b strings.Builder
	b.WriteString("; HEADER_BLOCK_START\n; BambuStudio 02.08.02.61\n")
	b.WriteString("; model printing time: 1h 2m 3s; total estimated time: 1h 5m 0s\n")
	b.WriteString("; total filament weight [g] : 12.50,3.25\n; HEADER_BLOCK_END\n")
	x := uint32(12345)
	for b.Len() < bodyBytes {
		x = x*1664525 + 1013904223
		fmt.Fprintf(&b, "G1 X%.3f Y%.3f E%.5f\n", float64(x%25600)/100, float64((x>>8)%25600)/100, float64(x%9973)/1e4)
	}
	b.WriteString("; filament_colour = #FF0000FF;#00AE42FF\n; filament_type = PLA;PETG\n")
	return []byte(b.String())
}

// Without slice_info, the embedded plate G-code is streamed (head/tail only) so
// files larger than the per-entry cap still parse.
func TestParse3MF_StreamsLargeEmbeddedGCode(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a multi-MB G-code")
	}
	gcode := realisticGCode(6 << 20) // > 2 × head/tail window
	data := buildRawZip(t, []zipEntry{{name: "Metadata/plate_1.gcode", data: gcode}})

	a, err := Parse3MF(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("Parse3MF: %v", err)
	}
	p := a.Plates[0]
	if p.PrintTimeSeconds != 3900 {
		t.Errorf("print time = %d, want 3900", p.PrintTimeSeconds)
	}
	if len(p.Filaments) != 2 || p.Filaments[0].Grams != 12.5 || p.Filaments[1].Grams != 3.25 {
		t.Fatalf("filaments = %+v", p.Filaments)
	}
	if p.Filaments[0].ColorHex != "#FF0000" || p.Filaments[1].Material != "PETG" {
		t.Errorf("tail metadata not parsed: %+v", p.Filaments)
	}
}

// A highly compressible embedded G-code (zip bomb) is rejected while streaming;
// with nothing else to read the archive is reported as not sliced.
func TestParse3MF_EmbeddedGCodeBombRejected(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	zw.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(w, flate.BestCompression)
	})
	w, err := zw.Create("Metadata/plate_1.gcode")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(bytes.Repeat([]byte{'G'}, 8<<20)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = Parse3MF(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != ErrNotSliced {
		t.Fatalf("err = %v, want ErrNotSliced", err)
	}
}

func TestDetectSlicer3MF_ApplicationMetadata(t *testing.T) {
	cases := map[string]struct{ name, version string }{
		`<metadata name="Application">BambuStudio-02.08.02.61</metadata>`: {"BambuStudio", "02.08.02.61"},
		`<metadata name="Application">OrcaSlicer-2.1.1</metadata>`:        {"OrcaSlicer", "2.1.1"},
		`<metadata name="Application">PrusaSlicer-2.7.1+win64</metadata>`: {"PrusaSlicer", "2.7.1"},
	}
	for head, want := range cases {
		got := detectSlicer3MF([]byte(head), nil, nil)
		if got.Name != want.name || !strings.HasPrefix(got.Version, want.version) {
			t.Errorf("%s → %+v, want %s %s", head, got, want.name, want.version)
		}
	}
}
