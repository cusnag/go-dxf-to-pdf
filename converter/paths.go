package converter

import "github.com/tdewolff/canvas"

// splitLongPath returns the path cut into pieces of at most maxSegments
// segments each. Cuts fall on a subpath start when one is due, otherwise the
// next piece restarts at the point the previous one ended on (a join becomes
// two caps at the seam).
//
// A filled path must not be split: its subpaths' windings interact. This is
// only used for stroked paths.
func splitLongPath(p *canvas.Path, maxSegments int) []*canvas.Path {
	if p == nil || maxSegments <= 0 {
		return []*canvas.Path{p}
	}
	total := pathSegments(p)
	if total <= maxSegments {
		return []*canvas.Path{p}
	}

	pieces := make([]*canvas.Path, 0, total/maxSegments+1)
	cur := &canvas.Path{}
	curSegs := 0
	atSubpathStart := false // cur's last MoveTo is the subpath's real start
	var subpathStart, last canvas.Point

	finalize := func() {
		if !cur.Empty() {
			pieces = append(pieces, cur)
		}
	}
	// addSegment finalizes the current piece first when it is full. A new
	// piece restarts at last; it is at the subpath start only if the cut
	// happened to fall there.
	addSegment := func(appendSegment func()) {
		if curSegs >= maxSegments {
			pieces = append(pieces, cur)
			cur = &canvas.Path{}
			cur.MoveTo(last.X, last.Y)
			curSegs = 0
			atSubpathStart = last.Equals(subpathStart)
		}
		appendSegment()
		curSegs++
	}

	for s := p.Scanner(); s.Scan(); {
		switch s.Cmd() {
		case canvas.MoveToCmd:
			pt := s.End()
			if curSegs >= maxSegments {
				pieces = append(pieces, cur)
				cur = &canvas.Path{}
				curSegs = 0
			}
			cur.MoveTo(pt.X, pt.Y)
			atSubpathStart = true
			subpathStart = pt
			last = pt

		case canvas.CloseCmd:
			// Close goes back to the most recent MoveTo. If a cut moved that
			// start into the middle of the subpath, draw the closing edge as a
			// line instead: Close would close to the cut, not the real start.
			if atSubpathStart {
				cur.Close()
			} else if !last.Equals(subpathStart) {
				addSegment(func() { cur.LineTo(subpathStart.X, subpathStart.Y) })
			}
			last = subpathStart

		default:
			addSegment(func() {
				switch s.Cmd() {
				case canvas.LineToCmd:
					cur.LineTo(s.End().X, s.End().Y)
				case canvas.QuadToCmd:
					cp := s.CP1()
					cur.QuadTo(cp.X, cp.Y, s.End().X, s.End().Y)
				case canvas.CubeToCmd:
					cp1, cp2 := s.CP1(), s.CP2()
					cur.CubeTo(cp1.X, cp1.Y, cp2.X, cp2.Y, s.End().X, s.End().Y)
				case canvas.ArcToCmd:
					rx, ry, rot, large, sweep := s.Arc()
					cur.ArcTo(rx, ry, rot, large, sweep, s.End().X, s.End().Y)
				}
			})
			last = s.End()
		}
	}
	finalize()
	return pieces
}

// pathSegments counts a path's drawn segments, using the same rule as the
// renderer's batching bound: a MoveTo starts a subpath and a Close does not
// add a segment of its own.
func pathSegments(p *canvas.Path) int {
	n := 0
	for s := p.Scanner(); s.Scan(); {
		switch s.Cmd() {
		case canvas.MoveToCmd, canvas.CloseCmd:
		default:
			n++
		}
	}
	return n
}
