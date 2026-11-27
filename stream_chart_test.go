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
	"image/color"
	"image/png"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamLayoutCenterBaseline(t *testing.T) {
	assert := assert.New(t)

	// two constant series: 1 and 2
	seriesList := NewSeriesListDataFromValues([][]float64{
		{1, 1, 1},
		{2, 2, 2},
	}, ChartTypeStream)
	layout, err := newStreamLayout(seriesList, StreamBaselineCenter, nil)
	assert.Nil(err)

	// baseline is -total/2 = -1.5
	for _, v := range layout.baseline {
		assert.Equal(-1.5, v)
	}
	// three boundaries: -1.5, -0.5 and 1.5
	assert.Equal(3, len(layout.boundaries))
	for j := 0; j < 3; j++ {
		assert.Equal(-1.5, layout.boundaries[0][j])
		assert.Equal(-0.5, layout.boundaries[1][j])
		assert.Equal(1.5, layout.boundaries[2][j])
		// the thickness of every band equals its series value
		assert.Equal(1.0, layout.boundaries[1][j]-layout.boundaries[0][j])
		assert.Equal(2.0, layout.boundaries[2][j]-layout.boundaries[1][j])
		// the sum of band thickness equals the total value
		assert.Equal(3.0, layout.boundaries[2][j]-layout.boundaries[0][j])
	}
	assert.Equal(-1.5, layout.minValue)
	assert.Equal(1.5, layout.maxValue)
}

func TestStreamLayoutWiggleBaseline(t *testing.T) {
	assert := assert.New(t)

	// d3-shape stackOffsetWiggle result for two varying series
	// {1,2,3} and {3,2,1}:
	// g0[0] = 0, g0[1] = -0.5, g0[2] = -1
	seriesList := NewSeriesListDataFromValues([][]float64{
		{1, 2, 3},
		{3, 2, 1},
	}, ChartTypeStream)
	layout, err := newStreamLayout(seriesList, StreamBaselineWiggle, nil)
	assert.Nil(err)
	assert.InDelta(0, layout.baseline[0], 1e-9)
	assert.InDelta(-0.5, layout.baseline[1], 1e-9)
	assert.InDelta(-1, layout.baseline[2], 1e-9)
	// the band thickness still equals the series value at every point
	for j := 0; j < 3; j++ {
		assert.InDelta(seriesList[0].Data[j].Value,
			layout.boundaries[1][j]-layout.boundaries[0][j], 1e-9)
		assert.InDelta(seriesList[1].Data[j].Value,
			layout.boundaries[2][j]-layout.boundaries[1][j], 1e-9)
	}
	// the thickness sum equals the total value at every point
	for j := 0; j < 3; j++ {
		total := 0.0
		for i := range layout.values {
			total += layout.values[i][j]
		}
		assert.InDelta(total,
			layout.boundaries[len(layout.boundaries)-1][j]-layout.boundaries[0][j], 1e-9)
	}

	// constant series do not wiggle: the baseline stays flat at zero
	constant := NewSeriesListDataFromValues([][]float64{
		{1, 1, 1},
		{2, 2, 2},
	}, ChartTypeStream)
	constantLayout, err := newStreamLayout(constant, StreamBaselineWiggle, nil)
	assert.Nil(err)
	for _, v := range constantLayout.baseline {
		assert.InDelta(0, v, 1e-9)
	}
}

func TestStreamLayoutMissingValue(t *testing.T) {
	assert := assert.New(t)

	// null value means missing data and contributes no thickness
	seriesList := NewSeriesListDataFromValues([][]float64{
		{1, nullValue, 1},
		{2, 2, 2},
	}, ChartTypeStream)
	layout, err := newStreamLayout(seriesList, StreamBaselineCenter, nil)
	assert.Nil(err)
	// total of the middle point is 2 only, baseline = -1
	assert.Equal(-1.5, layout.baseline[0])
	assert.Equal(-1.0, layout.baseline[1])
	assert.Equal(-1.5, layout.baseline[2])
	// first series collapses at the missing point
	assert.Equal(0.0, layout.boundaries[1][1]-layout.boundaries[0][1])
	assert.Equal(2.0, layout.boundaries[2][1]-layout.boundaries[1][1])
	// thickness sum equals the total at every point
	for j := 0; j < 3; j++ {
		total := 0.0
		for i := range layout.values {
			total += layout.values[i][j]
		}
		assert.Equal(total, layout.boundaries[len(layout.boundaries)-1][j]-layout.boundaries[0][j])
	}
}

func TestStreamLayoutNonEquidistantXValues(t *testing.T) {
	assert := assert.New(t)

	seriesList := NewSeriesListDataFromValues([][]float64{
		{1, 2, 3},
		{3, 2, 1},
	}, ChartTypeStream)
	// non-equidistant timestamps
	xValues := []float64{0, 1, 3}
	layout, err := newStreamLayout(seriesList, StreamBaselineCenter, xValues)
	assert.Nil(err)

	// pixel positions follow the numeric x scale: width 300,
	// point 1 at 1/3 of the span instead of the evenly spaced 150
	assert.Equal(0, layout.xPixel(300, 3, 0))
	assert.Equal(100, layout.xPixel(300, 3, 1))
	assert.Equal(300, layout.xPixel(300, 3, 2))
	// evenly spaced samples still get even pixel positions
	evenLayout, err := newStreamLayout(seriesList, StreamBaselineCenter, nil)
	assert.Nil(err)
	assert.Equal(0, evenLayout.xPixel(300, 3, 0))
	assert.Equal(150, evenLayout.xPixel(300, 3, 1))
	assert.Equal(300, evenLayout.xPixel(300, 3, 2))

	// invalid x values
	_, err = newStreamLayout(seriesList, StreamBaselineWiggle, []float64{0, 1, 1})
	assert.Equal("x values of stream chart should be strictly increasing", err.Error())
	_, err = newStreamLayout(seriesList, StreamBaselineWiggle, []float64{0, 1})
	assert.Equal("the length of x values should equal the data length of series", err.Error())
	_, err = newStreamLayout(seriesList, StreamBaselineWiggle, []float64{0, math.NaN(), 1})
	assert.NotNil(err)
}

func TestStreamLayoutInvalidData(t *testing.T) {
	assert := assert.New(t)

	// negative value
	negative := NewSeriesListDataFromValues([][]float64{
		{1, -1},
		{2, 2},
	}, ChartTypeStream)
	_, err := newStreamLayout(negative, StreamBaselineCenter, nil)
	assert.Equal("stream chart only supports non-negative value", err.Error())

	// non-finite value: infinity
	infSeries := NewSeriesListDataFromValues([][]float64{
		{1, math.Inf(1)},
		{2, 2},
	}, ChartTypeStream)
	_, err = newStreamLayout(infSeries, StreamBaselineCenter, nil)
	assert.Equal("stream chart does not support non-finite value", err.Error())

	// non-finite value: NaN
	nanSeries := NewSeriesListDataFromValues([][]float64{
		{1, math.NaN()},
		{2, 2},
	}, ChartTypeStream)
	_, err = newStreamLayout(nanSeries, StreamBaselineCenter, nil)
	assert.Equal("stream chart does not support non-finite value", err.Error())

	// zero total everywhere
	allNull := NewSeriesListDataFromValues([][]float64{
		{nullValue, nullValue},
		{0, 0},
	}, ChartTypeStream)
	_, err = newStreamLayout(allNull, StreamBaselineCenter, nil)
	assert.Equal("the total value of stream chart is always zero", err.Error())

	// a single zero point is allowed, only all-zero total is rejected
	someZero := NewSeriesListDataFromValues([][]float64{
		{0, 1},
		{0, 2},
	}, ChartTypeStream)
	_, err = newStreamLayout(someZero, StreamBaselineCenter, nil)
	assert.Nil(err)

	// mismatched series length
	mismatched := NewSeriesListDataFromValues([][]float64{
		{1, 1, 1},
		{2, 2},
	}, ChartTypeStream)
	_, err = newStreamLayout(mismatched, StreamBaselineCenter, nil)
	assert.Equal("all series of stream chart should have the same data length", err.Error())

	// empty series
	_, err = newStreamLayout(SeriesList{}, StreamBaselineCenter, nil)
	assert.Equal("series list of stream chart can not be empty", err.Error())

	// single point
	single := NewSeriesListDataFromValues([][]float64{
		{1},
	}, ChartTypeStream)
	_, err = newStreamLayout(single, StreamBaselineCenter, nil)
	assert.Equal("stream chart requires at least two data points", err.Error())

	// invalid baseline
	valid := NewSeriesListDataFromValues([][]float64{
		{1, 1},
		{2, 2},
	}, ChartTypeStream)
	_, err = newStreamLayout(valid, "unknown", nil)
	assert.Equal("baseline of stream chart should be center or wiggle", err.Error())
}

func TestStreamLayoutStableOrder(t *testing.T) {
	assert := assert.New(t)

	values := [][]float64{
		{1, 2, 3},
		{3, 2, 1},
		{2, 2, 2},
	}
	seriesList := NewSeriesListDataFromValues(values, ChartTypeStream)
	layout, err := newStreamLayout(seriesList, StreamBaselineCenter, nil)
	assert.Nil(err)

	// series order is stable: series i is always bounded by
	// boundaries[i] and boundaries[i+1] with its own thickness
	for i, row := range values {
		for j, v := range row {
			assert.Equal(v, layout.boundaries[i+1][j]-layout.boundaries[i][j])
		}
	}
}

// streamChartTestOption builds the deterministic option used by the
// fixed coordinate acceptance test: the y axis is pinned to the exact
// stacked boundary extent so the pixel coordinates are stable.
func streamChartTestOption(outputType string) (ChartOption, error) {
	values := [][]float64{
		{1, 1, 1},
		{2, 2, 2},
	}
	opt := ChartOption{
		Type:   outputType,
		Width:  400,
		Height: 300,
		Padding: Box{
			Top:    20,
			Right:  20,
			Bottom: 20,
			Left:   20,
		},
		XAxis: NewXAxisOption([]string{"a", "b", "c"}, FalseFlag()),
		Legend: NewLegendOption([]string{
			"s1",
			"s2",
		}),
		YAxisOptions: []YAxisOption{
			{
				Min: NewFloatPoint(-1.5),
				Max: NewFloatPoint(1.5),
			},
		},
		SeriesList:     NewSeriesListDataFromValues(values, ChartTypeStream),
		StreamBaseline: StreamBaselineCenter,
	}
	return opt, nil
}

func TestStreamChartRenderFixedCoordinatesSVG(t *testing.T) {
	assert := assert.New(t)

	opt, _ := streamChartTestOption(ChartOutputSVG)
	p, err := Render(opt)
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	svg := string(buf)
	assert.True(strings.HasPrefix(svg, "<svg"))
	// path data contains newline separators, normalize them for matching
	normalized := strings.ReplaceAll(svg, "\n", " ")

	// the two bands with fixed coordinates and theme colors.
	// the shared boundary (173) is reused by both polygons so the
	// scaled bands never leave a seam between them.
	assert.Contains(normalized,
		"M 57 173 L 219 173 L 380 173 L 380 250 L 219 250 L 57 250 Z")
	assert.Contains(normalized, "fill:rgba(84,112,198,1.0)")
	assert.Contains(normalized,
		"M 57 20 L 219 20 L 380 20 L 380 173 L 219 173 L 57 173 Z")
	assert.Contains(normalized, "fill:rgba(145,204,117,1.0)")
	// legend keeps the same series order and colors
	assert.Contains(svg, ">s1<")
	assert.Contains(svg, ">s2<")
	// y axis spans the exact stacked extent, no padding
	assert.Contains(svg, ">1.5<")
	assert.Contains(svg, ">-1.5<")
}

func TestStreamChartRenderFixedCoordinatesPNG(t *testing.T) {
	assert := assert.New(t)

	opt, _ := streamChartTestOption(ChartOutputPNG)
	p, err := Render(opt)
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	// valid png signature
	assert.Equal([]byte{0x89, 0x50, 0x4e, 0x47}, buf[:4])

	img, err := png.Decode(bytes.NewReader(buf))
	assert.Nil(err)
	blue := color.RGBA{R: 84, G: 112, B: 198, A: 255}
	green := color.RGBA{R: 145, G: 204, B: 117, A: 255}
	// columns away from the x tick columns (57/219/380):
	// green upper band rows 20..172, blue lower band rows 173..249,
	// every row between the outer boundaries is painted and the two
	// bands are contiguous without any white seam.
	for _, x := range []int{120, 260, 340} {
		for y := 20; y <= 172; y++ {
			assert.Equal(green, img.At(x, y), "green band at (%d,%d)", x, y)
		}
		for y := 173; y <= 249; y++ {
			assert.Equal(blue, img.At(x, y), "blue band at (%d,%d)", x, y)
		}
		// immediately outside the bands is background, confirming the
		// outer boundaries map exactly to the stacked values
		assert.NotEqual(green, img.At(x, 19))
		assert.NotEqual(blue, img.At(x, 250))
	}
}

func TestStreamChartRenderViaTopLevel(t *testing.T) {
	assert := assert.New(t)

	values := [][]float64{
		{1, 2, 3, 4},
		{4, 3, 2, 1},
	}
	// center baseline through the top level StreamRender entry
	p, err := StreamRender(values,
		WidthOptionFunc(500),
		HeightOptionFunc(300),
		LegendLabelsOptionFunc([]string{"a", "b"}),
		XAxisDataOptionFunc([]string{"q1", "q2", "q3", "q4"}, FalseFlag()),
		SVGTypeOption(),
	)
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	assert.True(strings.HasPrefix(string(buf), "<svg"))

	// wiggle baseline and non-equidistant x values, png output
	p, err = StreamRender(values,
		WidthOptionFunc(500),
		HeightOptionFunc(300),
		StreamBaselineOptionFunc(StreamBaselineWiggle),
		StreamXValuesOptionFunc([]float64{0, 2, 5, 9}),
		PNGTypeOption(),
	)
	assert.Nil(err)
	buf, err = p.Bytes()
	assert.Nil(err)
	assert.Equal([]byte{0x89, 0x50, 0x4e, 0x47}, buf[:4])
}

func TestStreamChartCannotMixOtherCharts(t *testing.T) {
	assert := assert.New(t)

	_, err := Render(ChartOption{
		Type: ChartOutputSVG,
		SeriesList: SeriesList{
			NewSeriesFromValues([]float64{1, 2}, ChartTypeStream),
			NewSeriesFromValues([]float64{2, 1}, ChartTypeLine),
		},
	})
	assert.Equal("Stream can not mix other charts", err.Error())
}

func TestStreamChartInvalidDataReturnsError(t *testing.T) {
	assert := assert.New(t)

	_, err := StreamRender([][]float64{
		{1, -1},
		{2, 2},
	}, SVGTypeOption())
	assert.Equal("stream chart only supports non-negative value", err.Error())

	_, err = StreamRender([][]float64{
		{0, 0},
		{0, 0},
	}, SVGTypeOption())
	assert.Equal("the total value of stream chart is always zero", err.Error())
}

func TestStreamChartDefaultBaseline(t *testing.T) {
	assert := assert.New(t)

	// an empty baseline defaults to the centered baseline instead of
	// returning an error, which also covers the ECharts JSON entry
	p, err := Render(ChartOption{
		Type:   ChartOutputSVG,
		Width:  400,
		Height: 300,
		XAxis:  NewXAxisOption([]string{"a", "b"}, FalseFlag()),
		Legend: NewLegendOption([]string{"s1", "s2"}),
		SeriesList: NewSeriesListDataFromValues([][]float64{
			{1, 1},
			{2, 2},
		}, ChartTypeStream),
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	assert.True(strings.HasPrefix(string(buf), "<svg"))
}

func TestStreamChartEChartsEntry(t *testing.T) {
	assert := assert.New(t)

	option := `{
		"type":"svg","width":400,"height":300,
		"title":{"text":"Stream"},
		"legend":{"data":["a","b"]},
		"xAxis":[{"data":["q1","q2","q3","q4"],"boundaryGap":false}],
		"yAxis":[{}],
		"series":[
			{"name":"a","type":"stream","data":[1,2,3,4]},
			{"name":"b","type":"stream","data":[4,3,2,1]}
		]
	}`
	buf, err := RenderEChartsToSVG(option)
	assert.Nil(err)
	assert.True(strings.HasPrefix(string(buf), "<svg"))
	// both series colors are present
	assert.Contains(string(buf), "fill:rgba(84,112,198,1.0)")
	assert.Contains(string(buf), "fill:rgba(145,204,117,1.0)")
}
