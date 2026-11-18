package converter

import (
	"testing"

	"github.com/tdewolff/canvas"
)

// zigzagPath returns a path with n non-collinear line segments.
func zigzagPath(n int) *canvas.Path {
	p := &canvas.Path{}
	p.MoveTo(0, 0)
	for i := 1; i <= n; i++ {
		y := 0.0
		if i%2 != 0 {
			y = 10
		}
		p.LineTo(float64(i), y)
	}
	return p
}

func commandCount(p *canvas.Path, cmd float64) int {
	n := 0
	for s := p.Scanner(); s.Scan(); {
		if s.Cmd() == cmd {
			n++
		}
	}
	return n
}

func TestSplitLongPathUnderLimit(t *testing.T) {
	p := zigzagPath(10)
	got := splitLongPath(p, 10)
	if len(got) != 1 || got[0] != p {
		t.Fatalf("splitLongPath under the limit = %d pieces, want the original path", len(got))
	}
}

func TestSplitLongPathDisabled(t *testing.T) {
	p := zigzagPath(5)
	for _, max := range []int{0, -1} {
		got := splitLongPath(p, max)
		if len(got) != 1 || got[0] != p {
			t.Errorf("splitLongPath(max=%d) = %d pieces, want the original path", max, len(got))
		}
	}
}

func TestSplitLongPathSegments(t *testing.T) {
	const segments, max = 50, 20
	p := zigzagPath(segments)
	pieces := splitLongPath(p, max)
	if len(pieces) != 3 {
		t.Fatalf("got %d pieces, want 3", len(pieces))
	}
	want := []int{20, 20, 10}
	last := p.StartPos()
	for i, piece := range pieces {
		if got := pathSegments(piece); got != want[i] {
			t.Errorf("piece %d has %d segments, want %d", i, got, want[i])
		}
		if i > 0 && !piece.StartPos().Equals(last) {
			t.Errorf("piece %d starts at %v, want the previous end %v", i, piece.StartPos(), last)
		}
		last = piece.Pos()
	}
}

func TestSplitLongPathCurves(t *testing.T) {
	const segments, max = 30, 10
	p := &canvas.Path{}
	p.MoveTo(0, 0)
	for i := 1; i <= segments; i++ {
		x := float64(i * 10)
		p.CubeTo(x-5, 5, x-5, -5, x, 0)
	}
	pieces := splitLongPath(p, max)
	if len(pieces) != 3 {
		t.Fatalf("got %d pieces, want 3", len(pieces))
	}
	for i, piece := range pieces {
		if got := pathSegments(piece); got > max {
			t.Errorf("piece %d has %d segments, want at most %d", i, got, max)
		}
		if got := commandCount(piece, canvas.CubeToCmd); got != pathSegments(piece) {
			t.Errorf("piece %d has %d curves and %d segments; curves were split", i, got, pathSegments(piece))
		}
	}
}

func TestSplitLongPathKeepsClose(t *testing.T) {
	closed := zigzagPath(5)
	closed.Close()
	got := splitLongPath(closed, 10)
	if len(got) != 1 || got[0] != closed || !got[0].Closed() {
		t.Fatal("a closed path under the limit lost its Close")
	}

	// A small closed subpath followed by a long open one: the cut falls in
	// the second subpath, and the first keeps its Close.
	p := zigzagPath(5)
	p.Close()
	next := &canvas.Path{}
	next.MoveTo(100, 0)
	for i := 1; i <= 30; i++ {
		y := 0.0
		if i%2 != 0 {
			y = 10
		}
		next.LineTo(100+float64(i), y)
	}
	p.Append(next)
	pieces := splitLongPath(p, 20)
	if len(pieces) < 2 {
		t.Fatalf("got %d pieces, want a split", len(pieces))
	}
	found := false
	for _, piece := range pieces {
		found = found || commandCount(piece, canvas.CloseCmd) > 0
	}
	if !found {
		t.Error("the closed subpath lost its Close")
	}
}

func TestSplitLongPathArcs(t *testing.T) {
	const segments, max = 30, 10
	p := &canvas.Path{}
	p.MoveTo(0, 0)
	for i := 1; i <= segments; i++ {
		x := float64(i * 10)
		p.ArcTo(5, 3, 0, false, i%2 == 0, x, 10)
	}
	pieces := splitLongPath(p, max)
	if len(pieces) < 2 {
		t.Fatalf("got %d pieces, want the path split", len(pieces))
	}
	total := 0
	for i, piece := range pieces {
		n := pathSegments(piece)
		if n > max {
			t.Errorf("piece %d has %d segments, want at most %d", i, n, max)
		}
		total += n
	}
	if total != pathSegments(p) {
		t.Errorf("pieces have %d segments in total, want %d", total, pathSegments(p))
	}
}
