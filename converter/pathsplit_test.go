package converter

import (
	"bytes"
	"compress/zlib"
	"os"
	"path/filepath"
	"strings"
	"testing"

	dxf "github.com/kedazo/dxf-go"
)

// pdfPathSegments returns the largest number of segments in one painted path
// across the PDF's decompressed content streams.
func pdfPathSegments(t *testing.T, path string) (max, paths int) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for {
		i := bytes.Index(data, []byte("stream"))
		if i < 0 {
			break
		}
		rest := bytes.TrimPrefix(data[i+len("stream"):], []byte("\r"))
		rest = bytes.TrimPrefix(rest, []byte("\n"))
		j := bytes.Index(rest, []byte("endstream"))
		if j < 0 {
			break
		}
		if zr, err := zlib.NewReader(bytes.NewReader(rest[:j])); err == nil {
			var content bytes.Buffer
			_, _ = content.ReadFrom(zr)
			_ = zr.Close()
			m, n := countPaintedPaths(content.String())
			max, paths = maxInt(max, m), paths+n
		}
		data = rest[j+len("endstream"):]
	}
	return max, paths
}

// countPaintedPaths counts the segments of each path object in a PDF content
// stream. A MoveTo starts a subpath, line/curve operators add a segment, and
// a painting operator ends the path.
func countPaintedPaths(content string) (max, paths int) {
	segs := 0
	for _, tok := range strings.Fields(content) {
		switch tok {
		case "l", "c", "v", "y":
			segs++
		case "S", "s", "f", "F", "f*", "B", "B*", "b", "b*", "n":
			if segs > 0 {
				paths++
				if segs > max {
					max = segs
				}
			}
			segs = 0
		}
	}
	return max, paths
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// A long stroked path is split at MaxPathSegments; 0 keeps it whole.
func TestMaxPathSegmentsInPDF(t *testing.T) {
	const segments, maxSegments = 1000, 100
	drawing := dxf.NewDrawing()
	point := func(i int) dxf.Point {
		return dxf.Point{X: float64(i) * 0.1, Y: float64(i%2) * 10}
	}
	for i := 0; i < segments; i++ {
		line := dxf.NewLine()
		line.P1, line.P2 = point(i), point(i+1)
		drawing.Entities = append(drawing.Entities, line)
	}

	dir := t.TempDir()
	in := filepath.Join(dir, "in.dxf")
	if err := drawing.SaveFile(in); err != nil {
		t.Fatal(err)
	}
	convert := func(name string, cap int) (max, paths int) {
		out := filepath.Join(dir, name)
		if _, err := Convert(in, out, Options{
			Scale: "1:1", AutoPaper: true, Margin: 10, MaxPathSegments: cap,
		}); err != nil {
			t.Fatalf("Convert(%s): %v", name, err)
		}
		return pdfPathSegments(t, out)
	}

	if max, _ := convert("off.pdf", 0); max <= maxSegments {
		t.Fatalf("unsplit PDF has %d segments in its longest path, want more than %d", max, maxSegments)
	}
	max, paths := convert("split.pdf", maxSegments)
	if max > maxSegments {
		t.Errorf("split PDF has a %d-segment path, want at most %d", max, maxSegments)
	}
	if paths < 2 {
		t.Errorf("split PDF has %d painted paths, want the long path split", paths)
	}
}
