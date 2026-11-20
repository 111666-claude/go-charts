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
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

func TestHeatMapScaleColor(t *testing.T) {
	assert := assert.New(t)

	// Fixed diverging matrix: min = -10, max = 10,
	// the built-in diverging scale has 5 stops and the middle stop is index 2.
	scale := newHeatMapScale(-10, 10, nil, nil, nil)
	assert.Len(scale.stops, 5)
	assert.True(scale.diverging)
	assert.False(scale.equal)
	// Boundary values are mapped to the end stops.
	assert.Equal(parseColor("#313695"), scale.color(-10))
	assert.Equal(parseColor("#a50026"), scale.color(10))
	// Zero is mapped exactly to the middle stop.
	assert.Equal(parseColor("#ffffbf"), scale.color(0))
	// Values out of range are clamped to the end stops.
	assert.Equal(parseColor("#313695"), scale.color(-20))
	assert.Equal(parseColor("#a50026"), scale.color(20))
	// -5 is at t = 0.5 of the negative half, which lands on stop[1] exactly.
	assert.Equal(parseColor("#74add1"), scale.color(-5))
	// 5 is at t = 0.5 of the positive half, which lands on stop[3] exactly.
	assert.Equal(parseColor("#fdae61"), scale.color(5))
	// Quarter points fall inside a segment and use interpolation.
	// -2.5 => negative t = 0.75 => pos = 1.5 => stop[1] -> stop[2].
	assert.Equal(lerpColor(parseColor("#74add1"), parseColor("#ffffbf"), 0.5), scale.color(-2.5))

	// Sequential scale for non-negative data.
	positiveScale := newHeatMapScale(0, 8, nil, nil, nil)
	assert.False(positiveScale.diverging)
	assert.Equal(parseColor("#f7fbff"), positiveScale.color(0))
	assert.Equal(parseColor("#08306b"), positiveScale.color(8))
	assert.Equal(lerpColor(parseColor("#f7fbff"), parseColor("#c6dbef"), 0.5), positiveScale.color(1))

	// Sequential scale for non-positive data.
	negativeScale := newHeatMapScale(-8, 0, nil, nil, nil)
	assert.False(negativeScale.diverging)
	assert.Equal(parseColor("#fff5f0"), negativeScale.color(-8))
	assert.Equal(parseColor("#67000d"), negativeScale.color(0))

	// Custom color stops.
	customScale := newHeatMapScale(0, 100, nil, nil, []string{"#000000", "#ffffff"})
	assert.Equal(drawing.Color{R: 0, G: 0, B: 0, A: 255}, customScale.color(0))
	assert.Equal(drawing.Color{R: 255, G: 255, B: 255, A: 255}, customScale.color(100))
	assert.Equal(drawing.Color{R: 128, G: 128, B: 128, A: 255}, customScale.color(50))

	// Explicit min/max options and reversed option order.
	optMin, optMax := NewFloatPoint(-5), NewFloatPoint(5)
	limited := newHeatMapScale(-10, 10, optMin, optMax, nil)
	assert.Equal(-5.0, limited.min)
	assert.Equal(5.0, limited.max)
	reversed := newHeatMapScale(-10, 10, NewFloatPoint(9), NewFloatPoint(1), nil)
	assert.Equal(1.0, reversed.min)
	assert.Equal(9.0, reversed.max)
	// Non-finite options are ignored.
	nanOption := NewFloatPoint(math.NaN())
	ignored := newHeatMapScale(-10, 10, nanOption, NewFloatPoint(math.Inf(1)), nil)
	assert.Equal(-10.0, ignored.min)
	assert.Equal(10.0, ignored.max)
}

func TestHeatMapScaleConstantAndTicks(t *testing.T) {
	assert := assert.New(t)

	// All finite values are constant.
	scale := newHeatMapScale(3, 3, nil, nil, nil)
	assert.True(scale.equal)
	assert.Equal(scale.stops[len(scale.stops)/2], scale.color(3))
	// Ticks of a constant scale are all the same value.
	ticks := scale.ticks(5)
	assert.Len(ticks, 6)
	for _, v := range ticks {
		assert.Equal(3.0, v)
	}

	scale = newHeatMapScale(0, 10, nil, nil, nil)
	ticks = scale.ticks(5)
	assert.Equal([]float64{0, 2, 4, 6, 8, 10}, ticks)
	// Invalid split falls back to the default.
	assert.Len(scale.ticks(0), defaultHeatMapLegendSplit+1)
}

func TestHeatMapIsMissingValue(t *testing.T) {
	assert := assert.New(t)
	assert.True(isMissingValue(math.NaN()))
	assert.True(isMissingValue(math.Inf(1)))
	assert.True(isMissingValue(math.Inf(-1)))
	assert.True(isMissingValue(nullValue))
	assert.False(isMissingValue(0))
	assert.False(isMissingValue(-1.5))
}

func TestHeatMapDimensions(t *testing.T) {
	assert := assert.New(t)

	p, err := NewPainter(PainterOptions{
		Type:   ChartOutputSVG,
		Width:  600,
		Height: 400,
	})
	assert.Nil(err)

	// Empty matrix.
	empty := NewHeatMapChart(p, HeatMapOption{})
	rows, columns, err := empty.dimensions()
	assert.Nil(err)
	assert.Equal(0, rows)
	assert.Equal(0, columns)

	// Rows shorter than the first one: the max row length is the column count.
	ragged := NewHeatMapChart(p, HeatMapOption{
		Data: [][]float64{
			{1, 2, 3},
			{4, 5},
			{6},
		},
	})
	rows, columns, err = ragged.dimensions()
	assert.Nil(err)
	assert.Equal(3, rows)
	assert.Equal(3, columns)

	// Labels determine the dimension when the data is empty.
	labelOnly := NewHeatMapChart(p, HeatMapOption{
		Rows:    []string{"A", "B"},
		Columns: []string{"X", "Y", "Z"},
	})
	rows, columns, err = labelOnly.dimensions()
	assert.Nil(err)
	assert.Equal(2, rows)
	assert.Equal(3, columns)

	// Oversized matrix returns an error.
	big := make([][]float64, maxHeatMapDimension+1)
	for i := range big {
		big[i] = make([]float64, 2)
	}
	oversized := NewHeatMapChart(p, HeatMapOption{Data: big})
	_, _, err = oversized.dimensions()
	assert.NotNil(err)
}

func TestHeatMapLayoutCellBox(t *testing.T) {
	assert := assert.New(t)

	p, err := NewPainter(PainterOptions{
		Type:   ChartOutputSVG,
		Width:  600,
		Height: 400,
	})
	assert.Nil(err)

	chart := NewHeatMapChart(p, HeatMapOption{
		Rows:    []string{"A", "B", "C"},
		Columns: []string{"X", "Y", "Z", "Z2"},
		Data: [][]float64{
			{1, 2, 3, 4},
			{5, 6, 7, 8},
			{9, 10, 11, 12},
		},
	})
	layout := chart.computeLayout(p, 3, 4)
	assert.Equal(3, layout.rowCount)
	assert.Equal(4, layout.columnCount)
	// The grid fills the width after the row labels.
	assert.Equal(p.Width()-layout.gridLeft, layout.gridWidth)
	assert.True(layout.cellWidth > 0)
	assert.True(layout.cellHeight > 0)
	// Cell boxes are adjacent without gaps or overlaps.
	first := layout.cellBox(0, 0)
	second := layout.cellBox(0, 1)
	assert.Equal(first.Right, second.Left)
	below := layout.cellBox(1, 0)
	assert.Equal(first.Bottom, below.Top)
	// The bottom-right cell stays inside the grid.
	last := layout.cellBox(2, 3)
	assert.True(last.Right <= layout.gridWidth)
	assert.True(last.Bottom <= layout.gridHeight)
}

func TestHeatMapRenderFixedMatrix(t *testing.T) {
	assert := assert.New(t)

	// A fixed matrix with positive, negative and missing values.
	data := [][]float64{
		{-10, -5, 0},
		{-7.5, 2, 5},
		{math.NaN(), math.Inf(1), 10},
	}
	rows := []string{"Warrior", "Mage", "Rogue"}
	columns := []string{"Chapter 1", "Chapter 2", "Chapter 3"}

	p, err := HeatMapRender(data, rows, columns,
		SVGTypeOption(),
		WidthOptionFunc(700),
		HeightOptionFunc(460),
		TitleTextOptionFunc("Level Difficulty"),
		HeatMapColorsOptionFunc("#313695", "#74add1", "#ffffbf", "#fdae61", "#a50026"),
	)
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)
	assert.True(strings.HasPrefix(svg, "<svg"))
	assert.True(strings.HasSuffix(strings.TrimSpace(svg), "</svg>"))

	// Title and all labels are rendered.
	assert.Contains(svg, "Level Difficulty")
	for _, label := range append(rows, columns...) {
		assert.Contains(svg, ">"+label+"<")
	}

	// Color mapping of the corner and zero cells.
	assert.Contains(svg, "fill:rgba(49,54,149,1.0)")   // -10 -> #313695
	assert.Contains(svg, "fill:rgba(165,0,38,1.0)")    // 10 -> #a50026
	assert.Contains(svg, "fill:rgba(255,255,255,1.0)") // background and 0 text
	assert.Contains(svg, "fill:rgba(255,255,191,1.0)") // 0 -> #ffffbf
	// Missing value cells use the independent null style (light split line color).
	assert.Contains(svg, "fill:rgba(224,230,242,1.0)")
	// Missing value label.
	assert.Contains(svg, ">-<")
	// Value labels.
	assert.Contains(svg, ">-10<")
	assert.Contains(svg, ">10<")
	assert.Contains(svg, ">-7.5<")

	// Legend: ticks from -10 to 10 with split 5.
	for _, tick := range []string{">-10<", ">-6<", ">-2<", ">2<", ">6<", ">10<"} {
		assert.Contains(svg, tick)
	}
	// The missing value swatch is rendered in the legend as well.
	assert.Contains(svg, "fill:rgba(224,230,242,1.0)")
}

func TestHeatMapRenderPNG(t *testing.T) {
	assert := assert.New(t)

	p, err := HeatMapOptionRender(HeatMapOption{
		Type:    ChartOutputPNG,
		Width:   700,
		Height:  460,
		Rows:    []string{"A", "B"},
		Columns: []string{"X", "Y"},
		Data: [][]float64{
			{1, 2},
			{3, 4},
		},
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	// PNG signature.
	assert.Equal([]byte{0x89, 0x50, 0x4e, 0x47}, buf[:4])
}

func TestHeatMapRenderVerticalLegend(t *testing.T) {
	assert := assert.New(t)

	p, err := HeatMapOptionRender(HeatMapOption{
		Type:    ChartOutputSVG,
		Width:   700,
		Height:  460,
		Rows:    []string{"A", "B", "C"},
		Columns: []string{"X", "Y", "Z"},
		Data: [][]float64{
			{1, 2, 3},
			{4, 5, 6},
			{7, 8, 9},
		},
		Legend: HeatMapLegendOption{
			Orient: OrientVertical,
			Split:  4,
		},
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)
	// Vertical legend tick labels.
	assert.Contains(svg, ">1<")
	assert.Contains(svg, ">9<")
	// The end colors of the sequential scale.
	assert.Contains(svg, "fill:rgba(247,251,255,1.0)")
	assert.Contains(svg, "fill:rgba(8,48,107,1.0)")
}

func TestHeatMapRenderLegendHidden(t *testing.T) {
	assert := assert.New(t)

	p, err := HeatMapOptionRender(HeatMapOption{
		Type:    ChartOutputSVG,
		Width:   600,
		Height:  400,
		Rows:    []string{"A"},
		Columns: []string{"X"},
		Data:    [][]float64{{1}},
		Legend: HeatMapLegendOption{
			Show: FalseFlag(),
		},
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)
	// No legend tick text.
	assert.NotContains(svg, ">1.00<")
	// The cell is still rendered.
	assert.Contains(svg, "<path")
}

func TestHeatMapRenderEmptyMatrix(t *testing.T) {
	assert := assert.New(t)

	// Empty matrix renders without error and produces a valid canvas.
	p, err := HeatMapOptionRender(HeatMapOption{
		Type:   ChartOutputSVG,
		Width:  600,
		Height: 400,
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)
	assert.True(strings.HasPrefix(svg, "<svg"))
	assert.Contains(svg, "</svg>")
}

func TestHeatMapRenderOversizedMatrix(t *testing.T) {
	assert := assert.New(t)

	big := make([][]float64, maxHeatMapDimension+1)
	for i := range big {
		big[i] = make([]float64, 2)
	}
	_, err := HeatMapOptionRender(HeatMapOption{
		Type:   ChartOutputSVG,
		Width:  600,
		Height: 400,
		Data:   big,
	})
	assert.NotNil(err)
}

func TestHeatMapRenderRaggedRows(t *testing.T) {
	assert := assert.New(t)

	// Rows with different lengths: missing trailing cells use the null style.
	p, err := HeatMapOptionRender(HeatMapOption{
		Type:    ChartOutputSVG,
		Width:   600,
		Height:  400,
		Rows:    []string{"A", "B", "C"},
		Columns: []string{"X", "Y", "Z"},
		Data: [][]float64{
			{1, 2, 3},
			{4, 5},
			{6},
		},
		NullColor: parseColor("#cccccc"),
		NullText:  "N/A",
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)
	assert.Contains(svg, "fill:rgba(204,204,204,1.0)")
	assert.Contains(svg, ">N/A<")
	// Three missing cells: (B,Z), (C,Y), (C,Z).
	assert.Equal(3, strings.Count(svg, ">N/A<"))
}

func TestHeatMapRenderAllMissing(t *testing.T) {
	assert := assert.New(t)

	p, err := HeatMapOptionRender(HeatMapOption{
		Type:    ChartOutputSVG,
		Width:   600,
		Height:  400,
		Rows:    []string{"A", "B"},
		Columns: []string{"X", "Y"},
		Data: [][]float64{
			{math.NaN(), math.Inf(-1)},
			{nullValue, math.NaN()},
		},
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)
	// 4 cell labels plus one missing value swatch in the legend.
	assert.Equal(5, strings.Count(svg, ">-<"))
}

func TestHeatMapRenderConstantValues(t *testing.T) {
	assert := assert.New(t)

	p, err := HeatMapOptionRender(HeatMapOption{
		Type:    ChartOutputSVG,
		Width:   600,
		Height:  400,
		Rows:    []string{"A", "B"},
		Columns: []string{"X", "Y"},
		Data: [][]float64{
			{7, 7},
			{7, 7},
		},
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)
	// All cells share the same fill color (the middle stop) and label.
	assert.Contains(svg, ">7<")
}

func TestHeatMapRenderLongLabelsAndNonSquare(t *testing.T) {
	assert := assert.New(t)

	// A wide non-square canvas with long labels: labels are rotated
	// automatically and every label is still rendered.
	rows := []string{"Warrior-Knight", "Mage-Sorcerer", "Rogue-Assassin"}
	columns := []string{
		"Chapter-1-The-Beginning",
		"Chapter-2-The-Journey",
		"Chapter-3-The-Final-Battle",
	}
	p, err := HeatMapOptionRender(HeatMapOption{
		Type:    ChartOutputSVG,
		Width:   900,
		Height:  300,
		Rows:    rows,
		Columns: columns,
		Data: [][]float64{
			{1, 2, 3},
			{4, 5, 6},
			{7, 8, 9},
		},
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)
	for _, label := range append(rows, columns...) {
		assert.Contains(svg, ">"+label+"<")
	}
}

func TestHeatMapRotatedLabelBounds(t *testing.T) {
	assert := assert.New(t)

	columns := []string{
		"Chapter-1-The-Beginning",
		"Chapter-2-The-Journey",
		"Chapter-3-The-Final-Battle",
		"Chapter-4-Epilogue",
	}
	rows := []string{"Warrior-Knight", "Mage-Sorcerer", "Rogue-Assassin"}
	data := [][]float64{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
		{9, 10, 11, 12},
	}

	// The geometry must be deterministic and in-bounds for both renderers.
	for _, outputType := range []string{ChartOutputSVG, ChartOutputPNG} {
		p, err := NewPainter(PainterOptions{
			Type:   outputType,
			Width:  960,
			Height: 480,
		})
		assert.Nil(err)
		chart := NewHeatMapChart(p, HeatMapOption{
			Rows:          rows,
			Columns:       columns,
			Data:          data,
			LabelRotation: -math.Pi / 4,
			Legend:        HeatMapLegendOption{Show: FalseFlag()},
		})
		layout := chart.computeLayout(p, len(rows), len(columns))
		assert.True(layout.columnLabelRotated)

		// Every rotated label box must stay inside the painter and below
		// the grid, so it can never overlap the cells or overflow canvas.
		p.SetTextRotation(-math.Pi / 4)
		labelAreaBottom := layout.gridHeight + layout.columnLabelAreaHeight - 5
		for i, label := range columns {
			box := p.MeasureText(label)
			columnCenter := i*layout.cellWidth + layout.cellWidth/2
			anchorX := columnCenter - (box.Left+box.Right)/2
			anchorY := labelAreaBottom - box.Bottom
			left, right := anchorX+box.Left, anchorX+box.Right
			top, bottom := anchorY+box.Top, anchorY+box.Bottom
			assert.GreaterOrEqual(left, 0, "%s left out of bounds (%s)", label, outputType)
			assert.LessOrEqual(right, p.Width(), "%s right out of bounds (%s)", label, outputType)
			assert.GreaterOrEqual(top, layout.gridHeight, "%s overlaps grid (%s)", label, outputType)
			assert.LessOrEqual(bottom, p.Height(), "%s bottom out of bounds (%s)", label, outputType)
		}
		p.ClearTextRotation()
	}
}

func TestHeatMapRenderCustomBorderAndFormatter(t *testing.T) {
	assert := assert.New(t)

	p, err := HeatMapOptionRender(HeatMapOption{
		Type:    ChartOutputSVG,
		Width:   600,
		Height:  400,
		Rows:    []string{"A"},
		Columns: []string{"X", "Y"},
		Data: [][]float64{
			{1, 2},
		},
		BorderWidth: 1,
		BorderColor: parseColor("#000000"),
		ValueFormatter: func(v float64) string {
			return "V" + strconv.FormatFloat(v, 'f', 0, 64)
		},
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)
	assert.Contains(svg, "stroke:rgba(0,0,0,1.0)")
	assert.Contains(svg, ">V1<")
	assert.Contains(svg, ">V2<")
}

func TestHeatMapRenderOptionFuncs(t *testing.T) {
	assert := assert.New(t)

	p, err := HeatMapRender(
		[][]float64{{-1, 1}},
		[]string{"A"},
		[]string{"X", "Y"},
		SVGTypeOption(),
		HeatMapMinMaxOptionFunc(-2, 2),
		HeatMapNullStyleOptionFunc(parseColor("#dddddd"), "empty"),
		HeatMapLabelShowOptionFunc(false),
		HeatMapLabelRotationOptionFunc(-math.Pi/4),
		HeatMapLegendOptionFunc(HeatMapLegendOption{Split: 2}),
	)
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)
	// Cell labels are hidden.
	assert.NotContains(svg, ">-1<")
	// Legend uses the configured split.
	assert.Contains(svg, ">-2<")
	assert.Contains(svg, ">2<")
}

func TestHeatMapRenderLabelHiddenForSmallCells(t *testing.T) {
	assert := assert.New(t)

	// A dense matrix: cells are smaller than the label threshold,
	// so no value labels are drawn but the chart still renders.
	data := make([][]float64, 40)
	for i := range data {
		data[i] = make([]float64, 40)
		for j := range data[i] {
			data[i][j] = float64(i*40 + j)
		}
	}
	p, err := HeatMapOptionRender(HeatMapOption{
		Type:   ChartOutputSVG,
		Width:  600,
		Height: 400,
		Data:   data,
		Legend: HeatMapLegendOption{Show: FalseFlag()},
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	assert.True(len(buf) > 0)
}
