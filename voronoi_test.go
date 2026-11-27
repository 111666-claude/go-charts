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
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func pointInPolygon(p VPoint, polygon []VPoint) bool {
	n := len(polygon)
	if n < 3 {
		return false
	}
	eps := 1e-9
	onSegment := func(a, b VPoint) bool {
		if math.Abs(cross(a, b, p)) > eps {
			return false
		}
		return p.X >= math.Min(a.X, b.X)-eps &&
			p.X <= math.Max(a.X, b.X)+eps &&
			p.Y >= math.Min(a.Y, b.Y)-eps &&
			p.Y <= math.Max(a.Y, b.Y)+eps
	}
	inside := false
	for i := 0; i < n; i++ {
		a := polygon[i]
		b := polygon[(i+1)%n]
		if onSegment(a, b) {
			return true
		}
		if (a.Y > p.Y) != (b.Y > p.Y) {
			x := (b.X-a.X)*(p.Y-a.Y)/(b.Y-a.Y) + a.X
			if p.X < x {
				inside = !inside
			}
		}
	}
	return inside
}

func pointsContained(polygon []VPoint, rect VoronoiRect) bool {
	for _, p := range polygon {
		if !rect.Contains(p) {
			return false
		}
	}
	return true
}

// assertPartition verifies the fundamental Voronoi invariants: every cell is
// inside the rectangle, the cells cover the whole rectangle without gaps or
// overlaps, and every covered point belongs to a nearest site.
func assertPartition(t *testing.T, sites []VPoint, rect VoronoiRect, cells []VoronoiCell) {
	assert := assert.New(t)
	assert.Len(cells, len(sites))

	totalArea := float64(0)
	activeCells := make([]VoronoiCell, 0)
	for _, cell := range cells {
		if cell.Duplicate {
			assert.Empty(cell.Polygon)
			continue
		}
		assert.GreaterOrEqual(len(cell.Polygon), 3)
		assert.True(pointsContained(cell.Polygon, rect))
		assert.Greater(cell.Area(), float64(0))
		totalArea += cell.Area()
		activeCells = append(activeCells, cell)
	}
	// No gaps and no overlaps: the cell areas add up to the rectangle area.
	assert.InDelta(rect.Area(), totalArea, 1e-8*math.Max(1, rect.Area()))

	// A dense grid must be fully covered, and every covered point must be
	// nearest to the site whose cell contains it.
	const grid = 40
	for i := 0; i <= grid; i++ {
		for j := 0; j <= grid; j++ {
			p := VPoint{
				X: rect.X0 + rect.Width()*float64(i)/grid,
				Y: rect.Y0 + rect.Height()*float64(j)/grid,
			}
			nearestDist := math.MaxFloat64
			for _, s := range sites {
				d := (p.X-s.X)*(p.X-s.X) + (p.Y-s.Y)*(p.Y-s.Y)
				if d < nearestDist {
					nearestDist = d
				}
			}
			covered := false
			for _, cell := range activeCells {
				if !pointInPolygon(p, cell.Polygon) {
					continue
				}
				covered = true
				s := sites[cell.Index]
				d := (p.X-s.X)*(p.X-s.X) + (p.Y-s.Y)*(p.Y-s.Y)
				assert.InDelta(nearestDist, d, 1e-8, "point %+v must belong to a nearest site", p)
			}
			assert.True(covered, "point %+v is not covered by any cell", p)
		}
	}
}

func TestBoundedVoronoiTwoSites(t *testing.T) {
	assert := assert.New(t)
	rect := VoronoiRect{X0: 0, Y0: -1, X1: 2, Y1: 1}
	sites := []VPoint{
		{X: 0, Y: 0},
		{X: 2, Y: 0},
	}
	cells, err := BoundedVoronoi(sites, rect)
	assert.Nil(err)

	// Fixed vertices: the view is split in half along x = 1.
	assert.Equal([]VPoint{
		{X: 0, Y: -1},
		{X: 1, Y: -1},
		{X: 1, Y: 1},
		{X: 0, Y: 1},
	}, cells[0].Polygon)
	assert.Equal([]VPoint{
		{X: 1, Y: -1},
		{X: 2, Y: -1},
		{X: 2, Y: 1},
		{X: 1, Y: 1},
	}, cells[1].Polygon)
	// Fixed coverage area: each site owns exactly half of the 2x2 view.
	assert.InDelta(2.0, cells[0].Area(), 1e-12)
	assert.InDelta(2.0, cells[1].Area(), 1e-12)
	assert.False(cells[0].Duplicate)
	assert.False(cells[1].Duplicate)
	assertPartition(t, sites, rect, cells)
}

func TestBoundedVoronoiSingleSite(t *testing.T) {
	assert := assert.New(t)
	rect := VoronoiRect{X0: 0, Y0: 0, X1: 4, Y1: 3}
	cells, err := BoundedVoronoi([]VPoint{{X: 1, Y: 1}}, rect)
	assert.Nil(err)
	assert.Len(cells, 1)
	assert.InDelta(rect.Area(), cells[0].Area(), 1e-12)
	assert.True(pointsContained(cells[0].Polygon, rect))
}

func TestBoundedVoronoiEmptySites(t *testing.T) {
	assert := assert.New(t)
	rect := VoronoiRect{X0: 0, Y0: 0, X1: 1, Y1: 1}
	cells, err := BoundedVoronoi(nil, rect)
	assert.Nil(err)
	assert.Empty(cells)
}

func TestBoundedVoronoiNonFinite(t *testing.T) {
	assert := assert.New(t)
	rect := VoronoiRect{X0: 0, Y0: 0, X1: 1, Y1: 1}

	_, err := BoundedVoronoi([]VPoint{{X: math.NaN(), Y: 0}}, rect)
	assert.NotNil(err)
	_, err = BoundedVoronoi([]VPoint{{X: 0, Y: math.Inf(1)}}, rect)
	assert.NotNil(err)
	_, err = BoundedVoronoi([]VPoint{{X: 0, Y: 0}}, VoronoiRect{
		X0: math.NaN(), Y0: 0, X1: 1, Y1: 1,
	})
	assert.NotNil(err)
	_, err = BoundedVoronoi([]VPoint{{X: 0, Y: 0}}, VoronoiRect{
		X0: 1, Y0: 0, X1: 1, Y1: 1,
	})
	assert.NotNil(err)
	_, err = BoundedVoronoi([]VPoint{{X: 0, Y: 0}}, VoronoiRect{
		X0: 0, Y0: 2, X1: 1, Y1: 1,
	})
	assert.NotNil(err)
}

func TestBoundedVoronoiDuplicates(t *testing.T) {
	assert := assert.New(t)
	rect := VoronoiRect{X0: 0, Y0: 0, X1: 2, Y1: 2}
	sites := []VPoint{
		{X: 1, Y: 1},
		{X: 1, Y: 1},
		{X: 1, Y: 1.0000000001},
	}
	cells, err := BoundedVoronoi(sites, rect)
	assert.Nil(err)
	assert.False(cells[0].Duplicate)
	assert.InDelta(rect.Area(), cells[0].Area(), 1e-12)
	assert.True(cells[1].Duplicate)
	assert.Empty(cells[1].Polygon)
	assert.True(cells[2].Duplicate)
	assert.Empty(cells[2].Polygon)
}

func TestBoundedVoronoiCollinear(t *testing.T) {
	assert := assert.New(t)
	rect := VoronoiRect{X0: 0, Y0: -1, X1: 2, Y1: 1}
	sites := []VPoint{
		{X: 0, Y: 0},
		{X: 1, Y: 0},
		{X: 2, Y: 0},
	}
	cells, err := BoundedVoronoi(sites, rect)
	assert.Nil(err)
	// Slabs of width 0.5, 1 and 0.5 over a height of 2.
	assert.InDelta(1.0, cells[0].Area(), 1e-12)
	assert.InDelta(2.0, cells[1].Area(), 1e-12)
	assert.InDelta(1.0, cells[2].Area(), 1e-12)
	for _, cell := range cells {
		assert.Len(cell.Polygon, 4, "collinear sites must produce rectangle cells")
	}
	assertPartition(t, sites, rect, cells)
}

func TestBoundedVoronoiSitesOnEdges(t *testing.T) {
	assert := assert.New(t)
	rect := VoronoiRect{X0: 0, Y0: 0, X1: 2, Y1: 2}
	sites := []VPoint{
		{X: 0, Y: 0},
		{X: 2, Y: 0},
		{X: 0, Y: 2},
		{X: 2, Y: 2},
	}
	cells, err := BoundedVoronoi(sites, rect)
	assert.Nil(err)
	assertPartition(t, sites, rect, cells)
}

func TestBoundedVoronoiEquidistant(t *testing.T) {
	assert := assert.New(t)
	rect := VoronoiRect{X0: 0, Y0: 0, X1: 2, Y1: 2}
	sites := []VPoint{
		{X: 0.5, Y: 0.5},
		{X: 1.5, Y: 0.5},
		{X: 0.5, Y: 1.5},
		{X: 1.5, Y: 1.5},
	}
	cells, err := BoundedVoronoi(sites, rect)
	assert.Nil(err)
	// Perfect symmetry: every site owns a quarter of the view.
	for _, cell := range cells {
		assert.InDelta(1.0, cell.Area(), 1e-12)
		assert.Len(cell.Polygon, 4)
	}
	assertPartition(t, sites, rect, cells)
}

func TestBoundedVoronoiGeneralPoints(t *testing.T) {
	rect := VoronoiRect{X0: -3, Y0: -2, X1: 4, Y1: 3}
	sites := []VPoint{
		{X: -2.7, Y: -1.8},
		{X: 3.8, Y: -1.5},
		{X: 0.3, Y: 0.2},
		{X: -1.1, Y: 2.7},
		{X: 3.4, Y: 2.6},
		{X: 1.9, Y: 0.4},
	}
	cells, err := BoundedVoronoi(sites, rect)
	assert.Nil(t, err)
	assertPartition(t, sites, rect, cells)
}

func TestBoundedVoronoiSitesOutsideRect(t *testing.T) {
	assert := assert.New(t)
	rect := VoronoiRect{X0: 0, Y0: 0, X1: 2, Y1: 2}
	// The third site is far outside of the view and owns no area in it.
	sites := []VPoint{
		{X: 0.5, Y: 1},
		{X: 1.5, Y: 1},
		{X: 100, Y: 100},
	}
	cells, err := BoundedVoronoi(sites, rect)
	assert.Nil(err)
	assert.False(cells[2].Duplicate)
	assert.Less(len(cells[2].Polygon), 3)
	assert.Equal(0.0, cells[2].Area())
	// The two in view sites still partition the whole rectangle.
	assert.InDelta(rect.Area(), cells[0].Area()+cells[1].Area(), 1e-8)
	assertPartition(t, sites[:2], rect, cells[:2])
}

func TestBoundedVoronoiDeterministic(t *testing.T) {
	assert := assert.New(t)
	rect := VoronoiRect{X0: 0, Y0: 0, X1: 5, Y1: 5}
	sites := []VPoint{
		{X: 1, Y: 1},
		{X: 4, Y: 1},
		{X: 2.5, Y: 4},
	}
	first, err := BoundedVoronoi(sites, rect)
	assert.Nil(err)
	second, err := BoundedVoronoi(sites, rect)
	assert.Nil(err)
	assert.Equal(first, second)
}

// TestBoundedVoronoiStress runs many deterministic pseudo random site sets
// and verifies the partition invariants for each of them.
func TestBoundedVoronoiStress(t *testing.T) {
	assert := assert.New(t)
	rect := VoronoiRect{X0: -5, Y0: -5, X1: 5, Y1: 5}
	// A simple linear congruential generator keeps the test deterministic.
	seed := uint32(0x12345678)
	next := func() float64 {
		seed = seed*1103515245 + 12345
		// Map the pseudo random value into the rect range.
		return rect.X0 + float64(seed%1000)/1000*rect.Width()
	}
	for round := 0; round < 30; round++ {
		siteCount := 2 + int(seed%12)
		sites := make([]VPoint, siteCount)
		for i := range sites {
			sites[i] = VPoint{X: next(), Y: next()}
		}
		cells, err := BoundedVoronoi(sites, rect)
		assert.Nil(err)
		assertPartition(t, sites, rect, cells)

		// The centroid of every cell must be strictly closer to its own site
		// than to any other non duplicate site.
		for _, cell := range cells {
			if cell.Duplicate {
				continue
			}
			cx, cy := float64(0), float64(0)
			for _, p := range cell.Polygon {
				cx += p.X
				cy += p.Y
			}
			n := float64(len(cell.Polygon))
			center := VPoint{X: cx / n, Y: cy / n}
			ownDist := dist2(center, cell.Site)
			for j, other := range sites {
				if j == cell.Index {
					continue
				}
				assert.LessOrEqual(ownDist, dist2(center, other)+1e-8)
			}
		}
	}
}

func dist2(a, b VPoint) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx*dx + dy*dy
}
