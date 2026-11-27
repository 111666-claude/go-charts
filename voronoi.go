// MIT License

// Copyright (c) 2022 Tree Xie

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package charts

import (
	"errors"
	"math"
)

// VPoint is a point in the two dimensional data coordinate space.
type VPoint struct {
	X float64
	Y float64
}

// VoronoiRect is an axis aligned rectangle in the data coordinate space.
type VoronoiRect struct {
	// X0 is the left boundary of the rectangle
	X0 float64
	// Y0 is the bottom boundary of the rectangle
	Y0 float64
	// X1 is the right boundary of the rectangle
	X1 float64
	// Y1 is the top boundary of the rectangle
	Y1 float64
}

// Width returns the width of the rectangle.
func (r VoronoiRect) Width() float64 {
	return r.X1 - r.X0
}

// Height returns the height of the rectangle.
func (r VoronoiRect) Height() float64 {
	return r.Y1 - r.Y0
}

// Area returns the area of the rectangle.
func (r VoronoiRect) Area() float64 {
	return r.Width() * r.Height()
}

// Contains reports whether the point is inside of the rectangle.
func (r VoronoiRect) Contains(p VPoint) bool {
	return p.X >= r.X0 && p.X <= r.X1 && p.Y >= r.Y0 && p.Y <= r.Y1
}

// VoronoiCell is the bounded service region of a site.
type VoronoiCell struct {
	// Index is the index of the site in the original input list.
	Index int
	// Site is the original site point.
	Site VPoint
	// Polygon is the counter clockwise ordered region polygon clipped by
	// the view rectangle. It is empty when the site is a duplicate of an
	// earlier site.
	Polygon []VPoint
	// Duplicate reports whether the site is a duplicate of an earlier one,
	// in which case the cell owns no area.
	Duplicate bool
}

// Area returns the signed area of the cell polygon.
func (c *VoronoiCell) Area() float64 {
	return polygonArea(c.Polygon)
}

// voronoiEpsilonScale is used to derive a scale aware floating point tolerance.
const voronoiEpsilonScale = 1e-10

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// pointsEqual reports whether two points are treated as the same location.
// A scale aware tolerance keeps the result deterministic for coordinates of
// any reasonable magnitude while collapsing only near identical points.
func pointsEqual(a, b VPoint, eps float64) bool {
	return math.Abs(a.X-b.X) <= eps && math.Abs(a.Y-b.Y) <= eps
}

// cross returns the z component of the cross product of vectors ab and ac.
func cross(a, b, c VPoint) float64 {
	return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
}

// polygonArea returns the area of a simple polygon. The vertices may be in
// any order, the absolute value is returned.
func polygonArea(points []VPoint) float64 {
	n := len(points)
	if n < 3 {
		return 0
	}
	sum := float64(0)
	for i := 0; i < n; i++ {
		a := points[i]
		b := points[(i+1)%n]
		sum += a.X*b.Y - b.X*a.Y
	}
	return math.Abs(sum) / 2
}

// clampPoint forces a point back into the view rectangle. This is only a
// safety net for floating point drift produced by the line intersections.
func clampPoint(p VPoint, rect VoronoiRect) VPoint {
	if p.X < rect.X0 {
		p.X = rect.X0
	}
	if p.X > rect.X1 {
		p.X = rect.X1
	}
	if p.Y < rect.Y0 {
		p.Y = rect.Y0
	}
	if p.Y > rect.Y1 {
		p.Y = rect.Y1
	}
	return p
}

// rectPolygon returns the counter clockwise vertex list of the rectangle,
// starting from the bottom left corner.
func rectPolygon(rect VoronoiRect) []VPoint {
	return []VPoint{
		{X: rect.X0, Y: rect.Y0},
		{X: rect.X1, Y: rect.Y0},
		{X: rect.X1, Y: rect.Y1},
		{X: rect.X0, Y: rect.Y1},
	}
}

// clipHalfPlane keeps the part of the polygon which is at least as close to
// site as to other. The bisector line between the two sites splits the plane:
// points p satisfying 2*(other-site)·p <= |other|²-|site|² belong to site.
// The input polygon is expected to be counter clockwise ordered.
func clipHalfPlane(polygon []VPoint, site, other VPoint) []VPoint {
	if len(polygon) == 0 {
		return polygon
	}
	// f(p) is positive on the site side and negative on the other side.
	f := func(p VPoint) float64 {
		return (other.X*other.X + other.Y*other.Y - site.X*site.X - site.Y*site.Y) -
			2*((other.X-site.X)*p.X+(other.Y-site.Y)*p.Y)
	}
	intersect := func(a, b VPoint) VPoint {
		fa := f(a)
		fb := f(b)
		// The segment straddles the line, the denominator can not be zero.
		t := fa / (fa - fb)
		return VPoint{
			X: a.X + t*(b.X-a.X),
			Y: a.Y + t*(b.Y-a.Y),
		}
	}
	result := make([]VPoint, 0, len(polygon)+1)
	// The exact sign is used on purpose: two sites sharing a bisector compute
	// the same intersection points, so the resulting polygons share their
	// boundary instead of overlapping or leaving a gap.
	for i, current := range polygon {
		previous := polygon[(i-1+len(polygon))%len(polygon)]
		currentInside := f(current) >= 0
		previousInside := f(previous) >= 0
		if currentInside {
			if !previousInside {
				result = append(result, intersect(previous, current))
			}
			result = append(result, current)
		} else if previousInside {
			result = append(result, intersect(previous, current))
		}
	}
	return result
}

// removeRepeated drops consecutive duplicate vertices, including the pair
// formed by the first and the last vertex of the ring.
func removeRepeated(points []VPoint, eps float64) []VPoint {
	if len(points) < 2 {
		return points
	}
	result := make([]VPoint, 0, len(points))
	for _, p := range points {
		if len(result) != 0 && pointsEqual(result[len(result)-1], p, eps) {
			continue
		}
		result = append(result, p)
	}
	if len(result) > 1 && pointsEqual(result[0], result[len(result)-1], eps) {
		result = result[:len(result)-1]
	}
	return result
}

// removeCollinear drops the middle vertex of three consecutive collinear
// points so that the polygon only keeps real corners. This makes the output
// vertices stable and minimal, including for collinear site arrangements.
// The cross product is normalized by the edge lengths so that the same
// angular tolerance works for coordinates of any magnitude.
func removeCollinear(points []VPoint, eps float64) []VPoint {
	n := len(points)
	if n < 3 {
		return points
	}
	kept := make([]bool, n)
	for i := range points {
		kept[i] = true
	}
	for i := 0; i < n; i++ {
		a := points[(i-1+n)%n]
		b := points[i]
		c := points[(i+1)%n]
		ab := math.Hypot(b.X-a.X, b.Y-a.Y)
		bc := math.Hypot(c.X-b.X, c.Y-b.Y)
		if ab == 0 || bc == 0 || math.Abs(cross(a, b, c))/(ab*bc) <= eps {
			kept[i] = false
		}
	}
	result := make([]VPoint, 0, n)
	for i, p := range points {
		if kept[i] {
			result = append(result, p)
		}
	}
	return result
}

// BoundedVoronoi computes the nearest site region of every site, clipped by
// the view rectangle. The result always has the same length and order as the
// input sites:
//   - non finite site coordinates and non finite or invalid rectangles return
//     an error;
//   - an empty site list returns an empty result without error;
//   - a site identical (within tolerance) to an earlier site gets an empty
//     polygon and Duplicate set to true, hence it owns no area;
//   - every non empty polygon is contained in the rectangle, polygons do not
//     overlap and together they cover the whole rectangle without gaps.
func BoundedVoronoi(sites []VPoint, rect VoronoiRect) ([]VoronoiCell, error) {
	if !isFinite(rect.X0) || !isFinite(rect.Y0) ||
		!isFinite(rect.X1) || !isFinite(rect.Y1) {
		return nil, errors.New("voronoi: rectangle coordinates must be finite")
	}
	if rect.X1 <= rect.X0 || rect.Y1 <= rect.Y0 {
		return nil, errors.New("voronoi: rectangle width and height must be positive")
	}
	cells := make([]VoronoiCell, len(sites))
	if len(sites) == 0 {
		return cells, nil
	}
	coordMax := math.Max(
		math.Max(math.Abs(rect.X0), math.Abs(rect.X1)),
		math.Max(math.Abs(rect.Y0), math.Abs(rect.Y1)),
	)
	for _, s := range sites {
		if !isFinite(s.X) || !isFinite(s.Y) {
			return nil, errors.New("voronoi: site coordinates must be finite")
		}
		if m := math.Max(math.Abs(s.X), math.Abs(s.Y)); m > coordMax {
			coordMax = m
		}
	}
	eps := voronoiEpsilonScale * math.Max(1, coordMax)

	seen := make([]VPoint, 0, len(sites))
	for i, site := range sites {
		cell := VoronoiCell{
			Index: i,
			Site:  site,
		}
		duplicate := false
		for _, previous := range seen {
			if pointsEqual(site, previous, eps) {
				duplicate = true
				break
			}
		}
		if duplicate {
			cell.Duplicate = true
			cells[i] = cell
			continue
		}
		seen = append(seen, site)

		polygon := rectPolygon(rect)
		for _, other := range sites {
			if pointsEqual(site, other, eps) {
				continue
			}
			polygon = clipHalfPlane(polygon, site, other)
			if len(polygon) < 3 {
				break
			}
		}
		for j := range polygon {
			polygon[j] = clampPoint(polygon[j], rect)
		}
		polygon = removeRepeated(polygon, eps)
		polygon = removeCollinear(polygon, voronoiEpsilonScale)
		cell.Polygon = polygon
		cells[i] = cell
	}
	return cells, nil
}
