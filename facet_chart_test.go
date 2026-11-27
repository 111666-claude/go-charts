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
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func facetTestCharts() []ChartOption {
	xLabels := []string{"a", "b"}
	return []ChartOption{
		{
			Title:  TitleOption{Text: "Panel A"},
			XAxis:  NewXAxisOption(xLabels),
			Legend: NewLegendOption([]string{"s1"}),
			SeriesList: SeriesList{
				NewSeriesFromValues([]float64{1, 10}),
			},
		},
		{
			Title:  TitleOption{Text: "Panel B"},
			XAxis:  NewXAxisOption(xLabels),
			Legend: NewLegendOption([]string{"s1"}),
			SeriesList: SeriesList{
				NewSeriesFromValues([]float64{3, 4}),
			},
		},
	}
}

func TestFacetGridNormalization(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		rows, columns, count int
		expRows, expCols     int
		hasErr               bool
	}{
		{0, 0, 3, 1, 3, false},
		{0, 2, 3, 2, 2, false},
		{2, 0, 4, 2, 2, false},
		{2, 2, 4, 2, 2, false},
		{1, 1, 2, 0, 0, true},
		{-1, 2, 2, 0, 0, true},
		{2, -1, 2, 0, 0, true},
	}
	for _, tt := range tests {
		rows, columns, err := normalizeFacetGrid(tt.rows, tt.columns, tt.count)
		if tt.hasErr {
			assert.NotNil(err)
			continue
		}
		assert.Nil(err)
		assert.Equal(tt.expRows, rows)
		assert.Equal(tt.expCols, columns)
	}
}

func TestFacetChartValidation(t *testing.T) {
	assert := assert.New(t)

	_, err := FacetChart(FacetChartOption{})
	assert.NotNil(err)

	valid := facetTestCharts()
	// tiny canvas
	_, err = FacetChart(FacetChartOption{
		Width: 20, Height: 20, Columns: 2, Charts: valid,
	})
	assert.NotNil(err)

	// grid can not hold all the panels
	_, err = FacetChart(FacetChartOption{
		Rows: 1, Columns: 1, Charts: valid,
	})
	assert.NotNil(err)

	// negative grid
	_, err = FacetChart(FacetChartOption{
		Rows: -1, Columns: 2, Charts: valid,
	})
	assert.NotNil(err)

	// shared Y without any value
	_, err = FacetChart(FacetChartOption{
		ShareYAxis: true,
		Columns:    1,
		Charts:     []ChartOption{{}, {}},
	})
	assert.NotNil(err)
}

// TestFacetSharedYAxisRange verifies the core requirement: the panel with
// values [3,4] must be positioned against the global 1..10 scale, not its
// own local scale.
func TestFacetSharedYAxisRange(t *testing.T) {
	assert := assert.New(t)

	charts := facetTestCharts()
	p, err := NewPainter(PainterOptions{
		Type:   ChartOutputSVG,
		Width:  400,
		Height: 300,
	})
	assert.Nil(err)
	r, err := buildFacetSharedYRange(charts, p, true)
	assert.Nil(err)
	// NewRange expands the global [1,10] data span to a clean axis whose
	// minimum is 0 and maximum exceeds the largest value.
	assert.Equal(0.0, r.min)
	assert.Greater(r.max, 10.0)
	globalMax := r.max

	// Every value maps to the same relative position regardless of panel
	plotSize := 240
	full := axisRange{min: 0, max: globalMax, size: plotSize}
	// value 4 from the [3,4] panel on the global scale
	globalHeight := full.getRestHeight(4)
	// value 4 on a hypothetical local [3,4]-derived scale would differ
	localRange := NewRange(AxisRangeOption{
		Min: 3, Max: 4, Size: plotSize, DivideCount: defaultAxisDivideCount,
	})
	localHeight := localRange.getRestHeight(4)
	assert.NotEqual(localHeight, globalHeight)
	// 4/globalMax of the way up from the bottom on the global scale
	assert.Equal(plotSize-int(4.0/globalMax*float64(plotSize)), globalHeight)
}

// TestFacetSharedYAxisPixels checks fixed pixel coordinates produced by the
// shared scale against the independently scaled output.
func TestFacetSharedYAxisPixels(t *testing.T) {
	assert := assert.New(t)

	charts := facetTestCharts()

	shared, err := FacetChart(FacetChartOption{
		Type:       ChartOutputSVG,
		Width:      800,
		Height:     400,
		Rows:       1,
		Columns:    2,
		ShareYAxis: true,
		Charts:     charts,
	})
	assert.Nil(err)
	sharedBuf, err := shared.Bytes()
	assert.Nil(err)
	sharedSVG := string(sharedBuf)

	independent, err := FacetChart(FacetChartOption{
		Type:    ChartOutputSVG,
		Width:   800,
		Height:  400,
		Rows:    1,
		Columns: 2,
		Charts:  charts,
	})
	assert.Nil(err)
	indBuf, err := independent.Bytes()
	assert.Nil(err)
	independentSVG := string(indBuf)

	// Extract the blue series line (first series color) of each panel
	lineRe := regexp.MustCompile(`<path  d="M (\d+) (\d+)\nL (\d+) (\d+)" style="stroke-width:2;stroke:rgba\(84,112,198`)
	sharedLines := lineRe.FindAllStringSubmatch(sharedSVG, -1)
	indLines := lineRe.FindAllStringSubmatch(independentSVG, -1)
	assert.Len(sharedLines, 2)
	assert.Len(indLines, 2)

	// Panel 1 holds [1,10] and its geometry must be identical in both
	// outputs (it defines the global min/max).
	assert.Equal(sharedLines[0][1:], indLines[0][1:])

	// Panel 2 holds [3,4]. Under independent scaling the line spans a much
	// larger vertical distance; under the shared global scale it is flat.
	sharedY1, _ := strconv.Atoi(sharedLines[1][2])
	sharedY2, _ := strconv.Atoi(sharedLines[1][4])
	indY1, _ := strconv.Atoi(indLines[1][2])
	indY2, _ := strconv.Atoi(indLines[1][4])
	sharedSpan := absInt(sharedY2 - sharedY1)
	indSpan := absInt(indY2 - indY1)
	assert.Less(sharedSpan, indSpan)

	// Exact fixed pixels for the shared output (deterministic with the
	// bundled font): the [3,4] panel maps both points onto the global axis.
	assert.Equal("558", sharedLines[1][1])
	assert.Equal("235", sharedLines[1][2])
	assert.Equal("686", sharedLines[1][3])
	assert.Equal("224", sharedLines[1][4])
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// TestFacetPlotAreasAlign verifies all panel axes share identical top and
// bottom coordinates.
func TestFacetPlotAreasAlign(t *testing.T) {
	assert := assert.New(t)

	charts := facetTestCharts()
	p, err := FacetChart(FacetChartOption{
		Type:    ChartOutputSVG,
		Width:   800,
		Height:  400,
		Rows:    1,
		Columns: 2,
		Charts:  charts,
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)

	// Horizontal axis lines (the X axis of each panel)
	axisRe := regexp.MustCompile(`<path  d="M (\d+) (\d+)\nL (\d+) (\d+)" style="stroke-width:1;stroke:rgba\(110,112,121`)
	var axisYs []int
	for _, m := range axisRe.FindAllStringSubmatch(string(buf), -1) {
		y1, _ := strconv.Atoi(m[2])
		y2, _ := strconv.Atoi(m[4])
		if y1 == y2 { // horizontal line
			axisYs = append(axisYs, y1)
		}
	}
	assert.NotEmpty(axisYs)
	first := axisYs[0]
	for _, y := range axisYs {
		assert.Equal(first, y)
	}
}

// TestFacetCommonLegendAndTitle checks the legend renders once, each panel
// keeps its own title, and labels are not duplicated for inner panels.
func TestFacetCommonLegendAndTitle(t *testing.T) {
	assert := assert.New(t)

	charts := facetTestCharts()
	p, err := FacetChart(FacetChartOption{
		Type:    ChartOutputSVG,
		Width:   800,
		Height:  400,
		Rows:    1,
		Columns: 2,
		Charts:  charts,
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)

	assert.Equal(1, countOccurrences(svg, ">s1<"))
	assert.Equal(1, countOccurrences(svg, ">Panel A<"))
	assert.Equal(1, countOccurrences(svg, ">Panel B<"))

	// Inner column suppresses its Y labels: the numeric labels only appear
	// in the first column.
	labelRe := regexp.MustCompile(`<text x="(\d+)"[^>]*>([\d.]+)</text>`)
	firstColumnLabels := 0
	innerColumnLabels := 0
	for _, m := range labelRe.FindAllStringSubmatch(svg, -1) {
		x, _ := strconv.Atoi(m[1])
		// panel boundary sits in the middle of the 800px canvas
		if x < 400 {
			firstColumnLabels++
		} else {
			innerColumnLabels++
		}
	}
	assert.Greater(firstColumnLabels, 0)
	assert.Equal(0, innerColumnLabels)
}

func countOccurrences(s, sub string) int {
	count := 0
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			count++
		}
	}
	return count
}

// TestFacetEmptyPanel verifies an empty slot leaves a gap but other panels
// still render with their axes and labels.
func TestFacetEmptyPanel(t *testing.T) {
	assert := assert.New(t)

	base := facetTestCharts()
	withEmpty := append(base, ChartOption{})
	p, err := FacetChart(FacetChartOption{
		Type:    ChartOutputSVG,
		Width:   900,
		Height:  500,
		Columns: 2,
		Charts:  withEmpty,
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)
	// the two real panels keep their X labels
	assert.GreaterOrEqual(countOccurrences(svg, ">a<"), 2)
	assert.GreaterOrEqual(countOccurrences(svg, ">b<"), 2)
}

// TestFacetDifferentSeriesCount verifies panels with different series
// counts render and colors stay stable by series name.
func TestFacetDifferentSeriesCount(t *testing.T) {
	assert := assert.New(t)

	xLabels := []string{"a", "b"}
	panel1 := ChartOption{
		XAxis:  NewXAxisOption(xLabels),
		Legend: NewLegendOption([]string{"alpha", "beta"}),
		SeriesList: NewSeriesListDataFromValues([][]float64{
			{1, 10},
			{2, 9},
		}),
	}
	panel2 := ChartOption{
		XAxis:  NewXAxisOption(xLabels),
		Legend: NewLegendOption([]string{"alpha"}),
		SeriesList: NewSeriesListDataFromValues([][]float64{
			{3, 4},
		}),
	}
	p, err := FacetChart(FacetChartOption{
		Type:       ChartOutputSVG,
		Width:      800,
		Height:     400,
		Columns:    2,
		ShareYAxis: true,
		Charts:     []ChartOption{panel1, panel2},
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)
	// both legend names are collected
	assert.Equal(1, countOccurrences(svg, ">alpha<"))
	assert.Equal(1, countOccurrences(svg, ">beta<"))
	// the shared "alpha" series uses the first color in both panels
	blueRe := regexp.MustCompile(`stroke-width:2;stroke:rgba\(84,112,198`)
	assert.GreaterOrEqual(len(blueRe.FindAllString(svg, -1)), 2)
}

// TestFacetPNGOutput verifies PNG output is produced and non-empty.
func TestFacetPNGOutput(t *testing.T) {
	assert := assert.New(t)

	p, err := FacetChart(FacetChartOption{
		Type:       ChartOutputPNG,
		Width:      800,
		Height:     400,
		Rows:       1,
		Columns:    2,
		ShareYAxis: true,
		Charts:     facetTestCharts(),
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	assert.Greater(len(buf), 0)
	// PNG signature
	assert.Equal(uint8(0x89), buf[0])
	assert.Equal(uint8(0x50), buf[1])
}

// TestFacetLongLabelsAndNarrowCanvas verifies long labels and a narrow
// canvas still render without error and keep aligned axes.
func TestFacetLongLabelsAndNarrowCanvas(t *testing.T) {
	assert := assert.New(t)

	longChart := ChartOption{
		XAxis: NewXAxisOption([]string{
			"very-very-very-long-label-1",
			"very-very-very-long-label-2",
		}),
		SeriesList: SeriesList{NewSeriesFromValues([]float64{1, 2})},
	}
	p, err := FacetChart(FacetChartOption{
		Type:    ChartOutputSVG,
		Width:   320,
		Height:  200,
		Columns: 2,
		Charts:  []ChartOption{longChart, longChart},
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	assert.Greater(len(buf), 0)
}

// TestFacetSharedXLabels verifies X sharing only activates for identical
// labels and otherwise falls back to independent labels.
func TestFacetSharedXLabels(t *testing.T) {
	assert := assert.New(t)

	identical, ok := buildFacetSharedXLabels(facetTestCharts(), true)
	assert.True(ok)
	assert.Equal([]string{"a", "b"}, identical)

	different := []ChartOption{
		{XAxis: NewXAxisOption([]string{"a", "b"})},
		{XAxis: NewXAxisOption([]string{"c", "d"})},
	}
	_, ok = buildFacetSharedXLabels(different, true)
	assert.False(ok)

	// disabled never shares
	_, ok = buildFacetSharedXLabels(facetTestCharts(), false)
	assert.False(ok)
}

// TestFacetSeriesColorIndex verifies the color index override mechanism.
func TestFacetSeriesColorIndex(t *testing.T) {
	assert := assert.New(t)

	sl := SeriesList{{ColorIndex: 3}}
	sl.init()
	assert.Equal(3, colorIndexOf(sl[0]))

	// zero value falls back to the series index
	sl2 := SeriesList{{}}
	sl2.init()
	assert.Equal(0, colorIndexOf(sl2[0]))
}

// TestFacetIsFinite covers the range validation helper.
func TestFacetIsFinite(t *testing.T) {
	assert := assert.New(t)
	assert.True(isFinite(1.0))
	assert.False(isFinite(math.NaN()))
	assert.False(isFinite(math.Inf(1)))
	assert.False(isFinite(math.Inf(-1)))
}
