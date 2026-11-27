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

	"github.com/golang/freetype/truetype"
)

const (
	// StreamBaselineCenter uses a symmetric baseline, the stacked
	// bands are centered around zero.
	StreamBaselineCenter = "center"
	// StreamBaselineWiggle uses the wiggle baseline which minimizes
	// the weighted wiggle of the layers (the classic streamgraph look).
	StreamBaselineWiggle = "wiggle"
)

// streamChart renders a streamgraph: a stacked area chart with a
// free (center or wiggle) baseline.
type streamChart struct {
	p   *Painter
	opt *StreamChartOption
}

// StreamChartOption is the option of stream chart
type StreamChartOption struct {
	// The theme
	Theme ColorPalette
	// The font
	Font *truetype.Font
	// The data series list, every series should have the same length
	// and only non-negative finite values are allowed.
	// A null value (see GetNullValue) is treated as missing data.
	SeriesList SeriesList
	// The x axis option
	XAxis XAxisOption
	// The padding of stream chart
	Padding Box
	// The y axis option
	YAxisOptions []YAxisOption
	// The option of title
	Title TitleOption
	// The legend option
	Legend LegendOption
	// The baseline type, it can be "center" or "wiggle".
	// Default value is "center"
	Baseline string
	// XValues are the numeric positions of the samples on the x axis,
	// e.g. unix timestamps. When nil the samples are spaced evenly,
	// which makes the chart support non-equidistant time series.
	// When set, its length must equal the data length of every series
	// and the values must be strictly increasing finite numbers.
	XValues []float64
	// background is filled
	backgroundIsFilled bool
}

// NewStreamChart returns a stream chart renderer
func NewStreamChart(p *Painter, opt StreamChartOption) *streamChart {
	if opt.Theme == nil {
		opt.Theme = defaultTheme
	}
	if opt.Baseline == "" {
		opt.Baseline = StreamBaselineCenter
	}
	return &streamChart{
		p:   p,
		opt: &opt,
	}
}

// streamLayout is the value-space layout result of a streamgraph.
type streamLayout struct {
	// values is the sanitized data, values[seriesIndex][pointIndex]
	values [][]float64
	// xValues is the numeric x position of every point
	xValues []float64
	// baseline is the y value of the bottom boundary of the stack
	baseline []float64
	// boundaries[k] is the upper boundary of series k-1,
	// boundaries[0] equals baseline and the last one is the top boundary
	boundaries [][]float64
	// minValue and maxValue span every boundary
	minValue float64
	maxValue float64
}

// streamSanitize converts the series list into a dense, non-negative
// finite matrix. Null values (missing data) become zero.
func streamSanitize(seriesList SeriesList, pointCount int) ([][]float64, error) {
	values := make([][]float64, len(seriesList))
	for i, series := range seriesList {
		if len(series.Data) != pointCount {
			return nil, errors.New("all series of stream chart should have the same data length")
		}
		row := make([]float64, pointCount)
		for j, item := range series.Data {
			v := item.Value
			// null value means missing data, it contributes no thickness
			if v == nullValue {
				row[j] = 0
				continue
			}
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, errors.New("stream chart does not support non-finite value")
			}
			if v < 0 {
				return nil, errors.New("stream chart only supports non-negative value")
			}
			row[j] = v
		}
		values[i] = row
	}
	return values, nil
}

// streamXValues builds the numeric x position of every point.
func streamXValues(custom []float64, pointCount int) ([]float64, error) {
	if custom == nil {
		xValues := make([]float64, pointCount)
		for i := 0; i < pointCount; i++ {
			xValues[i] = float64(i)
		}
		return xValues, nil
	}
	if len(custom) != pointCount {
		return nil, errors.New("the length of x values should equal the data length of series")
	}
	xValues := make([]float64, pointCount)
	prev := math.Inf(-1)
	for i, v := range custom {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, errors.New("x values of stream chart should be finite")
		}
		if v <= prev {
			return nil, errors.New("x values of stream chart should be strictly increasing")
		}
		xValues[i] = v
		prev = v
	}
	return xValues, nil
}

// newStreamLayout computes the baseline and every stacked boundary.
func newStreamLayout(seriesList SeriesList, baseline string, customXValues []float64) (*streamLayout, error) {
	if len(seriesList) == 0 {
		return nil, errors.New("series list of stream chart can not be empty")
	}
	pointCount := len(seriesList[0].Data)
	if pointCount < 2 {
		return nil, errors.New("stream chart requires at least two data points")
	}
	values, err := streamSanitize(seriesList, pointCount)
	if err != nil {
		return nil, err
	}
	xValues, err := streamXValues(customXValues, pointCount)
	if err != nil {
		return nil, err
	}
	seriesCount := len(values)

	// total value of every point
	total := make([]float64, pointCount)
	hasData := false
	for j := 0; j < pointCount; j++ {
		sum := 0.0
		for i := 0; i < seriesCount; i++ {
			sum += values[i][j]
		}
		total[j] = sum
		if sum > 0 {
			hasData = true
		}
	}
	if !hasData {
		return nil, errors.New("the total value of stream chart is always zero")
	}
	if baseline != StreamBaselineCenter && baseline != StreamBaselineWiggle {
		return nil, errors.New("baseline of stream chart should be center or wiggle")
	}

	base := make([]float64, pointCount)
	if baseline == StreamBaselineCenter {
		// symmetric baseline: -total/2
		for j := 0; j < pointCount; j++ {
			base[j] = -total[j] / 2
		}
	} else {
		// wiggle baseline, the exact d3-shape stackOffsetWiggle
		// algorithm (Byron & Wattenberg):
		// g0[0] = 0 and for every following point j the baseline moves
		// by the value weighted average of the per-series offsets
		//   s3_i = (f_i[j]-f_i[j-1])/2 + sum_{k<i}(f_k[j]-f_k[j-1])
		//   g0[j] = g0[j-1] - sum_i(s3_i*f_i[j]) / sum_i f_i[j]
		for j := 1; j < pointCount; j++ {
			s1 := 0.0
			s2 := 0.0
			for i := 0; i < seriesCount; i++ {
				sij0 := values[i][j]
				sij1 := values[i][j-1]
				s3 := (sij0 - sij1) / 2
				for k := 0; k < i; k++ {
					s3 += values[k][j] - values[k][j-1]
				}
				s1 += sij0
				s2 += s3 * sij0
			}
			if s1 != 0 {
				base[j] = base[j-1] - s2/s1
			} else {
				// zero total point: keep the previous baseline
				base[j] = base[j-1]
			}
		}
	}

	// boundaries: n+1 rows, the first one is the baseline
	boundaries := make([][]float64, seriesCount+1)
	boundaries[0] = base
	minValue := math.MaxFloat64
	maxValue := -math.MaxFloat64
	updateExtent := func(v float64) {
		if v < minValue {
			minValue = v
		}
		if v > maxValue {
			maxValue = v
		}
	}
	for j := 0; j < pointCount; j++ {
		updateExtent(base[j])
	}
	cursor := make([]float64, pointCount)
	copy(cursor, base)
	for i := 0; i < seriesCount; i++ {
		upper := make([]float64, pointCount)
		for j := 0; j < pointCount; j++ {
			cursor[j] += values[i][j]
			upper[j] = cursor[j]
			updateExtent(upper[j])
		}
		boundaries[i+1] = upper
	}

	return &streamLayout{
		values:     values,
		xValues:    xValues,
		baseline:   base,
		boundaries: boundaries,
		minValue:   minValue,
		maxValue:   maxValue,
	}, nil
}

// xPixel returns the pixel x position of point index i.
func (l *streamLayout) xPixel(width, pointCount int, i int) int {
	if pointCount == 1 {
		return 0
	}
	min := l.xValues[0]
	max := l.xValues[pointCount-1]
	span := max - min
	if span == 0 {
		return width * i / (pointCount - 1)
	}
	// boundaryGap is false: the first and last points touch the edges
	return int(math.Round((l.xValues[i] - min) / span * float64(width)))
}

// yPixel converts a value-space y to a pixel row inside a painter
// of the given height. Adjacent bands share the exact same boundary
// rows, so the scaled bands never leave a seam between them.
func (l *streamLayout) yPixel(height int, min, max, value float64) int {
	if max <= min {
		return height
	}
	v := (value - min) / (max - min)
	return height - int(math.Round(v*float64(height)))
}

// bandPoints builds the closed polygon of one series: the upper
// boundary from left to right and the lower boundary back.
func (l *streamLayout) bandPoints(seriesIndex, width, height int, min, max float64) []Point {
	pointCount := len(l.xValues)
	points := make([]Point, 0, pointCount*2+1)
	// upper boundary, left to right
	for j := 0; j < pointCount; j++ {
		points = append(points, Point{
			X: l.xPixel(width, pointCount, j),
			Y: l.yPixel(height, min, max, l.boundaries[seriesIndex+1][j]),
		})
	}
	// lower boundary, right to left
	for j := pointCount - 1; j >= 0; j-- {
		points = append(points, Point{
			X: l.xPixel(width, pointCount, j),
			Y: l.yPixel(height, min, max, l.boundaries[seriesIndex][j]),
		})
	}
	return points
}

func (s *streamChart) render(result *defaultRenderResult, seriesList SeriesList) (Box, error) {
	p := s.p
	opt := s.opt
	layout, err := newStreamLayout(seriesList, opt.Baseline, opt.XValues)
	if err != nil {
		return BoxZero, err
	}
	seriesPainter := result.seriesPainter
	width := seriesPainter.Width()
	height := seriesPainter.Height()
	yRange := result.axisRanges[0]

	for index := range seriesList {
		series := seriesList[index]
		fillColor := opt.Theme.GetSeriesColor(series.index)
		if !series.Style.FillColor.IsZero() {
			fillColor = series.Style.FillColor
		}
		points := layout.bandPoints(index, width, height, yRange.min, yRange.max)
		// fill only: every band is a closed polygon and neighboring
		// bands reuse the same boundary pixels, no stroke is needed.
		// The polygon is closed explicitly because the raster (png)
		// renderer does not close fill paths implicitly like svg.
		seriesPainter.SetDrawingStyle(Style{
			FillColor: fillColor,
		})
		for i, point := range points {
			if i == 0 {
				seriesPainter.MoveTo(point.X, point.Y)
			} else {
				seriesPainter.LineTo(point.X, point.Y)
			}
		}
		seriesPainter.Close()
		seriesPainter.Fill()
	}
	return p.box, nil
}

// Render renders the stream chart
func (s *streamChart) Render() (Box, error) {
	p := s.p
	opt := s.opt
	seriesList := opt.SeriesList.Filter(ChartTypeStream)
	// The free baseline means the values on the y axis are the
	// stacked boundaries rather than the raw series values.
	layout, err := newStreamLayout(seriesList, opt.Baseline, opt.XValues)
	if err != nil {
		return BoxZero, err
	}
	yAxisOptions := opt.YAxisOptions
	if len(yAxisOptions) == 0 {
		yAxisOptions = []YAxisOption{{}}
	}
	// Pin the axis range to the boundary extent unless the caller
	// has explicitly configured min/max.
	span := layout.maxValue - layout.minValue
	pad := span * 0.05
	if yAxisOptions[0].Min == nil {
		min := layout.minValue - pad
		yAxisOptions[0].Min = &min
	}
	if yAxisOptions[0].Max == nil {
		max := layout.maxValue + pad
		yAxisOptions[0].Max = &max
	}
	xAxis := opt.XAxis
	// time series points sit on their ticks, no category gap
	if xAxis.BoundaryGap == nil {
		xAxis.BoundaryGap = FalseFlag()
	}
	renderResult, err := defaultRender(p, defaultRenderOption{
		Theme:              opt.Theme,
		Padding:            opt.Padding,
		SeriesList:         seriesList,
		XAxis:              xAxis,
		YAxisOptions:       yAxisOptions,
		TitleOption:        opt.Title,
		LegendOption:       opt.Legend,
		backgroundIsFilled: opt.backgroundIsFilled,
		fixedRange:         true,
	})
	if err != nil {
		return BoxZero, err
	}
	return s.render(renderResult, seriesList)
}
