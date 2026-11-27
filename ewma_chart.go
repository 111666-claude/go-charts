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
	"strconv"

	"github.com/golang/freetype/truetype"
)

// EWMAOption is the option of EWMA(exponentially weighted moving average)
// process control chart computation.
type EWMAOption struct {
	// The ordered observations. The null value(see SetNullValue), NaN and
	// Inf are treated as missing observations: they do not update the EWMA
	// state nor the sample count, and leave a gap in the EWMA line.
	Values []float64
	// The baseline(in-control) mean of the process
	Mean float64
	// The baseline(in-control) standard deviation of the process, it should
	// be a finite value >= 0. Zero variance collapses the control limits to
	// the center line, any EWMA value not equal to the mean is out of control.
	StdDev float64
	// The smoothing factor(lambda), it should be in (0, 1]
	Lambda float64
	// The width of control limits in sigma, default value is 3
	LimitSigma float64
	// The indexes of observations where the EWMA state and the sample count
	// are reset to the baseline, it splits the observations into segments
	ResetIndexes []int
}

// EWMAPoint is the computed result of one observation
type EWMAPoint struct {
	// The EWMA value of the observation, it is NaN if the observation is missing
	Value float64
	// The upper control limit, it varies with the sample count of the segment
	UCL float64
	// The lower control limit
	LCL float64
	// The center line value, it always equals the baseline mean
	Center float64
	// The flag for missing observation
	Missing bool
	// The flag for the EWMA value out of the control limits
	OutOfControl bool
}

func isEWMAMissingValue(v float64) bool {
	return v == nullValue || math.IsNaN(v) || math.IsInf(v, 0)
}

// EWMAProcess computes the EWMA value and the control limits of every
// observation. The EWMA starts from the baseline mean:
//
//	z(0) = mean, z(i) = lambda*x(i) + (1-lambda)*z(i-1)
//
// and the control limits of the i-th valid observation(n) of a segment are:
//
//	mean ± limitSigma*stdDev*sqrt(lambda/(2-lambda)*(1-(1-lambda)^(2n)))
func EWMAProcess(opt EWMAOption) ([]EWMAPoint, error) {
	if math.IsNaN(opt.Lambda) || opt.Lambda <= 0 || opt.Lambda > 1 {
		return nil, errors.New("ewma: lambda should be in (0, 1]")
	}
	if math.IsNaN(opt.Mean) || math.IsInf(opt.Mean, 0) {
		return nil, errors.New("ewma: mean should be a finite value")
	}
	if math.IsNaN(opt.StdDev) || math.IsInf(opt.StdDev, 0) || opt.StdDev < 0 {
		return nil, errors.New("ewma: std dev should be a finite value >= 0")
	}
	limitSigma := opt.LimitSigma
	if limitSigma == 0 {
		limitSigma = 3
	}
	if math.IsNaN(limitSigma) || math.IsInf(limitSigma, 0) || limitSigma < 0 {
		return nil, errors.New("ewma: limit sigma should be a finite value >= 0")
	}
	resetDict := make(map[int]bool, len(opt.ResetIndexes))
	for _, index := range opt.ResetIndexes {
		if index < 0 || index >= len(opt.Values) {
			return nil, errors.New("ewma: reset index out of range")
		}
		resetDict[index] = true
	}

	points := make([]EWMAPoint, len(opt.Values))
	z := opt.Mean
	// the count of valid observations of the current segment
	n := 0
	oneMinusLambda := 1 - opt.Lambda
	limitBase := limitSigma * opt.StdDev * math.Sqrt(opt.Lambda/(2-opt.Lambda))
	for i, v := range opt.Values {
		if resetDict[i] {
			z = opt.Mean
			n = 0
		}
		point := EWMAPoint{
			Center: opt.Mean,
		}
		if isEWMAMissingValue(v) {
			point.Missing = true
			point.Value = math.NaN()
		} else {
			n++
			z = opt.Lambda*v + oneMinusLambda*z
			point.Value = z
		}
		// 控制限随当前分段的有效样本数变化，缺测点不更新样本数，
		// 因此缺测点处的控制限与前一个有效点一致，保证界限线连续
		width := limitBase * math.Sqrt(1-math.Pow(oneMinusLambda, float64(2*n)))
		point.UCL = opt.Mean + width
		point.LCL = opt.Mean - width
		if !point.Missing && (point.Value > point.UCL || point.Value < point.LCL) {
			point.OutOfControl = true
		}
		points[i] = point
	}
	return points, nil
}

// EWMAChartOption is the option of EWMA process control chart
type EWMAChartOption struct {
	// The theme
	Theme ColorPalette
	// The font size
	Font *truetype.Font
	// The option of title
	Title TitleOption
	// The legend option, the default data is
	// ["Raw", "EWMA", "UCL", "LCL", "Center"]
	Legend LegendOption
	// The x axis option, if the data is empty, the index(1..n) of
	// observations is used
	XAxis XAxisOption
	// The y axis option list
	YAxisOptions []YAxisOption
	// The padding of chart
	Padding Box
	// The output type of chart, "svg" or "png", default value is "svg"
	Type string
	// The width of chart, default width is 600
	Width int
	// The height of chart, default height is 400
	Height int
	// The background color of chart
	BackgroundColor Color
	// The option of EWMA computation
	EWMA EWMAOption
	// The flag for show symbol of the raw line, set this to *false will hide symbol
	SymbolShow *bool
	// The color of control limits(UCL/LCL), if not set, the 3rd and 4th
	// series colors of the theme are used
	LimitColor Color
	// The color of the center line, if not set, the 5th series color of
	// the theme is used
	CenterColor Color
	// The color of out-of-control marks, if not set, a red color is used
	OutOfControlColor Color
}

type ewmaChart struct {
	p   *Painter
	opt *EWMAChartOption
}

// NewEWMAChart returns a EWMA process control chart render
func NewEWMAChart(p *Painter, opt EWMAChartOption) *ewmaChart {
	if opt.Theme == nil {
		opt.Theme = defaultTheme
	}
	if opt.Font != nil {
		opt.Theme.SetFont(opt.Font)
	}
	return &ewmaChart{
		p:   p,
		opt: &opt,
	}
}

var defaultEWMAOutOfControlColor = Color{
	R: 0xee,
	G: 0x66,
	B: 0x66,
	A: 0xff,
}

func (e *ewmaChart) Render() (Box, error) {
	p := e.p
	opt := e.opt
	points, err := EWMAProcess(opt.EWMA)
	if err != nil {
		return BoxZero, err
	}
	count := len(points)
	if count == 0 {
		return BoxZero, errors.New("ewma: values can not be empty")
	}
	if len(opt.XAxis.Data) == 0 {
		data := make([]string, count)
		for i := range data {
			data[i] = strconv.Itoa(i + 1)
		}
		opt.XAxis.Data = data
	}

	// 原始值、EWMA、上下控制限与中心线分别作为一条曲线，
	// 保证y轴范围包含所有曲线，PNG与SVG共用同一刻度计算
	rawData := make([]SeriesData, count)
	ewmaData := make([]SeriesData, count)
	uclData := make([]SeriesData, count)
	lclData := make([]SeriesData, count)
	centerData := make([]SeriesData, count)
	for i, point := range points {
		rawValue := opt.EWMA.Values[i]
		if isEWMAMissingValue(rawValue) {
			rawValue = nullValue
		}
		rawData[i] = SeriesData{Value: rawValue}
		ewmaValue := nullValue
		if !point.Missing {
			ewmaValue = point.Value
		}
		ewmaData[i] = SeriesData{Value: ewmaValue}
		uclData[i] = SeriesData{Value: point.UCL}
		lclData[i] = SeriesData{Value: point.LCL}
		centerData[i] = SeriesData{Value: point.Center}
	}
	seriesList := SeriesList{
		{Type: ChartTypeLine, Data: rawData},
		{Type: ChartTypeLine, Data: ewmaData},
		{Type: ChartTypeLine, Data: uclData},
		{Type: ChartTypeLine, Data: lclData},
		{Type: ChartTypeLine, Data: centerData},
	}
	legendOption := opt.Legend
	if len(legendOption.Data) == 0 {
		legendOption.Data = []string{"Raw", "EWMA", "UCL", "LCL", "Center"}
	}

	// 背景色优先使用指定的颜色
	backgroundColor := opt.BackgroundColor
	if backgroundColor.IsZero() {
		backgroundColor = opt.Theme.GetBackgroundColor()
	}
	p.SetBackground(p.Width(), p.Height(), backgroundColor)

	renderResult, err := defaultRender(p, defaultRenderOption{
		Theme:        opt.Theme,
		Padding:      opt.Padding,
		SeriesList:   seriesList,
		XAxis:        opt.XAxis,
		YAxisOptions: opt.YAxisOptions,
		TitleOption:  opt.Title,
		LegendOption: legendOption,
		// 前置已设置背景色
		backgroundIsFilled: true,
	})
	if err != nil {
		return BoxZero, err
	}

	seriesPainter := renderResult.seriesPainter
	yRange := renderResult.axisRanges[0]

	boundaryGap := true
	if isFalse(opt.XAxis.BoundaryGap) {
		boundaryGap = false
	}
	xDivideCount := len(opt.XAxis.Data)
	if !boundaryGap {
		xDivideCount--
	}
	xDivideValues := autoDivide(seriesPainter.Width(), xDivideCount)
	xValues := make([]int, len(xDivideValues)-1)
	if boundaryGap {
		for i := 0; i < len(xDivideValues)-1; i++ {
			xValues[i] = (xDivideValues[i] + xDivideValues[i+1]) >> 1
		}
	} else {
		xValues = xDivideValues
	}
	toPoints := func(data []SeriesData) []Point {
		points := make([]Point, 0, len(data))
		for i, item := range data {
			if i >= len(xValues) {
				break
			}
			h := yRange.getRestHeight(item.Value)
			if item.Value == nullValue {
				h = int(math.MaxInt32)
			}
			points = append(points, Point{
				X: xValues[i],
				Y: h,
			})
		}
		return points
	}

	theme := opt.Theme
	rawColor := theme.GetSeriesColor(0)
	ewmaColor := theme.GetSeriesColor(1)
	uclColor := theme.GetSeriesColor(2)
	lclColor := theme.GetSeriesColor(3)
	if !opt.LimitColor.IsZero() {
		uclColor = opt.LimitColor
		lclColor = opt.LimitColor
	}
	centerColor := opt.CenterColor
	if centerColor.IsZero() {
		centerColor = theme.GetSeriesColor(4)
	}
	outOfControlColor := opt.OutOfControlColor
	if outOfControlColor.IsZero() {
		outOfControlColor = defaultEWMAOutOfControlColor
	}

	// 控制限与中心线使用虚线，与原始折线、EWMA曲线区分
	dashArray := []float64{6, 4}
	seriesPainter.SetDrawingStyle(Style{
		StrokeColor:     uclColor,
		StrokeWidth:     1,
		StrokeDashArray: dashArray,
	})
	seriesPainter.LineStroke(toPoints(uclData))
	seriesPainter.SetDrawingStyle(Style{
		StrokeColor:     lclColor,
		StrokeWidth:     1,
		StrokeDashArray: dashArray,
	})
	seriesPainter.LineStroke(toPoints(lclData))
	seriesPainter.SetDrawingStyle(Style{
		StrokeColor:     centerColor,
		StrokeWidth:     1,
		StrokeDashArray: dashArray,
	})
	seriesPainter.LineStroke(toPoints(centerData))

	// 原始折线带数据点
	rawPoints := toPoints(rawData)
	seriesPainter.SetDrawingStyle(Style{
		StrokeColor: rawColor,
		StrokeWidth: defaultStrokeWidth,
	})
	seriesPainter.LineStroke(rawPoints)
	if !isFalse(opt.SymbolShow) {
		dots := make([]Point, 0, len(rawPoints))
		for i, point := range rawPoints {
			if rawData[i].Value == nullValue {
				continue
			}
			dots = append(dots, point)
		}
		if theme.IsDark() {
			seriesPainter.SetDrawingStyle(Style{
				StrokeColor: rawColor,
				StrokeWidth: 1,
				FillColor:   rawColor,
			})
		} else {
			seriesPainter.SetDrawingStyle(Style{
				StrokeColor: rawColor,
				StrokeWidth: 1,
				FillColor:   backgroundColor,
			})
		}
		seriesPainter.Dots(dots)
	}

	// EWMA曲线使用更粗的实线
	seriesPainter.SetDrawingStyle(Style{
		StrokeColor: ewmaColor,
		StrokeWidth: defaultStrokeWidth + 1,
	})
	seriesPainter.LineStroke(toPoints(ewmaData))

	// 越界点使用醒目的圆点标记
	seriesPainter.SetDrawingStyle(Style{
		FillColor:   outOfControlColor,
		StrokeColor: outOfControlColor,
		StrokeWidth: 1,
	})
	for i, point := range points {
		if !point.OutOfControl || i >= len(xValues) {
			continue
		}
		seriesPainter.Circle(4, xValues[i], yRange.getRestHeight(point.Value))
	}
	seriesPainter.FillStroke()

	return p.box, nil
}

// EWMARender renders a EWMA process control chart
func EWMARender(opt EWMAChartOption) (*Painter, error) {
	if opt.Theme == nil {
		opt.Theme = defaultTheme
	}
	if opt.Type == "" {
		opt.Type = ChartOutputSVG
	}
	if opt.Width <= 0 {
		opt.Width = defaultChartWidth
	}
	if opt.Height <= 0 {
		opt.Height = defaultChartHeight
	}
	if opt.Font == nil {
		opt.Font, _ = GetDefaultFont()
	}
	if opt.Padding.IsZero() {
		opt.Padding = Box{
			Top:    20,
			Right:  20,
			Bottom: 20,
			Left:   20,
		}
	}
	p, err := NewPainter(PainterOptions{
		Type:   opt.Type,
		Width:  opt.Width,
		Height: opt.Height,
		Font:   opt.Font,
	})
	if err != nil {
		return nil, err
	}
	_, err = NewEWMAChart(p, opt).Render()
	if err != nil {
		return nil, err
	}
	return p, nil
}
