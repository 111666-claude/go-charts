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
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func choroplethValue(v float64) *float64 {
	return &v
}

func unitSquare(x0 float64) []Point2D {
	return []Point2D{
		{x0, 0},
		{x0 + 1, 0},
		{x0 + 1, 1},
		{x0, 1},
	}
}

func decodePNG(t *testing.T, p *Painter) *image.RGBA {
	buf, err := p.Bytes()
	assert.Nil(t, err)
	img, err := png.Decode(bytes.NewReader(buf))
	assert.Nil(t, err)
	rgba, ok := img.(*image.RGBA)
	assert.True(t, ok)
	return rgba
}

func TestChoroplethValidation(t *testing.T) {
	assert := assert.New(t)
	square := unitSquare(0)

	tests := []struct {
		name    string
		opt     ChoroplethChartOption
		wantErr error
	}{
		{
			name:    "empty regions",
			opt:     ChoroplethChartOption{Regions: nil},
			wantErr: errEmptyRegions,
		},
		{
			name: "too few vertices",
			opt: ChoroplethChartOption{
				Regions: []MapRegion{
					{Name: "x", Polygon: []Point2D{{0, 0}, {1, 0}}, Value: choroplethValue(1)},
				},
			},
			wantErr: errEmptyRegion,
		},
		{
			name: "non finite vertex",
			opt: ChoroplethChartOption{
				Regions: []MapRegion{
					{Name: "x", Polygon: []Point2D{{0, 0}, {1, 0}, {math.NaN(), 1}, {0, 1}}, Value: choroplethValue(1)},
				},
			},
			wantErr: errNonFiniteVertex,
		},
		{
			name: "infinite vertex",
			opt: ChoroplethChartOption{
				Regions: []MapRegion{
					{Name: "x", Polygon: []Point2D{{0, 0}, {1, 0}, {1, math.Inf(1)}, {0, 1}}, Value: choroplethValue(1)},
				},
			},
			wantErr: errNonFiniteVertex,
		},
		{
			name: "self intersecting ring",
			opt: ChoroplethChartOption{
				Regions: []MapRegion{
					{Name: "x", Polygon: []Point2D{{0, 0}, {1, 1}, {1, 0}, {0, 1}}, Value: choroplethValue(1)},
				},
			},
			wantErr: errSelfIntersectRing,
		},
		{
			name: "duplicated name",
			opt: ChoroplethChartOption{
				Regions: []MapRegion{
					{Name: "x", Polygon: square, Value: choroplethValue(1)},
					{Name: "x", Polygon: square, Value: choroplethValue(2)},
				},
			},
			wantErr: errors.New("choropleth region name is duplicated: x"),
		},
		{
			name: "invalid visual map range",
			opt: ChoroplethChartOption{
				Regions:   []MapRegion{{Name: "x", Polygon: square, Value: choroplethValue(1)}},
				VisualMap: VisualMap{Min: NewFloatPoint(4), Max: NewFloatPoint(4)},
			},
			wantErr: errInvalidVisualMap,
		},
		{
			name: "invalid piece",
			opt: ChoroplethChartOption{
				Regions: []MapRegion{{Name: "x", Polygon: square, Value: choroplethValue(1)}},
				VisualMap: VisualMap{
					Pieces: []VisualMapPiece{{Min: 2, Max: 2}},
				},
			},
			wantErr: errInvalidVisualMap,
		},
		{
			name: "non finite hole",
			opt: ChoroplethChartOption{
				Regions: []MapRegion{{
					Name:    "x",
					Polygon: square,
					Holes:   [][]Point2D{{{0.2, 0.2}, {0.8, 0.2}, {math.Inf(-1), 0.8}, {0.2, 0.8}}},
					Value:   choroplethValue(1),
				}},
			},
			wantErr: errNonFiniteVertex,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.opt.Width = 100
			tt.opt.Height = 100
			_, err := ChoroplethRender(tt.opt)
			assert.Equal(tt.wantErr, err)
		})
	}
}

func TestChoroplethConcaveAndMissing(t *testing.T) {
	assert := assert.New(t)
	// concave L shape and a missing value region should both render
	opt := ChoroplethChartOption{
		Width:  200,
		Height: 240,
		Regions: []MapRegion{
			{
				Name: "L",
				Polygon: []Point2D{
					{0, 0}, {2, 0}, {2, 1}, {1, 1}, {1, 2}, {0, 2},
				},
				Value: choroplethValue(1),
			},
			{
				Name:    "missing",
				Polygon: unitSquare(3),
				Value:   nil,
			},
		},
		VisualMap: VisualMap{
			Min: NewFloatPoint(0), Max: NewFloatPoint(4), PieceCount: 4,
			ShowNoData: true,
		},
	}
	p, err := ChoroplethRender(opt)
	assert.Nil(err)
	assert.NotNil(p)
}

func TestChoroplethTwoSquaresHalfAndColors(t *testing.T) {
	assert := assert.New(t)
	// two adjacent unit squares, values 1 and 3
	opt := ChoroplethChartOption{
		Width:       220,
		Height:      268,
		Padding:     Box{Top: 10, Right: 10, Bottom: 10, Left: 10},
		BorderWidth: 0,
		Regions: []MapRegion{
			{Name: "A", Polygon: unitSquare(0), Value: choroplethValue(1)},
			{Name: "B", Polygon: unitSquare(1), Value: choroplethValue(3)},
		},
		// two classes so 1 and 3 are assigned different color levels
		VisualMap: VisualMap{
			Min: NewFloatPoint(0), Max: NewFloatPoint(4), PieceCount: 2,
		},
	}
	img := decodePNG(t, mustRender(t, opt))
	// plot area: x 10..210 (200px), y 10..210 (200px)
	colorA := img.At(15, 15).(color.RGBA)
	colorB := img.At(195, 15).(color.RGBA)
	assert.NotEqual(colorA, colorB)

	cntA, cntB := 0, 0
	for y := 10; y < 210; y++ {
		for x := 10; x < 210; x++ {
			c := img.At(x, y).(color.RGBA)
			if c == colorA {
				cntA++
			}
			if c == colorB {
				cntB++
			}
		}
	}
	// each square occupies half of the plot area
	assert.Equal(cntA, cntB)
	assert.InDelta(0.5, float64(cntA)/float64(cntA+cntB), 0.001)
}

func TestChoroplethHoleKeepsBackground(t *testing.T) {
	assert := assert.New(t)
	region := MapRegion{
		Name:    "A",
		Polygon: unitSquare(0),
		// hole of area 1/4 in the middle
		Holes: [][]Point2D{
			{{0.25, 0.25}, {0.75, 0.25}, {0.75, 0.75}, {0.25, 0.75}},
		},
		Value: choroplethValue(2),
	}
	opt := ChoroplethChartOption{
		Width:       220,
		Height:      268,
		Padding:     Box{Top: 10, Right: 10, Bottom: 10, Left: 10},
		BorderWidth: 0,
		Regions:     []MapRegion{region},
		VisualMap: VisualMap{
			Min: NewFloatPoint(0), Max: NewFloatPoint(4), PieceCount: 4,
		},
	}
	img := decodePNG(t, mustRender(t, opt))
	fill := img.At(15, 15).(color.RGBA)
	// a point well inside the hole must keep the background white
	holeCenter := img.At(105, 105).(color.RGBA)
	assert.Equal(uint8(255), holeCenter.R)
	assert.Equal(uint8(255), holeCenter.G)
	assert.Equal(uint8(255), holeCenter.B)

	// the hole occupies roughly 1/4 of the region pixels (allow anti-alias)
	fillCount, whiteCount := 0, 0
	for y := 10; y < 210; y++ {
		for x := 10; x < 210; x++ {
			c := img.At(x, y).(color.RGBA)
			if c == fill {
				fillCount++
			}
			if c.R == 255 && c.G == 255 && c.B == 255 {
				whiteCount++
			}
		}
	}
	// colored area is close to 3/4 of the whole region (exact in vector geometry)
	ratio := float64(fillCount) / float64(fillCount+whiteCount)
	assert.InDelta(0.75, ratio, 0.06)
}

func TestChoroplethSVGAndPNGWinding(t *testing.T) {
	assert := assert.New(t)
	// outer ring and hole are supplied with the same winding; the renderer
	// must reverse the hole so both renderers punch it out
	region := MapRegion{
		Name: "A",
		Polygon: []Point2D{
			{0, 0}, {0, 1}, {1, 1}, {1, 0},
		},
		Holes: [][]Point2D{
			{{0.25, 0.25}, {0.25, 0.75}, {0.75, 0.75}, {0.75, 0.25}},
		},
		Value: choroplethValue(2),
	}
	base := ChoroplethChartOption{
		Width:       220,
		Height:      268,
		Padding:     Box{Top: 10, Right: 10, Bottom: 10, Left: 10},
		BorderWidth: 0,
		Regions:     []MapRegion{region},
		VisualMap: VisualMap{
			Min: NewFloatPoint(0), Max: NewFloatPoint(4), PieceCount: 4,
		},
	}
	// PNG: hole center is background
	pngOpt := base
	pngImg := decodePNG(t, mustRender(t, pngOpt))
	holePNG := pngImg.At(105, 105).(color.RGBA)
	assert.Equal(uint8(255), holePNG.R)

	// SVG: the single fill path contains two subpaths
	svgOpt := base
	svgOpt.Type = ChartOutputSVG
	sp, err := ChoroplethRender(svgOpt)
	assert.Nil(err)
	svgBytes, err := sp.Bytes()
	assert.Nil(err)
	svg := string(svgBytes)
	// the fill path (not stroke) should include both an outer and a hole ring
	assert.Contains(svg, "Z\nM")
}

func TestChoroplethCanvasClipping(t *testing.T) {
	assert := assert.New(t)
	// coordinates spanning 1e12 must be clipped instead of panicking
	opt := ChoroplethChartOption{
		Width:  100,
		Height: 140,
		Regions: []MapRegion{
			{
				Name: "big",
				Polygon: []Point2D{
					{0, 0}, {1e12, 0}, {1e12, 1e12}, {0, 1e12},
				},
				Value: choroplethValue(1),
			},
		},
		VisualMap: VisualMap{Min: NewFloatPoint(0), Max: NewFloatPoint(4), PieceCount: 4},
	}
	p, err := ChoroplethRender(opt)
	assert.Nil(err)
	assert.NotNil(p)

	opt.Type = ChartOutputSVG
	sp, err := ChoroplethRender(opt)
	assert.Nil(err)
	assert.NotNil(sp)
}

func TestChoroplethPieceColorAndNormalize(t *testing.T) {
	assert := assert.New(t)
	pieces, err := normalizeVisualMap(&VisualMap{
		Min: NewFloatPoint(0), Max: NewFloatPoint(4), PieceCount: 4,
	}, 0, 4)
	assert.Nil(err)
	assert.Len(pieces, 4)
	// value 1 -> first piece, value 3 -> third piece (different levels)
	c1, ok := pieceColor(pieces, 1)
	assert.True(ok)
	c3, ok := pieceColor(pieces, 3)
	assert.True(ok)
	assert.NotEqual(c1, c3)
	// the max value is included in the last piece
	last, ok := pieceColor(pieces, 4)
	assert.True(ok)
	assert.Equal(pieces[3].Color, last)
	// an out of range value matches no piece
	_, ok = pieceColor(pieces, 99)
	assert.False(ok)

	// custom pieces keep their labels and colors
	custom := []VisualMapPiece{
		{Min: 0, Max: 2, Label: "low", Color: parseColor("#112233")},
		{Min: 2, Max: 4, Label: "high", Color: parseColor("#445566")},
	}
	got, err := normalizeVisualMap(&VisualMap{Pieces: custom}, 0, 4)
	assert.Nil(err)
	assert.Equal("low", got[0].Label)
	assert.Equal(parseColor("#112233"), got[0].Color)
}

func TestChoroplethHoleGeometryExact(t *testing.T) {
	assert := assert.New(t)
	// exact vector geometry, independent of raster anti-aliasing
	outer := unitSquare(0)
	hole := []Point2D{{0.25, 0.25}, {0.75, 0.25}, {0.75, 0.75}, {0.25, 0.75}}
	outerArea := math.Abs(ringSignedArea(outer))
	holeArea := math.Abs(ringSignedArea(hole))
	// the hole covers exactly one quarter of the region
	assert.InDelta(0.25, holeArea/outerArea, 1e-12)
	// therefore the colored area (outer minus hole) is exactly three quarters
	assert.InDelta(0.75, (outerArea-holeArea)/outerArea, 1e-12)

	// two unit squares have equal area, so each fills half of the drawing area
	left := unitSquare(0)
	right := unitSquare(1)
	assert.InDelta(math.Abs(ringSignedArea(left)), math.Abs(ringSignedArea(right)), 1e-12)
}

func TestChoroplethRingHelpers(t *testing.T) {
	assert := assert.New(t)
	// counter clockwise square has positive signed area
	ccw := []Point2D{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
	assert.True(ringSignedArea(ccw) > 0)
	cw := []Point2D{{0, 0}, {0, 1}, {1, 1}, {1, 0}}
	assert.True(ringSignedArea(cw) < 0)

	// centroid of a unit square is its center
	c := ringCentroid(ccw)
	assert.InDelta(0.5, c.X, 1e-9)
	assert.InDelta(0.5, c.Y, 1e-9)

	// clipping keeps values inside the safe range
	assert.Equal(maxCanvasCoord, clampCanvasInt(1e15))
	assert.Equal(-maxCanvasCoord, clampCanvasInt(-1e15))
	assert.Equal(0, clampCanvasInt(math.NaN()))
}

func TestChoroplethPointInRingAndLabel(t *testing.T) {
	assert := assert.New(t)
	square := []Point2D{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
	assert.True(pointInRing(Point2D{0.5, 0.5}, square))
	assert.False(pointInRing(Point2D{2, 2}, square))
	assert.False(pointInRing(Point2D{-0.1, 0.5}, square))

	hole := [][]Point2D{{{0.3, 0.3}, {0.7, 0.3}, {0.7, 0.7}, {0.3, 0.7}}}
	assert.True(pointInHole(Point2D{0.5, 0.5}, hole))
	assert.False(pointInHole(Point2D{0.1, 0.1}, hole))

	// the label point must stay outside the hole even though the outer
	// centroid (0.5, 0.5) is inside it
	region := MapRegion{
		Name:    "A",
		Polygon: square,
		Holes:   hole,
	}
	label := regionLabelPoint(region)
	assert.True(pointInRing(label, square))
	assert.False(pointInHole(label, hole))

	// without a hole the label is the centroid
	plain := MapRegion{Name: "B", Polygon: square}
	label2 := regionLabelPoint(plain)
	assert.InDelta(0.5, label2.X, 1e-9)
	assert.InDelta(0.5, label2.Y, 1e-9)
}

func mustRender(t *testing.T, opt ChoroplethChartOption) *Painter {
	p, err := ChoroplethRender(opt)
	assert.Nil(t, err)
	return p
}
