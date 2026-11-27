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
	"github.com/wcharczuk/go-chart/v2"
)

const (
	// defaultWindRoseDivideCount is the default count of radial rings
	defaultWindRoseDivideCount = 4
	// defaultWindRoseSectorWidth is the default sector width in degrees
	defaultWindRoseSectorWidth = 30.0
	// fullAngle is the angle of a full circle in degrees
	fullAngle = 360.0
)

// WindRoseRadialAxisOption is the option of the radial (value) axis of a wind rose chart
type WindRoseRadialAxisOption struct {
	// The flag for show radial axis, set this to *false will hide it
	Show *bool
	// The count of radial rings (excluding the center), default value is 4
	DivideCount int
	// The maximum value of radial axis. When it is empty, the maximum
	// value is calculated from the aggregated weights automatically.
	Max *float64
	// The label list of radial axis, the length should be DivideCount + 1.
	// When it is empty, numeric labels are generated automatically.
	Data []string
	// The font size of radial axis label
	FontSize float64
	// The color of radial axis label
	FontColor Color
	// The color of radial grid line
	SplitLineColor Color
}

// WindRoseChartOption is the option of wind rose chart
type WindRoseChartOption struct {
	// The theme
	Theme ColorPalette
	// The font
	Font *truetype.Font
	// The data series list
	SeriesList SeriesList
	// The padding of chart
	Padding Box
	// The option of title
	Title TitleOption
	// The legend option
	Legend LegendOption
	// The width of sector in degrees, it must be greater than 0 and
	// divide 360 evenly, default value is 30
	SectorWidth float64
	// The labels of angular axis, which are rendered outside of the
	// outer ring from north clockwise. The default value is the compass
	// labels (N, NE, E, ...). Set Show of AngularAxis to false to hide.
	AngularLabels []string
	// The flag for show angular axis (the labels and spokes),
	// set this to *false will hide it
	AngularAxisShow *bool
	// The radial axis option
	RadialAxis WindRoseRadialAxisOption
	// Radius for wind rose chart, e.g.: 40%, default is "40%"
	Radius string
	// background is filled
	backgroundIsFilled bool
}

type windRoseChart struct {
	p   *Painter
	opt *WindRoseChartOption
}

// windRoseSector is an aggregated sector of the wind rose chart
type windRoseSector struct {
	// start angle in radians, 0 means north and increases clockwise
	start float64
	// delta angle in radians, always positive
	delta float64
	// the stacked values of each series, the last one is the total
	values []float64
}

// windRosePoint converts a polar point to a cartesian point.
// The angle uses the wind convention: 0 means north and positive values go
// clockwise. The underlying renderer measures angles from the east axis, so
// a quarter turn is subtracted before converting to cartesian coordinates.
func windRosePoint(center Point, radius, windAngle float64) Point {
	rendererAngle := windAngle - math.Pi/2
	return Point{
		X: center.X + int(radius*math.Cos(rendererAngle)),
		Y: center.Y + int(radius*math.Sin(rendererAngle)),
	}
}

// NewWindRoseChart returns a wind rose chart renderer
func NewWindRoseChart(p *Painter, opt WindRoseChartOption) *windRoseChart {
	if opt.Theme == nil {
		opt.Theme = defaultTheme
	}
	return &windRoseChart{
		p:   p,
		opt: &opt,
	}
}

// normalizeWindRoseAngle normalizes an angle to [0, 360).
// NaN is returned as NaN so that the caller can reject it.
func normalizeWindRoseAngle(angle float64) float64 {
	if math.IsNaN(angle) {
		return math.NaN()
	}
	if math.IsInf(angle, 0) {
		return math.NaN()
	}
	v := math.Mod(angle, fullAngle)
	if v < 0 {
		v += fullAngle
	}
	return v
}

// aggregateWindRose aggregates the series data into sectors.
// sectorWidth must be greater than 0 and divide 360 evenly.
// Angles are in degrees [0, 360): 0 means north and increases clockwise.
// The sectors are centered on the compass directions, i.e. the first sector
// is centered on north and spans [-sectorWidth/2, sectorWidth/2). Therefore
// 359 and 1 both belong to the north sector when the width is 90.
// A value exactly on a sector boundary belongs to the sector starting at
// that boundary (the lower angle sector), e.g. with a 30 degree width the
// value 45 belongs to the [45, 75) sector.
func aggregateWindRose(seriesList SeriesList, sectorWidth float64) ([]windRoseSector, error) {
	if sectorWidth <= 0 || sectorWidth > fullAngle {
		return nil, errors.New("The sector width of wind rose should be greater than 0 and not greater than 360")
	}
	sectorCount := int(math.Round(fullAngle / sectorWidth))
	// The width must divide 360 evenly
	if math.Abs(float64(sectorCount)*sectorWidth-fullAngle) > 1e-9 {
		return nil, errors.New("The sector width of wind rose should divide 360 evenly")
	}
	sectors := make([]windRoseSector, sectorCount)
	delta := sectorWidth * math.Pi / 180.0
	halfWidth := sectorWidth / 2
	for index := range sectors {
		start := float64(index)*delta - halfWidth*math.Pi/180.0
		if sectorCount == 1 {
			// The single full circle sector is started from north, its
			// start angle does not affect the shape but keeps it deterministic
			start = 0
		}
		sectors[index] = windRoseSector{
			// The first sector starts at -halfWidth so that north (0) is
			// the center of the first sector
			start: start,
			delta: delta,
			// values[0] is always zero, values[i+1] stores the stacked total
			// of the first i+1 series
			values: make([]float64, len(seriesList)+1),
		}
	}
	for seriesIndex, series := range seriesList {
		for _, item := range series.Data {
			if item.Value < 0 {
				return nil, errors.New("The weight of wind rose should not be negative")
			}
			angle := normalizeWindRoseAngle(item.Angle)
			if math.IsNaN(angle) {
				return nil, errors.New("The angle of wind rose should be a finite number")
			}
			// Shift by half a sector so that index 0 is centered on north
			sectorIndex := int(math.Floor(math.Mod(angle+halfWidth, fullAngle)/sectorWidth + 1e-9))
			if sectorIndex >= sectorCount {
				// Handles the value next to the upper boundary, e.g. 360
				sectorIndex = sectorCount - 1
			}
			sectors[sectorIndex].values[seriesIndex+1] += item.Value
		}
	}
	// Convert the per series value to stacked values
	for index := range sectors {
		for i := 1; i < len(sectors[index].values); i++ {
			sectors[index].values[i] += sectors[index].values[i-1]
		}
	}
	return sectors, nil
}

// windRoseRadialValues calculates the radial tick values.
// The ticks always start from zero, which keeps the center of the polar
// coordinate aligned with the value zero.
func windRoseRadialValues(maxValue float64, divideCount int, max *float64) []float64 {
	if divideCount <= 0 {
		divideCount = defaultWindRoseDivideCount
	}
	axisMax := maxValue
	if max != nil && *max >= maxValue {
		axisMax = *max
	}
	if axisMax <= 0 {
		// All of the weights are zero, use a unit radial scale so that
		// the grid can still be rendered
		axisMax = float64(divideCount)
	}
	// Snap the max value to a multiple of divideCount, which makes the
	// radius align with every ring exactly like the radar chart does
	unit := ceilFloatToInt(axisMax / float64(divideCount))
	if unit <= 0 {
		unit = 1
	}
	values := make([]float64, divideCount+1)
	for index := range values {
		values[index] = float64(unit * index)
	}
	return values
}

func (w *windRoseChart) render(result *defaultRenderResult, seriesList SeriesList) (Box, error) {
	opt := w.opt
	theme := opt.Theme
	painter := result.seriesPainter

	sectorWidth := opt.SectorWidth
	if sectorWidth == 0 {
		sectorWidth = defaultWindRoseSectorWidth
	}
	sectors, err := aggregateWindRose(seriesList, sectorWidth)
	if err != nil {
		return BoxZero, err
	}
	seriesCount := len(seriesList)
	maxValue := float64(0)
	for _, sector := range sectors {
		if sector.values[seriesCount] > maxValue {
			maxValue = sector.values[seriesCount]
		}
	}
	radialOpt := opt.RadialAxis
	radialValues := windRoseRadialValues(
		maxValue,
		radialOpt.DivideCount,
		radialOpt.Max,
	)
	divideCount := len(radialValues) - 1
	axisMax := radialValues[divideCount]
	axisMin := radialValues[0]

	cx := painter.Width() >> 1
	cy := painter.Height() >> 1
	diameter := chart.MinInt(painter.Width(), painter.Height())
	radius := getRadius(float64(diameter), opt.Radius)
	// Snap the radius to the rings so that the outer ring and the ticks align
	ringRadius := float64(int(radius / float64(divideCount)))
	radius = ringRadius * float64(divideCount)
	center := Point{X: cx, Y: cy}

	// radiusOf converts a value to its pixel radius
	radiusOf := func(value float64) float64 {
		if axisMax <= axisMin {
			return 0
		}
		return (value - axisMin) / (axisMax - axisMin) * radius
	}

	splitLineColor := radialOpt.SplitLineColor
	if splitLineColor.IsZero() {
		splitLineColor = theme.GetAxisSplitLineColor()
	}

	// Draw the radial grid rings
	painter.OverrideDrawingStyle(Style{
		StrokeColor: splitLineColor,
		StrokeWidth: 1,
	})
	for index := 1; index <= divideCount; index++ {
		r := ringRadius * float64(index)
		painter.Circle(r, cx, cy).Stroke()
	}

	// Draw the angular spokes and labels
	showAngularAxis := !isFalse(opt.AngularAxisShow)
	if showAngularAxis {
		painter.OverrideDrawingStyle(Style{
			StrokeColor: splitLineColor,
			StrokeWidth: 1,
		})
		for index := range sectors {
			point := windRosePoint(center, radius, sectors[index].start)
			painter.MoveTo(cx, cy)
			painter.LineTo(point.X, point.Y)
			painter.Stroke()
		}
		labels := opt.AngularLabels
		if len(labels) == 0 {
			labels = defaultWindRoseAngularLabels(len(sectors))
		}
		if len(labels) == len(sectors) {
			fontSize := labelFontSize
			painter.OverrideTextStyle(Style{
				FontColor: theme.GetTextColor(),
				FontSize:  float64(fontSize),
				Font:      opt.Font,
			})
			labelRadius := radius + 12
			for index, label := range labels {
				if label == "" {
					continue
				}
				// The label is placed at the center angle of its sector
				angle := sectors[index].start + sectors[index].delta/2
				point := windRosePoint(center, labelRadius, angle)
				textBox := painter.MeasureText(label)
				x := point.X - textBox.Width()>>1
				y := point.Y + textBox.Height()>>1 - 1
				painter.Text(label, x, y)
			}
		}
	}

	// Draw the sectors (stacked series from inner to outer)
	sectorCount := len(sectors)
	gap := 0.0
	if sectorCount > 1 {
		// A 2px surface gap between adjacent sectors
		gap = 2.0 / radius
	}
	for seriesIndex := 0; seriesIndex < seriesCount; seriesIndex++ {
		color := theme.GetSeriesColor(seriesList[seriesIndex].index)
		// For a full circle sector, the inner disk is repainted with the
		// previous series color (or the background color for the first one)
		innerColor := theme.GetBackgroundColor()
		if seriesIndex > 0 {
			innerColor = theme.GetSeriesColor(seriesList[seriesIndex-1].index)
		}
		painter.OverrideDrawingStyle(Style{
			StrokeWidth: 1,
			StrokeColor: color,
			FillColor:   color,
		})
		for _, sector := range sectors {
			innerValue := sector.values[seriesIndex]
			outerValue := sector.values[seriesIndex+1]
			if outerValue <= innerValue {
				// Zero weight, nothing to draw for this layer
				continue
			}
			innerRadius := radiusOf(innerValue)
			outerRadius := radiusOf(outerValue)
			start := sector.start
			delta := sector.delta
			if gap > 0 && sectorCount > 1 {
				start += gap / 2
				delta -= gap
			}
			w.drawSector(center, innerRadius, outerRadius, start, delta, color, innerColor)
		}
	}

	// Draw the radial axis labels on the east spoke (90 degrees clockwise
	// from north), which avoids overlapping the north angular label
	if !isFalse(radialOpt.Show) {
		fontSize := radialOpt.FontSize
		if fontSize == 0 {
			fontSize = smallLabelFontSize
		}
		fontColor := radialOpt.FontColor
		if fontColor.IsZero() {
			fontColor = theme.GetTextColor()
		}
		painter.OverrideTextStyle(Style{
			FontColor: fontColor,
			FontSize:  fontSize,
			Font:      opt.Font,
		})
		for index := 1; index <= divideCount; index++ {
			r := ringRadius * float64(index)
			// The label is anchored to the ring on the east spoke (90 degrees
			// clockwise from north), which avoids the north angular label
			label := ""
			if index < len(radialOpt.Data) {
				label = radialOpt.Data[index]
			}
			if label == "" {
				label = strconv.Itoa(int(radialValues[index]))
			}
			textBox := painter.MeasureText(label)
			anchor := windRosePoint(center, r, math.Pi/2)
			// Place the text just inside the ring and below the east spoke so
			// that it never reads as a minus sign and stays clear of the
			// outer angular label
			textX := anchor.X - textBox.Width() - 2
			textY := anchor.Y + 2
			if index == divideCount {
				// Pull the outermost label inward to avoid the 90 degree label
				textX = anchor.X - textBox.Width() - 8
			}
			painter.Text(label, textX, textY)
		}
	}

	return w.p.box, nil
}

// drawSector draws an annular sector. When innerRadius is zero, a full pie
// wedge is drawn. The start and delta angles use the wind convention:
// 0 means north and positive values go clockwise.
// fillColor is the color of the current series, innerColor is the color used
// to repaint the inner disk when the sector spans a full circle.
func (w *windRoseChart) drawSector(center Point, innerRadius, outerRadius, start, delta float64, fillColor, innerColor Color) {
	p := w.p
	cx, cy := center.X, center.Y
	// A full circle is drawn as circles, an arc cannot span 360 degrees.
	// There is only one sector in this case, so the inner disk is repainted
	// with the previous series color to reconstruct the stacked rings.
	if delta >= 2*math.Pi-1e-9 {
		p.OverrideDrawingStyle(Style{
			StrokeWidth: 1,
			StrokeColor: fillColor,
			FillColor:   fillColor,
		})
		p.Circle(outerRadius, cx, cy).FillStroke()
		if innerRadius > 0 {
			p.OverrideDrawingStyle(Style{
				StrokeWidth: 1,
				StrokeColor: fillColor,
				FillColor:   innerColor,
			})
			p.Circle(innerRadius, cx, cy).FillStroke()
		}
		return
	}
	// The renderer measures angles from the east axis
	rendererStart := start - math.Pi/2
	if innerRadius <= 0 {
		p.MoveTo(cx, cy)
		p.ArcTo(cx, cy, outerRadius, outerRadius, rendererStart, delta)
		p.LineTo(cx, cy)
		p.Close()
		p.FillStroke()
		return
	}
	// Annular sector: outer arc clockwise from start to end, then the
	// inner arc clockwise from end back to start. Both arcs use positive
	// (clockwise) delta because the SVG renderer always sweeps clockwise.
	outerStart := windRosePoint(center, outerRadius, start)
	endAngle := start + delta
	p.MoveTo(outerStart.X, outerStart.Y)
	p.ArcTo(cx, cy, outerRadius, outerRadius, rendererStart, delta)
	innerEnd := windRosePoint(center, innerRadius, endAngle)
	p.LineTo(innerEnd.X, innerEnd.Y)
	p.ArcTo(cx, cy, innerRadius, innerRadius, rendererStart, delta)
	p.Close()
	p.FillStroke()
}

// defaultWindRoseAngularLabels returns the default compass labels.
// When the sector count is not 4, 8 or 16, numeric degree labels are returned.
func defaultWindRoseAngularLabels(sectorCount int) []string {
	labels := make([]string, sectorCount)
	if sectorCount == 4 {
		copy(labels, []string{"N", "E", "S", "W"})
		return labels
	}
	if sectorCount == 8 {
		copy(labels, []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"})
		return labels
	}
	if sectorCount == 16 {
		copy(labels, []string{
			"N", "NNE", "NE", "ENE",
			"E", "ESE", "SE", "SSE",
			"S", "SSW", "SW", "WSW",
			"W", "WNW", "NW", "NNW",
		})
		return labels
	}
	width := fullAngle / float64(sectorCount)
	for index := range labels {
		// The first sector is centered on north, so the center angle of
		// each sector is index * width (normalized to [0, 360))
		centerAngle := math.Mod(float64(index)*width, fullAngle)
		labels[index] = strconv.Itoa(int(math.Round(centerAngle))) + "°"
	}
	return labels
}

// Render renders the wind rose chart
func (w *windRoseChart) Render() (Box, error) {
	opt := w.opt
	renderResult, err := defaultRender(w.p, defaultRenderOption{
		Theme:      opt.Theme,
		Padding:    opt.Padding,
		SeriesList: opt.SeriesList,
		XAxis: XAxisOption{
			Show: FalseFlag(),
		},
		YAxisOptions: []YAxisOption{
			{
				Show: FalseFlag(),
			},
		},
		TitleOption:        opt.Title,
		LegendOption:       opt.Legend,
		backgroundIsFilled: opt.backgroundIsFilled,
	})
	if err != nil {
		return BoxZero, err
	}
	seriesList := opt.SeriesList.Filter(ChartTypeWindRose)
	return w.render(renderResult, seriesList)
}
