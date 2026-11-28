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
	"image/png"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newSlopeTestOption() SlopeChartOption {
	return SlopeChartOption{
		Width:       600,
		Height:      400,
		Padding:     Box{Left: 20, Right: 20, Top: 20, Bottom: 20},
		LabelShow:   true,
		DivideCount: 6,
		Min:         NewFloatPoint(0),
		Max:         NewFloatPoint(6),
		Data: []SlopeChartData{
			{Name: "A", Left: 1, Right: 5},
			{Name: "B", Left: 4, Right: 2},
		},
	}
}

// segmentIntersection returns the intersection point of two segments,
// ok is false when the segments are parallel or do not cross inside them.
func segmentIntersection(p1, p2, p3, p4 Point) (x, y float64, ok bool) {
	x1, y1 := float64(p1.X), float64(p1.Y)
	x2, y2 := float64(p2.X), float64(p2.Y)
	x3, y3 := float64(p3.X), float64(p3.Y)
	x4, y4 := float64(p4.X), float64(p4.Y)
	den := (x1-x2)*(y3-y4) - (y1-y2)*(x3-x4)
	if den == 0 {
		return 0, 0, false
	}
	t := ((x1-x3)*(y3-y4) - (y1-y3)*(x3-x4)) / den
	u := -((x1-x2)*(y1-y3) - (y1-y2)*(x1-x3)) / den
	if t < 0 || t > 1 || u < 0 || u > 1 {
		return 0, 0, false
	}
	return x1 + t*(x2-x1), y1 + t*(y2-y1), true
}

func TestSlopeChartGeometry(t *testing.T) {
	assert := assert.New(t)
	opt := newSlopeTestOption()
	err := opt.fillDefault()
	assert.Nil(err)

	p, err := NewPainter(PainterOptions{
		Type:   ChartOutputSVG,
		Width:  opt.Width,
		Height: opt.Height,
		Font:   opt.font,
	})
	assert.Nil(err)
	c := NewSlopeChart(p, opt)
	content := p.Child(PainterPaddingOption(opt.Padding))
	layout, err := c.layout(content)
	assert.Nil(err)

	r := layout.r
	assert.Equal(0.0, r.min)
	assert.Equal(6.0, r.max)
	height := layout.height
	expectedY := func(v float64) int {
		return height - int((v-r.min)/(r.max-r.min)*float64(height))
	}

	// A: 1 -> 5, B: 4 -> 2, the input order is kept without sorting
	assert.Equal([]string{"A", "B"}, []string{
		layout.rows[0].data.Name,
		layout.rows[1].data.Name,
	})
	// The endpoints must land at the real value converted pixel positions
	assert.Equal(expectedY(1), layout.leftPoints[0].Y)
	assert.Equal(expectedY(5), layout.rightPoints[0].Y)
	assert.Equal(expectedY(4), layout.leftPoints[1].Y)
	assert.Equal(expectedY(2), layout.rightPoints[1].Y)
	assert.Equal(layout.xLeft, layout.leftPoints[0].X)
	assert.Equal(layout.xRight, layout.rightPoints[0].X)
	assert.False(layout.leftPoints[0].Missing)

	// A goes up and B goes down, the two lines cross at value 3
	a1 := Point{X: layout.xLeft, Y: layout.leftPoints[0].Y}
	a2 := Point{X: layout.xRight, Y: layout.rightPoints[0].Y}
	b1 := Point{X: layout.xLeft, Y: layout.leftPoints[1].Y}
	b2 := Point{X: layout.xRight, Y: layout.rightPoints[1].Y}
	x, y, ok := segmentIntersection(a1, a2, b1, b2)
	assert.True(ok)
	// The crossing happens halfway between the two columns
	assert.InDelta(float64(layout.xLeft)+float64(layout.xRight-layout.xLeft)/2, x, 1.5)
	assert.InDelta(float64(expectedY(3)), y, 1.5)

	// Equal values produce a horizontal segment
	flat := []SlopeChartData{
		{Name: "C", Left: 3, Right: 3},
	}
	flatOpt := SlopeChartOption{
		Width:       400,
		Height:      300,
		Padding:     Box{Left: 20, Right: 20, Top: 20, Bottom: 20},
		DivideCount: 6,
		Data:        flat,
	}
	assert.Nil(flatOpt.fillDefault())
	flatPainter, err := NewPainter(PainterOptions{
		Type:   ChartOutputSVG,
		Width:  400,
		Height: 300,
		Font:   flatOpt.font,
	})
	assert.Nil(err)
	flatLayout, err := NewSlopeChart(flatPainter, flatOpt).layout(flatPainter.Child(PainterPaddingOption(flatOpt.Padding)))
	assert.Nil(err)
	assert.Equal(flatLayout.leftPoints[0].Y, flatLayout.rightPoints[0].Y)
}

func TestSlopeChartDegenerateRange(t *testing.T) {
	assert := assert.New(t)
	// All equal values would produce a zero-sized range, it must be padded
	// instead of collapsing every endpoint to the same pixel
	for _, values := range [][]float64{{0, 0}, {3, 3}} {
		opt := SlopeChartOption{
			Width:       400,
			Height:      300,
			Padding:     Box{Left: 20, Right: 20, Top: 20, Bottom: 20},
			DivideCount: 6,
			Data: []SlopeChartData{
				{Name: "A", Left: values[0], Right: values[1]},
			},
		}
		assert.Nil(opt.fillDefault())
		p, err := NewPainter(PainterOptions{
			Type:   ChartOutputSVG,
			Width:  400,
			Height: 300,
			Font:   opt.font,
		})
		assert.Nil(err)
		layout, err := NewSlopeChart(p, opt).layout(p.Child(PainterPaddingOption(opt.Padding)))
		assert.Nil(err)
		assert.True(layout.r.max > layout.r.min)
		_, renderErr := NewSlopeChart(p, opt).Render()
		assert.Nil(renderErr)
	}
}

func TestSlopeChartSort(t *testing.T) {
	assert := assert.New(t)
	data := []SlopeChartData{
		{Name: "A", Left: 2, Right: 5},
		{Name: "B", Left: 2, Right: 1},
		{Name: "C", Left: 4, Right: 3},
		{Name: "D", Left: math.NaN(), Right: 2},
	}
	opt := SlopeChartOption{
		Width:       400,
		Height:      300,
		Padding:     Box{Left: 20, Right: 20, Top: 20, Bottom: 20},
		DivideCount: 6,
		SortBy:      SlopeSortLeft,
		SortOrder:   SlopeOrderDesc,
		Data:        data,
	}
	assert.Nil(opt.fillDefault())
	p, err := NewPainter(PainterOptions{
		Type:   ChartOutputSVG,
		Width:  400,
		Height: 300,
		Font:   opt.font,
	})
	assert.Nil(err)
	rows := NewSlopeChart(p, opt).sortedRows()
	// C(4) -> A(2) -> B(2), the equal values keep the input order,
	// the missing row is always ranked last
	assert.Equal([]string{"C", "A", "B", "D"}, []string{
		rows[0].data.Name,
		rows[1].data.Name,
		rows[2].data.Name,
		rows[3].data.Name,
	})
	// The original index is kept so the color follows the entity
	assert.Equal(2, rows[0].index)
	assert.Equal(0, rows[1].index)

	opt.SortBy = SlopeSortRight
	opt.SortOrder = SlopeOrderAsc
	rows = NewSlopeChart(p, opt).sortedRows()
	assert.Equal([]string{"B", "D", "C", "A"}, []string{
		rows[0].data.Name,
		rows[1].data.Name,
		rows[2].data.Name,
		rows[3].data.Name,
	})

	// No sort keeps the input order
	opt.SortBy = ""
	rows = NewSlopeChart(p, opt).sortedRows()
	assert.Equal([]string{"A", "B", "C", "D"}, []string{
		rows[0].data.Name,
		rows[1].data.Name,
		rows[2].data.Name,
		rows[3].data.Name,
	})
}

func TestSlopeChartMissingAndNonFinite(t *testing.T) {
	assert := assert.New(t)
	data := []SlopeChartData{
		{Name: "A", Left: 1, Right: 5},
		{Name: "B", Left: math.NaN(), Right: 2},
		{Name: "C", Left: 3, Right: math.Inf(1)},
		{Name: "D", Left: 4, Right: GetNullValue()},
	}
	opt := SlopeChartOption{
		Width:       500,
		Height:      300,
		Padding:     Box{Left: 20, Right: 20, Top: 20, Bottom: 20},
		DivideCount: 6,
		LabelShow:   true,
		Data:        data,
	}
	assert.Nil(opt.fillDefault())
	p, err := NewPainter(PainterOptions{
		Type:   ChartOutputSVG,
		Width:  500,
		Height: 300,
		Font:   opt.font,
	})
	assert.Nil(err)
	layout, err := NewSlopeChart(p, opt).layout(p.Child(PainterPaddingOption(opt.Padding)))
	assert.Nil(err)

	assert.False(layout.leftPoints[0].Missing)
	// NaN on the left, +Inf and null value on the right are missing
	assert.True(layout.leftPoints[1].Missing)
	assert.True(layout.rightPoints[2].Missing)
	assert.True(layout.rightPoints[3].Missing)
	// The missing side has no data coordinate and borrows the opposite
	// endpoint y only for the label anchor
	assert.Equal(layout.rightPoints[1].Y, layout.leftPoints[1].Y)
	// The missing label shows the configured missing text
	findLabel := func(labels []*slopeLabelItem, row int) string {
		for _, item := range labels {
			if item.Row == row {
				return item.text
			}
		}
		return ""
	}
	assert.True(strings.HasSuffix(findLabel(layout.leftLabels, 1), defaultSlopeMissingText))
	assert.True(strings.HasSuffix(findLabel(layout.rightLabels, 2), defaultSlopeMissingText))
	assert.True(strings.HasSuffix(findLabel(layout.rightLabels, 3), defaultSlopeMissingText))
	// The finite labels keep their value text
	assert.Equal("A 5", findLabel(layout.rightLabels, 0))

	// Both values missing is rejected
	bad := newSlopeTestOption()
	bad.Data = []SlopeChartData{
		{Name: "X", Left: math.NaN(), Right: math.Inf(-1)},
	}
	assert.NotNil(bad.fillDefault())
	bad = newSlopeTestOption()
	bad.Data = nil
	assert.NotNil(bad.fillDefault())
	bad = newSlopeTestOption()
	bad.SortBy = "unknown"
	assert.NotNil(bad.fillDefault())
}

func TestAdjustSlopeLabels(t *testing.T) {
	assert := assert.New(t)
	// Five labels anchored at the same position must not overlap, while the
	// anchors keep pointing at the unchanged data coordinates
	anchors := []int{100, 100, 100, 100, 100}
	labels := make([]*slopeLabelItem, len(anchors))
	for index, y := range anchors {
		labels[index] = &slopeLabelItem{
			Row:     index,
			anchorY: y,
			text:    "label",
			textW:   30,
			textH:   12,
		}
	}
	adjustSlopeLabels(labels, 0, 200, 8)
	for index, item := range labels {
		// The anchor is never modified
		assert.Equal(anchors[index], item.anchorY)
		assert.GreaterOrEqual(item.top, 0)
		assert.LessOrEqual(item.top+item.textH, 200)
		if index > 0 {
			assert.GreaterOrEqual(item.top, labels[index-1].top+labels[index-1].textH+8)
		}
	}

	// Labels near the bottom are clamped inside the plot area
	labels = []*slopeLabelItem{
		{Row: 0, anchorY: 195, text: "x", textW: 10, textH: 12},
		{Row: 1, anchorY: 196, text: "y", textW: 10, textH: 12},
	}
	adjustSlopeLabels(labels, 0, 200, 8)
	assert.LessOrEqual(labels[1].top+labels[1].textH, 200)
}

func TestSlopeChartRenderSVGAndPNG(t *testing.T) {
	assert := assert.New(t)
	svgPainter, err := SlopeRender(
		[]SlopeChartData{
			{Name: "A", Left: 1, Right: 5},
			{Name: "B", Left: 4, Right: 2},
		},
		SlopeTypeOptionFunc(ChartOutputSVG),
		SlopeTitleOptionFunc(TitleOption{Text: "Frame Latency Ranking"}),
		SlopeTimeLabelOptionFunc("Before", "After"),
		SlopeLabelShowOptionFunc(true),
		SlopeMinMaxOptionFunc(0, 6),
	)
	assert.Nil(err)
	svgBytes, err := svgPainter.Bytes()
	assert.Nil(err)
	svg := string(svgBytes)
	assert.True(strings.HasPrefix(svg, "<svg"))
	assert.Contains(svg, "Frame Latency Ranking")
	assert.Contains(svg, "Before")
	assert.Contains(svg, "After")
	// Both names are labeled on both sides
	assert.Equal(2, strings.Count(svg, "A "))
	assert.Equal(2, strings.Count(svg, "B "))
	// Four endpoint symbols are drawn (two rows, two time points)
	assert.Equal(4, strings.Count(svg, "<circle"))
	// The connecting lines are drawn with the two theme series colors
	assert.Contains(svg, "rgba(84,112,198,1.0)")
	assert.Contains(svg, "rgba(145,204,117,1.0)")

	// Re-render the same data through the standalone renderer to get the
	// expected absolute endpoint coordinates and verify the SVG circles use
	// exactly those coordinates
	opt := newSlopeTestOption()
	assert.Nil(opt.fillDefault())
	p, err := NewPainter(PainterOptions{
		Type:   ChartOutputSVG,
		Width:  opt.Width,
		Height: opt.Height,
		Font:   opt.font,
	})
	assert.Nil(err)
	layout, err := NewSlopeChart(p, opt).layout(p.Child(PainterPaddingOption(opt.Padding)))
	assert.Nil(err)
	_, err = NewSlopeChart(p, opt).Render()
	assert.Nil(err)
	rendered, err := p.Bytes()
	assert.Nil(err)
	renderedSVG := string(rendered)
	for _, point := range append(append([]slopeEndpoint{}, layout.leftPoints...), layout.rightPoints...) {
		cx := layout.plotBox.Left + point.X
		cy := layout.plotBox.Top + point.Y
		assert.Contains(renderedSVG, "cx=\""+strconv.Itoa(cx)+"\" cy=\""+strconv.Itoa(cy)+"\"")
	}

	// PNG output is a valid non-blank image with the same size
	pngPainter, err := SlopeRender(
		[]SlopeChartData{
			{Name: "A", Left: 1, Right: 5},
			{Name: "B", Left: 4, Right: 2},
		},
		SlopeTypeOptionFunc(ChartOutputPNG),
		SlopeMinMaxOptionFunc(0, 6),
	)
	assert.Nil(err)
	pngBytes, err := pngPainter.Bytes()
	assert.Nil(err)
	assert.True(bytes.HasPrefix(pngBytes, []byte{0x89, 'P', 'N', 'G'}))
	img, err := png.Decode(bytes.NewReader(pngBytes))
	assert.Nil(err)
	assert.Equal(600, img.Bounds().Dx())
	assert.Equal(400, img.Bounds().Dy())
	// The chart draws lines and text, so the image can not be all background
	whiteCount := 0
	totalCount := 0
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			totalCount++
			if r>>8 == 255 && g>>8 == 255 && b>>8 == 255 {
				whiteCount++
			}
		}
	}
	assert.Less(float64(whiteCount)/float64(totalCount), 0.99)
}

func TestSlopeChartColorsAndLabels(t *testing.T) {
	assert := assert.New(t)
	red := parseColor("#ee6666")
	blue := parseColor("#5470c6")
	p, err := SlopeRender(
		[]SlopeChartData{
			{Name: "A", Left: 1, Right: 5, Style: Style{StrokeColor: red}},
			{Name: "B", Left: 4, Right: 2},
		},
		SlopeTypeOptionFunc(ChartOutputSVG),
		SlopeColorsOptionFunc(red, blue),
		SlopeLegendOptionFunc(NewLegendOption([]string{"A", "B"})),
		SlopeMinMaxOptionFunc(0, 6),
	)
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)
	assert.Contains(svg, "rgba(238,102,102,1.0)")
	assert.Contains(svg, "rgba(84,112,198,1.0)")
}
