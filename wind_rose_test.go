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

func windRoseDataList(data [][2]float64) SeriesList {
	return NewWindRoseSeriesList(data)
}

func TestNormalizeWindRoseAngle(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(0.0, normalizeWindRoseAngle(0))
	assert.Equal(1.0, normalizeWindRoseAngle(1))
	assert.Equal(359.0, normalizeWindRoseAngle(-1))
	assert.Equal(0.0, normalizeWindRoseAngle(360))
	assert.Equal(0.0, normalizeWindRoseAngle(720))
	assert.InDelta(0.5, normalizeWindRoseAngle(-359.5), 1e-9)
	assert.True(math.IsNaN(normalizeWindRoseAngle(math.NaN())))
	assert.True(math.IsNaN(normalizeWindRoseAngle(math.Inf(1))))
	assert.True(math.IsNaN(normalizeWindRoseAngle(math.Inf(-1))))
}

func TestAggregateWindRoseNorthCenter(t *testing.T) {
	assert := assert.New(t)
	// 359 and 1 should be aggregated into the north sector with a total of 2
	sectors, err := aggregateWindRose(
		windRoseDataList([][2]float64{{359, 1}, {1, 1}}),
		90,
	)
	assert.Nil(err)
	assert.Len(sectors, 4)
	// The north sector is the first one and spans [-45, 45), i.e.
	// [315, 360) union [0, 45)
	assert.Equal(2.0, sectors[0].values[1])
	assert.Equal(0.0, sectors[1].values[1])
	assert.Equal(0.0, sectors[2].values[1])
	assert.Equal(0.0, sectors[3].values[1])
	// The start angle of the north sector is -45 degrees in radians
	assert.InDelta(-math.Pi/4, sectors[0].start, 1e-9)
	assert.InDelta(math.Pi/2, sectors[0].delta, 1e-9)
}

func TestAggregateWindRoseSectorBoundary(t *testing.T) {
	assert := assert.New(t)
	// With a 30 degree width the sectors are centered on 0, 30, 60 ... so
	// their boundaries are at 15, 45, 75 ...
	// A value exactly on a boundary belongs to the sector starting at the
	// boundary (the lower angle sector)
	sectors, err := aggregateWindRose(
		windRoseDataList([][2]float64{{45, 1}, {0, 2}, {90, 3}, {345, 4}}),
		30,
	)
	assert.Nil(err)
	assert.Len(sectors, 12)
	// 45 is the start of the [45, 75) sector which is centered on 60,
	// its index is 2
	assert.Equal(1.0, sectors[2].values[1])
	// 90 is the start of the sector centered on 105, its index is 3
	assert.Equal(3.0, sectors[3].values[1])
	// Both 0 (weight 2) and 345 (weight 4) belong to the north sector
	// [345, 15) which crosses the 0 degree boundary
	assert.Equal(6.0, sectors[0].values[1])
}

func TestAggregateWindRoseAngle360(t *testing.T) {
	assert := assert.New(t)
	// 360 is normalized to 0 and belongs to the first sector
	sectors, err := aggregateWindRose(
		windRoseDataList([][2]float64{{360, 5}}),
		30,
	)
	assert.Nil(err)
	assert.Len(sectors, 12)
	assert.Equal(5.0, sectors[0].values[1])
}

func TestAggregateWindRoseZeroWeight(t *testing.T) {
	assert := assert.New(t)
	sectors, err := aggregateWindRose(
		windRoseDataList([][2]float64{{0, 0}, {180, 0}, {270, 0}}),
		90,
	)
	assert.Nil(err)
	for _, sector := range sectors {
		assert.Equal(0.0, sector.values[1])
	}
}

func TestAggregateWindRoseMultiSeries(t *testing.T) {
	assert := assert.New(t)
	seriesList := SeriesList{
		{
			Type: ChartTypeWindRose,
			Data: []SeriesData{
				NewWindRoseData(3, 10),
			},
		},
		{
			Type: ChartTypeWindRose,
			Data: []SeriesData{
				NewWindRoseData(5, 10),
			},
		},
	}
	sectors, err := aggregateWindRose(seriesList, 90)
	assert.Nil(err)
	// values[0] is zero, values[1] is the first series total and
	// values[2] is the stacked total of both series
	assert.Equal(0.0, sectors[0].values[0])
	assert.Equal(3.0, sectors[0].values[1])
	assert.Equal(8.0, sectors[0].values[2])
}

func TestAggregateWindRoseInvalidInput(t *testing.T) {
	assert := assert.New(t)
	// Negative weight
	_, err := aggregateWindRose(
		windRoseDataList([][2]float64{{0, -1}}),
		30,
	)
	assert.NotNil(err)
	// Illegal angle
	_, err = aggregateWindRose(
		windRoseDataList([][2]float64{{math.NaN(), 1}}),
		30,
	)
	assert.NotNil(err)
	_, err = aggregateWindRose(
		windRoseDataList([][2]float64{{math.Inf(1), 1}}),
		30,
	)
	assert.NotNil(err)
	// Width does not divide 360 evenly
	_, err = aggregateWindRose(
		windRoseDataList([][2]float64{{0, 1}}),
		37,
	)
	assert.NotNil(err)
	_, err = aggregateWindRose(
		windRoseDataList([][2]float64{{0, 1}}),
		100,
	)
	assert.NotNil(err)
	// Zero or negative width
	_, err = aggregateWindRose(
		windRoseDataList([][2]float64{{0, 1}}),
		0,
	)
	assert.NotNil(err)
	// Width greater than 360
	_, err = aggregateWindRose(
		windRoseDataList([][2]float64{{0, 1}}),
		400,
	)
	assert.NotNil(err)
}

func TestAggregateWindRoseFullCircleWidth(t *testing.T) {
	assert := assert.New(t)
	// A width of 360 produces a single sector that contains everything
	sectors, err := aggregateWindRose(
		windRoseDataList([][2]float64{{0, 1}, {179, 2}, {359, 3}}),
		360,
	)
	assert.Nil(err)
	assert.Len(sectors, 1)
	assert.Equal(6.0, sectors[0].values[1])
	assert.InDelta(0.0, sectors[0].start, 1e-9)
	assert.InDelta(2*math.Pi, sectors[0].delta, 1e-9)
}

func TestWindRoseRadialValues(t *testing.T) {
	assert := assert.New(t)
	// The max value snaps up to a multiple of divide count
	values := windRoseRadialValues(7, 4, nil)
	assert.Equal([]float64{0, 2, 4, 6, 8}, values)
	// Zero data still produces a usable unit scale
	values = windRoseRadialValues(0, 4, nil)
	assert.Equal([]float64{0, 1, 2, 3, 4}, values)
	// Explicit max is respected when it is not less than the data max
	max := 10.0
	values = windRoseRadialValues(7, 5, &max)
	assert.Equal([]float64{0, 2, 4, 6, 8, 10}, values)
	// An explicit max smaller than the data is ignored
	smallMax := 3.0
	values = windRoseRadialValues(7, 4, &smallMax)
	assert.Equal(8.0, values[4])
	// Invalid divide count falls back to the default value
	values = windRoseRadialValues(8, 0, nil)
	assert.Len(values, defaultWindRoseDivideCount+1)
}

func newWindRoseTestPainter(t *testing.T, typ string) *Painter {
	p, err := NewPainter(PainterOptions{
		Type:   typ,
		Width:  600,
		Height: 400,
	}, PainterThemeOption(defaultTheme))
	assert.Nil(t, err)
	return p.Child(PainterPaddingOption(Box{
		Left:   20,
		Top:    20,
		Right:  20,
		Bottom: 20,
	}))
}

// renderWindRose renders a chart and returns its bytes for both formats
func renderWindRose(t *testing.T, typ string, opt WindRoseChartOption) []byte {
	p := newWindRoseTestPainter(t, typ)
	opt.backgroundIsFilled = true
	_, err := NewWindRoseChart(p, opt).Render()
	assert.Nil(t, err)
	buf, err := p.Bytes()
	assert.Nil(t, err)
	assert.NotEmpty(t, buf)
	return buf
}

func TestWindRoseChartRenderSVG(t *testing.T) {
	assert := assert.New(t)
	buf := renderWindRose(t, ChartOutputSVG, WindRoseChartOption{
		SeriesList: windRoseDataList([][2]float64{
			{359, 1},
			{1, 1},
			{80, 3},
			{100, 2},
			{200, 4},
		}),
		SectorWidth: 90,
		Title: TitleOption{
			Text: "Wind Rose",
		},
		Legend: NewLegendOption([]string{
			"Wind Speed",
		}),
	})
	svg := string(buf)
	assert.True(strings.HasPrefix(svg, "<svg"))
	assert.Contains(svg, "Wind Rose")
	assert.Contains(svg, "Wind Speed")
	// The default compass labels should be rendered
	assert.Contains(svg, ">N<")
	assert.Contains(svg, ">E<")
	assert.Contains(svg, ">S<")
	assert.Contains(svg, ">W<")
	// Radial tick labels exist
	assert.Contains(svg, ">2<")
}

func TestWindRoseChartRenderPNG(t *testing.T) {
	assert := assert.New(t)
	buf := renderWindRose(t, ChartOutputPNG, WindRoseChartOption{
		SeriesList: windRoseDataList([][2]float64{
			{359, 1},
			{1, 1},
			{90, 2},
		}),
		SectorWidth: 90,
	})
	// The PNG header magic number
	assert.True(bytes.HasPrefix(buf, []byte{0x89, 0x50, 0x4e, 0x47}))
}

func TestWindRoseChartEmptyData(t *testing.T) {
	assert := assert.New(t)
	// Empty data is valid: only the grid is rendered
	buf := renderWindRose(t, ChartOutputSVG, WindRoseChartOption{
		SeriesList:  windRoseDataList(nil),
		SectorWidth: 30,
	})
	assert.NotEmpty(t, buf)
}

func TestWindRoseChartAllZeroWeights(t *testing.T) {
	assert := assert.New(t)
	buf := renderWindRose(t, ChartOutputSVG, WindRoseChartOption{
		SeriesList: windRoseDataList([][2]float64{
			{0, 0},
			{90, 0},
		}),
		SectorWidth: 30,
	})
	assert.NotEmpty(t, buf)
}

func TestWindRoseChartInvalidRender(t *testing.T) {
	assert := assert.New(t)
	p := newWindRoseTestPainter(t, ChartOutputSVG)
	_, err := NewWindRoseChart(p, WindRoseChartOption{
		SeriesList:  windRoseDataList([][2]float64{{0, -1}}),
		SectorWidth: 30,
	}).Render()
	assert.NotNil(err)

	p = newWindRoseTestPainter(t, ChartOutputSVG)
	_, err = NewWindRoseChart(p, WindRoseChartOption{
		SeriesList:  windRoseDataList([][2]float64{{0, 1}}),
		SectorWidth: 37,
	}).Render()
	assert.NotNil(err)
}

func TestWindRoseRenderDispatcher(t *testing.T) {
	assert := assert.New(t)
	// The top level Render function should support wind rose
	for _, typ := range []string{ChartOutputSVG, ChartOutputPNG} {
		p, err := WindRoseRender(
			[][2]float64{{359, 1}, {1, 1}, {90, 2}},
			TypeOptionFunc(typ),
			TitleTextOptionFunc("Wind Rose"),
			WindRoseSectorWidthOptionFunc(90),
			LegendLabelsOptionFunc([]string{"Speed"}),
		)
		assert.Nil(err)
		buf, err := p.Bytes()
		assert.Nil(err)
		assert.NotEmpty(t, buf)
	}
}

func TestWindRoseRenderMixError(t *testing.T) {
	assert := assert.New(t)
	_, err := Render(ChartOption{
		SeriesList: SeriesList{
			NewSeriesFromValues([]float64{1}, ChartTypeLine),
			NewSeriesFromValues([]float64{1}, ChartTypeWindRose),
		},
	})
	assert.NotNil(err)
}

// TestWindRoseGeometryPNG verifies the actual pixel geometry of the rendered
// PNG: north centered sectors and aligned center
func TestWindRoseGeometryPNG(t *testing.T) {
	assert := assert.New(t)
	p, err := NewPainter(PainterOptions{
		Type:   ChartOutputPNG,
		Width:  201,
		Height: 201,
	}, PainterThemeOption(defaultTheme))
	assert.Nil(err)
	_, err = NewWindRoseChart(p, WindRoseChartOption{
		SeriesList: windRoseDataList([][2]float64{
			{359, 1},
			{1, 1},
		}),
		SectorWidth:     90,
		AngularAxisShow: FalseFlag(),
		RadialAxis: WindRoseRadialAxisOption{
			Show: FalseFlag(),
		},
	}).Render()
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	img, err := png.Decode(bytes.NewReader(buf))
	assert.Nil(err)
	// The center should be inside the filled north wedge
	centerColor := img.At(100, 100)
	cr, _, _, _ := centerColor.RGBA()
	assert.NotEqual(255, cr>>8, "center should be filled by the north wedge")
	// A point straight north of center should be filled
	northColor := img.At(100, 40)
	nr, _, _, _ := northColor.RGBA()
	assert.NotEqual(255, nr>>8, "north point should be filled")
	// East and south should remain background (white)
	eastColor := img.At(160, 100)
	er, eg, eb, _ := eastColor.RGBA()
	assert.Equal(color.RGBA{R: 255, G: 255, B: 255, A: 255},
		color.RGBA{R: uint8(er >> 8), G: uint8(eg >> 8), B: uint8(eb >> 8), A: 255})
}
