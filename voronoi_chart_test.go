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
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func voronoiTestSites() []VoronoiSite {
	return []VoronoiSite{
		{X: 0, Y: 0, Name: "alpha"},
		{X: 2, Y: 0, Name: "beta"},
	}
}

func TestVoronoiRenderSVG(t *testing.T) {
	assert := assert.New(t)
	p, err := VoronoiRender(voronoiTestSites(), VoronoiRect{
		X0: 0, Y0: -1, X1: 2, Y1: 1,
	},
		TitleTextOptionFunc("Bounded Voronoi"),
		LegendLabelsOptionFunc([]string{"alpha", "beta"}),
	)
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)
	assert.True(strings.HasPrefix(svg, "<svg"))
	assert.Contains(svg, "</svg>")
	// The title and the legend labels are rendered.
	assert.Contains(svg, "Bounded Voronoi")
	assert.Contains(svg, "alpha")
	assert.Contains(svg, "beta")
	// Two filled regions and their borders are rendered.
	assert.GreaterOrEqual(strings.Count(svg, "<path"), 2)
}

func TestVoronoiOptionRenderPNG(t *testing.T) {
	assert := assert.New(t)
	p, err := VoronoiOptionRender(VoronoiChartOption{
		Type:   ChartOutputPNG,
		Width:  500,
		Height: 300,
		Sites: []VoronoiSite{
			{X: 0, Y: 0, Name: "alpha"},
			{X: 2, Y: 0, Name: "beta"},
			{X: 1, Y: 1, Name: "gamma"},
		},
		Rect: VoronoiRect{
			X0: 0, Y0: -1, X1: 2, Y1: 1,
		},
		ShowBorder: true,
		ShowSite:   true,
		ShowLabel:  true,
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	// PNG signature.
	assert.True(bytes.HasPrefix(buf, []byte{0x89, 0x50, 0x4E, 0x47}))
	assert.Equal(500, p.Width())
	assert.Equal(300, p.Height())
}

func TestVoronoiOptionRenderSVG(t *testing.T) {
	assert := assert.New(t)
	p, err := VoronoiOptionRender(VoronoiChartOption{
		Type: ChartOutputSVG,
		Sites: []VoronoiSite{
			{X: 0, Y: 0, Name: "alpha"},
			{X: 2, Y: 0, Name: "beta"},
		},
		Rect: VoronoiRect{
			X0: 0, Y0: -1, X1: 2, Y1: 1,
		},
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	assert.True(strings.HasPrefix(string(buf), "<svg"))
}

func TestVoronoiRenderCustomColors(t *testing.T) {
	assert := assert.New(t)
	p, err := VoronoiOptionRender(VoronoiChartOption{
		Type: ChartOutputSVG,
		Sites: []VoronoiSite{
			{
				X: 0, Y: 0, Name: "alpha",
				FillColor:   parseColor("#ff0000"),
				StrokeColor: parseColor("#00ff00"),
			},
			{X: 2, Y: 0, Name: "beta"},
		},
		Rect: VoronoiRect{
			X0: 0, Y0: -1, X1: 2, Y1: 1,
		},
		ShowBorder:  true,
		BorderColor: parseColor("#0000ff"),
		BorderWidth: 2,
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)
	assert.Contains(svg, "rgba(255,0,0,")
	assert.Contains(svg, "rgba(0,255,0,1.0)")
	assert.Contains(svg, "rgba(0,0,255,1.0)")
}

func TestVoronoiRenderInvalid(t *testing.T) {
	assert := assert.New(t)
	// Non finite site coordinates.
	_, err := VoronoiOptionRender(VoronoiChartOption{
		Type: ChartOutputSVG,
		Sites: []VoronoiSite{
			{X: 0, Y: 0},
			{X: 2, Y: 0},
			{X: 1, Y: math.NaN()},
		},
		Rect: VoronoiRect{X0: 0, Y0: -1, X1: 2, Y1: 1},
	})
	assert.NotNil(err)
	// Invalid rectangle.
	_, err = VoronoiOptionRender(VoronoiChartOption{
		Type:  ChartOutputSVG,
		Sites: voronoiTestSites(),
		Rect:  VoronoiRect{X0: 2, Y0: -1, X1: 2, Y1: 1},
	})
	assert.NotNil(err)
}

func TestVoronoiRenderDuplicatesAndEmpty(t *testing.T) {
	assert := assert.New(t)
	// Duplicate sites keep a deterministic output without overlapping text.
	p, err := VoronoiOptionRender(VoronoiChartOption{
		Type: ChartOutputSVG,
		Sites: []VoronoiSite{
			{X: 0, Y: 0, Name: "alpha"},
			{X: 0, Y: 0, Name: "alpha-dup"},
			{X: 2, Y: 0, Name: "beta"},
		},
		Rect:      VoronoiRect{X0: 0, Y0: -1, X1: 2, Y1: 1},
		ShowLabel: true,
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	assert.Contains(string(buf), "<svg")

	// An empty site list renders the background only and never errors.
	p, err = VoronoiOptionRender(VoronoiChartOption{
		Type:  ChartOutputPNG,
		Sites: nil,
		Rect:  VoronoiRect{X0: 0, Y0: 0, X1: 1, Y1: 1},
	})
	assert.Nil(err)
	buf, err = p.Bytes()
	assert.Nil(err)
	assert.True(bytes.HasPrefix(buf, []byte{0x89, 0x50, 0x4E, 0x47}))
}

func TestVoronoiToPixel(t *testing.T) {
	assert := assert.New(t)
	p, err := NewPainter(PainterOptions{
		Type:   ChartOutputSVG,
		Width:  200,
		Height: 100,
	}, PainterPaddingOption(Box{
		Left: 0, Top: 0, Right: 0, Bottom: 0,
	}))
	assert.Nil(err)
	chart := NewVoronoiChart(p, VoronoiChartOption{
		Rect: VoronoiRect{X0: 0, Y0: 0, X1: 2, Y1: 2},
	})
	// Bottom left corner maps to the origin, top right to the far corner.
	assert.Equal(Point{X: 0, Y: 100}, chart.toPixel(p, VPoint{X: 0, Y: 0}))
	assert.Equal(Point{X: 200, Y: 0}, chart.toPixel(p, VPoint{X: 2, Y: 2}))
	assert.Equal(Point{X: 100, Y: 50}, chart.toPixel(p, VPoint{X: 1, Y: 1}))
}

func TestNewVoronoiSeriesList(t *testing.T) {
	assert := assert.New(t)
	list := NewVoronoiSeriesList(voronoiTestSites())
	assert.Len(list, 2)
	assert.Equal(ChartTypeVoronoi, list[0].Type)
	assert.Equal("alpha", list[0].Name)
	assert.Equal("beta", list[1].Name)
}
