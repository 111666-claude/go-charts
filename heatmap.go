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

	"github.com/dustin/go-humanize"
	"github.com/golang/freetype/truetype"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

const (
	// The maximum count of row or column for a heat map,
	// which protects the renderer from extremely large matrices.
	maxHeatMapDimension = 1000
	// Default split count of heat map legend.
	defaultHeatMapLegendSplit = 5
	// Default height(px) of horizontal legend color bar.
	defaultHeatMapLegendBarHeight = 12
	// Default width(px) of vertical legend color bar.
	defaultHeatMapLegendBarWidth = 12
	// The gap between legend color bar and tick text.
	legendTextGap = 5
	// The size of missing value swatch.
	legendNullSwatchSize = 12
)

// HeatMapLegendOption is the option of heat map legend (visual map).
type HeatMapLegendOption struct {
	// The flag for show legend, set this to *false will hide legend.
	Show *bool
	// The layout orientation of legend, it can be "horizontal" or "vertical".
	// The default value is "horizontal".
	Orient string
	// The count of color segments, default value is 5.
	Split int
	// The font size of tick text.
	FontSize float64
	// The font color of tick text.
	FontColor Color
	// The width of vertical legend color bar.
	BarWidth int
	// The height of horizontal legend color bar.
	BarHeight int
}

// HeatMapOption is the option of heat map chart.
type HeatMapOption struct {
	// The output type of chart, "svg" or "png", default value is "png".
	Type string
	// The theme of chart, "light", "dark", "grafana" and "ant".
	Theme string
	// The font family, which should be installed first.
	FontFamily string
	Font       *truetype.Font
	// The width of chart, default width is 600.
	Width int
	// The height of chart, default height is 400.
	Height int
	// The padding for chart.
	Padding Box
	// The title option.
	Title TitleOption
	// The row labels (the y axis category list).
	Rows []string
	// The column labels (the x axis category list).
	Columns []string
	// The value matrix. Rows are outer slice, columns are inner slice.
	// Non-finite values (NaN, +Inf, -Inf) and values equal to the null
	// value are treated as missing value.
	Data [][]float64
	// The color list of color scale. Empty means the built-in scale:
	// diverging blue-yellow-red for mixed positive/negative data,
	// sequential blue for non-negative data,
	// sequential red for non-positive data.
	Colors []string
	// The minimum value of color scale, default is the minimum of data.
	Min *float64
	// The maximum value of color scale, default is the maximum of data.
	Max *float64
	// The background color of missing value cell.
	NullColor Color
	// The text of missing value cell, default value is "-".
	NullText string
	// The flag for show the value label in each cell, default is true.
	// Labels are hidden automatically when the cell is too small.
	LabelShow *bool
	// The font size of cell label.
	LabelFontSize float64
	// The border width of cell, default value is 0 (no border).
	BorderWidth int
	// The border color of cell.
	BorderColor Color
	// The text rotation of column labels in radians.
	// A negative value means rotating counterclockwise, e.g. -math.Pi/4.
	// Zero means no rotation, long labels are rotated automatically.
	LabelRotation float64
	// The margin between axis label and heat map grid.
	LabelMargin int
	// The legend option.
	Legend HeatMapLegendOption
	// The background color of chart.
	BackgroundColor Color
	// The custom value formatter for cell label and legend tick.
	ValueFormatter ValueFormatter
}

// heatMapScale maps a finite value to a color of the configured color scale.
type heatMapScale struct {
	min float64
	max float64
	// equal is true when min == max (all finite values are constant).
	equal bool
	// diverging is true when min < 0 && max > 0, zero is mapped to
	// the middle color stop in this mode.
	diverging bool
	stops     []Color
}

func lerpColor(a, b Color, t float64) Color {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	// Convert to float64 before subtraction, otherwise uint8
	// subtraction wraps around when b < a (e.g. 165 - 253 => 168).
	ar, ag, ab := float64(a.R), float64(a.G), float64(a.B)
	br, bg, bb := float64(b.R), float64(b.G), float64(b.B)
	return Color{
		R: uint8(math.Round(ar + (br-ar)*t)),
		G: uint8(math.Round(ag + (bg-ag)*t)),
		B: uint8(math.Round(ab + (bb-ab)*t)),
		A: 255,
	}
}

// defaultHeatMapColors returns the built-in color stops for the data range.
func defaultHeatMapColors(min, max float64) []string {
	// Diverging blue -> light yellow -> red, zero is the middle stop.
	if min < 0 && max > 0 {
		return []string{
			"#313695",
			"#74add1",
			"#ffffbf",
			"#fdae61",
			"#a50026",
		}
	}
	// Non-positive data, sequential red.
	if max <= 0 {
		return []string{
			"#fff5f0",
			"#fcbba1",
			"#fb6a4a",
			"#cb181d",
			"#67000d",
		}
	}
	// Non-negative data, sequential blue.
	return []string{
		"#f7fbff",
		"#c6dbef",
		"#6baed6",
		"#2171b5",
		"#08306b",
	}
}

// newHeatMapScale creates a color scale. min and max are the finite range
// of the data, colors are the configured color hex list (may be empty).
func newHeatMapScale(min, max float64, optMin, optMax *float64, colors []string) *heatMapScale {
	s := &heatMapScale{
		min: min,
		max: max,
	}
	if optMin != nil && isFinite(*optMin) {
		s.min = *optMin
	}
	if optMax != nil && isFinite(*optMax) {
		s.max = *optMax
	}
	if s.min > s.max {
		s.min, s.max = s.max, s.min
	}
	s.equal = s.min == s.max
	s.diverging = s.min < 0 && s.max > 0
	if len(colors) == 0 {
		colors = defaultHeatMapColors(s.min, s.max)
	}
	s.stops = make([]Color, 0, len(colors))
	for _, c := range colors {
		s.stops = append(s.stops, parseColor(c))
	}
	return s
}

// color returns the mapped color of a finite value.
func (s *heatMapScale) color(value float64) Color {
	// Only one color stop, use it directly.
	if len(s.stops) == 1 {
		return s.stops[0]
	}
	t := 0.0
	switch {
	case s.equal:
		// All values are constant: use the middle color.
		idx := len(s.stops) / 2
		return s.stops[idx]
	case s.diverging:
		if value < 0 {
			if value <= s.min {
				return s.stops[0]
			}
			t = (value - s.min) / (0 - s.min)
			// Negative half: stops [0, middle].
			mid := len(s.stops) / 2
			pos := t * float64(mid)
			i := int(pos)
			if i >= mid {
				return s.stops[mid]
			}
			return lerpColor(s.stops[i], s.stops[i+1], pos-float64(i))
		}
		if value >= 0 {
			if value >= s.max {
				return s.stops[len(s.stops)-1]
			}
			t = value / s.max
			mid := len(s.stops) / 2
			pos := float64(mid) + t*float64(len(s.stops)-1-mid)
			i := int(pos)
			if i >= len(s.stops)-1 {
				return s.stops[len(s.stops)-1]
			}
			return lerpColor(s.stops[i], s.stops[i+1], pos-float64(i))
		}
	default:
		if value <= s.min {
			return s.stops[0]
		}
		if value >= s.max {
			return s.stops[len(s.stops)-1]
		}
		t = (value - s.min) / (s.max - s.min)
	}
	pos := t * float64(len(s.stops)-1)
	i := int(pos)
	if i >= len(s.stops)-1 {
		return s.stops[len(s.stops)-1]
	}
	return lerpColor(s.stops[i], s.stops[i+1], pos-float64(i))
}

// ticks returns the tick values of the color scale.
func (s *heatMapScale) ticks(split int) []float64 {
	if split < 1 {
		split = defaultHeatMapLegendSplit
	}
	values := make([]float64, split+1)
	if s.equal {
		for i := range values {
			values[i] = s.min
		}
		return values
	}
	step := (s.max - s.min) / float64(split)
	for i := 0; i <= split; i++ {
		values[i] = s.min + step*float64(i)
	}
	return values
}

// isFinite reports whether the value is neither NaN nor infinity.
func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// isMissingValue reports whether the value is a missing value.
func isMissingValue(value float64) bool {
	return !isFinite(value) || value == nullValue
}

// heatMapLayout is the computed geometry of the heat map.
type heatMapLayout struct {
	rowCount    int
	columnCount int
	// The offset of the grid area inside the content painter.
	gridLeft int
	gridTop  int
	// The size of the grid area.
	gridWidth  int
	gridHeight int
	cellWidth  int
	cellHeight int
	// The rotated flag of column labels.
	columnLabelRotated bool
	// The maximum text width of column labels,
	// used to reserve space for the rotated labels.
	columnLabelMaxWidth int
	// The height reserved for the column label area.
	columnLabelAreaHeight int
}

// cellBox returns the box of the cell in the grid painter.
func (l *heatMapLayout) cellBox(row, column int) Box {
	left := column * l.cellWidth
	top := row * l.cellHeight
	return Box{
		Left:   left,
		Top:    top,
		Right:  left + l.cellWidth,
		Bottom: top + l.cellHeight,
	}
}

type heatMapChart struct {
	p   *Painter
	opt *HeatMapOption
}

// NewHeatMapChart returns a heat map renderer.
func NewHeatMapChart(p *Painter, opt HeatMapOption) *heatMapChart {
	return &heatMapChart{
		p:   p,
		opt: &opt,
	}
}

func (h *heatMapChart) theme() ColorPalette {
	return h.p.theme
}

func (h *heatMapChart) font() *truetype.Font {
	if h.opt.Font != nil {
		return h.opt.Font
	}
	return h.theme().GetFont()
}

func (h *heatMapChart) dimensions() (int, int, error) {
	rowCount := len(h.opt.Data)
	if rowCount == 0 {
		rowCount = len(h.opt.Rows)
	}
	columnCount := len(h.opt.Columns)
	if columnCount == 0 && len(h.opt.Data) != 0 {
		columnCount = len(h.opt.Data[0])
	}
	for _, row := range h.opt.Data {
		if len(row) > columnCount {
			columnCount = len(row)
		}
	}
	if rowCount > maxHeatMapDimension || columnCount > maxHeatMapDimension {
		return 0, 0, errors.New("the row or column count of heat map exceeds the maximum limit 1000")
	}
	return rowCount, columnCount, nil
}

// finiteMinMax returns the min and max of finite values,
// ok is false when there is no finite value.
func (h *heatMapChart) finiteMinMax() (min, max float64, ok bool) {
	first := true
	for _, row := range h.opt.Data {
		for _, value := range row {
			if isMissingValue(value) {
				continue
			}
			if first {
				min, max = value, value
				first = false
				continue
			}
			if value < min {
				min = value
			}
			if value > max {
				max = value
			}
		}
	}
	return min, max, !first
}

func (h *heatMapChart) labelMargin() int {
	if h.opt.LabelMargin > 0 {
		return h.opt.LabelMargin
	}
	return 5
}

func (h *heatMapChart) valueText(value float64) string {
	if h.opt.ValueFormatter != nil {
		return h.opt.ValueFormatter(value)
	}
	return humanize.FtoaWithDigits(value, 2)
}

func (h *heatMapChart) nullText() string {
	if h.opt.NullText != "" {
		return h.opt.NullText
	}
	return "-"
}

func (h *heatMapChart) nullColor() Color {
	if !h.opt.NullColor.IsZero() {
		return h.opt.NullColor
	}
	return h.theme().GetAxisSplitLineColor()
}

func (h *heatMapChart) labelShow() bool {
	return h.opt.LabelShow == nil || *h.opt.LabelShow
}

// computeLayout calculates the geometry of grid and axis labels.
func (h *heatMapChart) computeLayout(p *Painter, rowCount, columnCount int) *heatMapLayout {
	theme := h.theme()
	font := h.font()
	fontSize := theme.GetFontSize()
	labelMargin := h.labelMargin()

	layout := &heatMapLayout{
		rowCount:    rowCount,
		columnCount: columnCount,
	}

	textStyle := Style{
		Font:      font,
		FontSize:  fontSize,
		FontColor: theme.GetTextColor(),
	}

	// Row labels (y axis, right aligned).
	rowLabelWidth := 0
	if rowCount > 0 {
		p.SetTextStyle(textStyle)
		rowLabels := h.categoryLabels(h.opt.Rows, rowCount)
		w, _ := p.MeasureTextMaxWidthHeight(rowLabels)
		rowLabelWidth = w
	}

	// Column labels (x axis).
	columnLabels := h.categoryLabels(h.opt.Columns, columnCount)
	columnLabelHeight := 0
	if columnCount > 0 {
		p.SetTextStyle(textStyle)
		w, textHeight := p.MeasureTextMaxWidthHeight(columnLabels)
		layout.columnLabelMaxWidth = w
		columnLabelHeight = textHeight
		rotation := h.opt.LabelRotation
		rotated := rotation != 0
		if !rotated {
			// Auto rotate when the widest label would crowd its column.
			// A label wider than 80% of the column width is rotated to
			// keep adjacent labels from overlapping.
			availableWidth := p.Width() - rowLabelWidth - labelMargin
			columnWidth := availableWidth / columnCount
			if columnWidth <= 0 || w*5 > columnWidth*4 {
				rotated = true
				rotation = -math.Pi / 4
			}
		}
		layout.columnLabelRotated = rotated
		if rotated {
			// Reserve the full vertical projection of the longest label.
			// For a box of width w / height h rotated by angle theta, the
			// vertical extent is w*|sin| + h*|cos|.
			sin, cos := math.Sin(math.Abs(rotation)), math.Cos(math.Abs(rotation))
			columnLabelHeight = int(math.Ceil(float64(w)*sin)) + int(math.Ceil(float64(textHeight)*cos)) + labelMargin*2
		}
		layout.columnLabelAreaHeight = columnLabelHeight
	}

	gridLeft := 0
	if rowCount > 0 {
		gridLeft = rowLabelWidth + labelMargin
	}
	gridTop := 0
	// Rotated labels are centered on their column, so each end label
	// extends by half its horizontal projection; reserve that margin on
	// both sides to keep the end labels inside the canvas.
	rotatedSideMargin := 0
	if layout.columnLabelRotated && columnCount > 0 {
		rotatedSideMargin = int(math.Ceil(float64(layout.columnLabelMaxWidth) * math.Cos(math.Pi/4) / 2))
	}
	gridLeft += rotatedSideMargin
	gridWidth := p.Width() - gridLeft - rotatedSideMargin
	// The column label area is allocated first; the grid takes the
	// remaining space, so long labels never overflow the canvas or
	// overlap the grid.
	gridHeight := p.Height() - columnLabelHeight
	if gridWidth < 1 {
		gridWidth = 1
	}
	if gridHeight < 1 {
		gridHeight = 1
	}
	layout.gridLeft = gridLeft
	layout.gridTop = gridTop
	layout.gridWidth = gridWidth
	layout.gridHeight = gridHeight
	if columnCount > 0 {
		layout.cellWidth = gridWidth / columnCount
	}
	if rowCount > 0 {
		layout.cellHeight = gridHeight / rowCount
	}
	if layout.cellWidth < 1 {
		layout.cellWidth = 1
	}
	if layout.cellHeight < 1 {
		layout.cellHeight = 1
	}
	return layout
}

// categoryLabels returns labels of the given count,
// missing labels are generated as "1", "2" ...
func (h *heatMapChart) categoryLabels(labels []string, count int) []string {
	result := make([]string, count)
	for i := 0; i < count; i++ {
		if i < len(labels) {
			result[i] = labels[i]
		} else {
			result[i] = humanize.Ftoa(float64(i + 1))
		}
	}
	return result
}

// legendMetrics returns the occupied size of the legend inside the content area.
func (h *heatMapChart) legendMetrics(p *Painter, hasNull bool) (width, height int) {
	if isFalse(h.opt.Legend.Show) {
		return 0, 0
	}
	// Empty matrix has no legend.
	if len(h.opt.Data) == 0 {
		return 0, 0
	}
	split := h.opt.Legend.Split
	if split < 1 {
		split = defaultHeatMapLegendSplit
	}
	fontSize := h.opt.Legend.FontSize
	if fontSize == 0 {
		fontSize = smallLabelFontSize
	}
	fontColor := h.opt.Legend.FontColor
	if fontColor.IsZero() {
		fontColor = h.theme().GetTextColor()
	}
	p.SetTextStyle(Style{
		Font:      h.font(),
		FontSize:  fontSize,
		FontColor: fontColor,
	})
	min, max, ok := h.finiteMinMax()
	var tickTexts []string
	if ok {
		scale := newHeatMapScale(min, max, h.opt.Min, h.opt.Max, h.opt.Colors)
		for _, v := range scale.ticks(split) {
			tickTexts = append(tickTexts, h.valueText(v))
		}
	} else {
		tickTexts = []string{h.nullText()}
	}
	textMaxWidth, textMaxHeight := p.MeasureTextMaxWidthHeight(tickTexts)
	if h.opt.Legend.Orient == OrientVertical {
		barWidth := h.opt.Legend.BarWidth
		if barWidth <= 0 {
			barWidth = defaultHeatMapLegendBarWidth
		}
		width = barWidth + legendTextGap + textMaxWidth
		height = split * (textMaxHeight + 4)
		if hasNull {
			height += legendNullSwatchSize + legendTextGap + textMaxHeight
		}
		return width, height
	}
	barHeight := h.opt.Legend.BarHeight
	if barHeight <= 0 {
		barHeight = defaultHeatMapLegendBarHeight
	}
	height = barHeight + legendTextGap + textMaxHeight
	if hasNull {
		height += legendTextGap + legendNullSwatchSize
	}
	return 0, height
}

// renderLegend draws the legend. areaWidth/areaHeight is the content area size.
func (h *heatMapChart) renderLegend(p *Painter, areaWidth, areaHeight int) {
	if isFalse(h.opt.Legend.Show) || len(h.opt.Data) == 0 {
		return
	}
	split := h.opt.Legend.Split
	if split < 1 {
		split = defaultHeatMapLegendSplit
	}
	fontSize := h.opt.Legend.FontSize
	if fontSize == 0 {
		fontSize = smallLabelFontSize
	}
	fontColor := h.opt.Legend.FontColor
	if fontColor.IsZero() {
		fontColor = h.theme().GetTextColor()
	}
	font := h.font()
	p.SetTextStyle(Style{
		Font:      font,
		FontSize:  fontSize,
		FontColor: fontColor,
	})
	min, max, ok := h.finiteMinMax()
	hasNull := false
	for _, row := range h.opt.Data {
		for _, value := range row {
			if isMissingValue(value) {
				hasNull = true
				break
			}
		}
		if hasNull {
			break
		}
	}
	// No finite value: only draw the missing value swatch.
	if !ok {
		if hasNull {
			nullColor := h.nullColor()
			p.OverrideDrawingStyle(Style{
				FillColor: nullColor,
			}).Rect(Box{
				Left:   0,
				Top:    0,
				Right:  legendNullSwatchSize,
				Bottom: legendNullSwatchSize,
			})
			p.Text(h.nullText(), legendNullSwatchSize+legendTextGap, legendNullSwatchSize-2)
		}
		return
	}
	scale := newHeatMapScale(min, max, h.opt.Min, h.opt.Max, h.opt.Colors)
	ticks := scale.ticks(split)

	if h.opt.Legend.Orient == OrientVertical {
		h.renderVerticalLegend(p, scale, ticks, split, font, fontSize, fontColor, hasNull, areaHeight)
		return
	}
	h.renderHorizontalLegend(p, scale, ticks, split, font, fontSize, fontColor, hasNull, areaWidth)
}

func (h *heatMapChart) renderHorizontalLegend(p *Painter, scale *heatMapScale, ticks []float64, split int,
	font *truetype.Font, fontSize float64, fontColor Color, hasNull bool, areaWidth int) {
	barHeight := h.opt.Legend.BarHeight
	if barHeight <= 0 {
		barHeight = defaultHeatMapLegendBarHeight
	}
	segmentWidth := areaWidth / split
	if segmentWidth < 1 {
		segmentWidth = 1
	}
	// Draw color segments.
	for i := 0; i < split; i++ {
		value := ticks[i]
		if !scale.equal {
			value = ticks[i] + (ticks[i+1]-ticks[i])/2
		}
		c := scale.color(value)
		left := i * segmentWidth
		p.OverrideDrawingStyle(Style{
			FillColor: c,
		}).Rect(Box{
			Left:   left,
			Top:    0,
			Right:  left + segmentWidth,
			Bottom: barHeight,
		})
	}
	// Draw tick texts.
	textY := barHeight + legendTextGap + int(fontSize)
	for i, value := range ticks {
		text := h.valueText(value)
		box := p.MeasureText(text)
		x := i * segmentWidth
		if i == split {
			x = areaWidth - box.Width()
		} else if i != 0 {
			x -= box.Width() >> 1
		}
		if x < 0 {
			x = 0
		}
		p.Text(text, x, textY)
	}
	if hasNull {
		swatchY := textY + legendTextGap
		nullColor := h.nullColor()
		p.OverrideDrawingStyle(Style{
			FillColor: nullColor,
		}).Rect(Box{
			Left:   0,
			Top:    swatchY,
			Right:  legendNullSwatchSize,
			Bottom: swatchY + legendNullSwatchSize,
		})
		p.Text(h.nullText(), legendNullSwatchSize+legendTextGap, swatchY+legendNullSwatchSize-2)
	}
}

func (h *heatMapChart) renderVerticalLegend(p *Painter, scale *heatMapScale, ticks []float64, split int,
	font *truetype.Font, fontSize float64, fontColor Color, hasNull bool, areaHeight int) {
	barWidth := h.opt.Legend.BarWidth
	if barWidth <= 0 {
		barWidth = defaultHeatMapLegendBarWidth
	}
	barHeight := areaHeight
	if hasNull {
		barHeight -= legendNullSwatchSize + legendTextGap + int(fontSize)
	}
	if barHeight < split {
		barHeight = split
	}
	segmentHeight := barHeight / split
	if segmentHeight < 1 {
		segmentHeight = 1
	}
	// Draw color segments from top to bottom (max to min).
	for i := 0; i < split; i++ {
		value := ticks[split-i]
		if !scale.equal {
			value = ticks[split-i-1] + (ticks[split-i]-ticks[split-i-1])/2
		}
		c := scale.color(value)
		top := i * segmentHeight
		p.OverrideDrawingStyle(Style{
			FillColor: c,
		}).Rect(Box{
			Left:   0,
			Top:    top,
			Right:  barWidth,
			Bottom: top + segmentHeight,
		})
	}
	// Draw tick texts.
	for i, value := range ticks {
		text := h.valueText(value)
		box := p.MeasureText(text)
		y := (split-i)*segmentHeight - (segmentHeight+box.Height())/2
		if y < 0 {
			y = 0
		}
		p.Text(text, barWidth+legendTextGap, y+box.Height()-2)
	}
	if hasNull {
		swatchY := barHeight + legendTextGap
		nullColor := h.nullColor()
		p.OverrideDrawingStyle(Style{
			FillColor: nullColor,
		}).Rect(Box{
			Left:   0,
			Top:    swatchY,
			Right:  legendNullSwatchSize,
			Bottom: swatchY + legendNullSwatchSize,
		})
		p.Text(h.nullText(), legendNullSwatchSize+legendTextGap, swatchY+legendNullSwatchSize-2)
	}
}

// renderAxisLabels draws the row and column labels.
func (h *heatMapChart) renderAxisLabels(p *Painter, layout *heatMapLayout) {
	theme := h.theme()
	fontSize := theme.GetFontSize()
	style := Style{
		Font:      h.font(),
		FontSize:  fontSize,
		FontColor: theme.GetTextColor(),
	}
	p.SetTextStyle(style)
	labelMargin := h.labelMargin()

	// Row labels: right aligned, vertically centered.
	if layout.rowCount > 0 {
		rowLabels := h.categoryLabels(h.opt.Rows, layout.rowCount)
		for i, text := range rowLabels {
			box := p.MeasureText(text)
			x := layout.gridLeft - labelMargin - box.Width()
			y := i*layout.cellHeight + (layout.cellHeight+box.Height())/2 - 2
			p.Text(text, x, y)
		}
	}

	// Column labels.
	if layout.columnCount == 0 {
		return
	}
	columnLabels := h.categoryLabels(h.opt.Columns, layout.columnCount)
	rotation := h.opt.LabelRotation
	if layout.columnLabelRotated && rotation == 0 {
		rotation = -math.Pi / 4
	}
	if rotation != 0 {
		// The renderer rotates the text around its baseline start (x, y).
		// Position every label from its renderer-measured rotated box so
		// the result is identical for PNG and SVG:
		//   - the box is horizontally centered on its column;
		//   - the box bottom aligns with the bottom of the label area,
		//     so labels never overlap the grid above or the legend below.
		p.SetTextRotation(rotation)
		labelAreaBottom := layout.gridHeight + layout.columnLabelAreaHeight - labelMargin
		for i, text := range columnLabels {
			rotated := p.MeasureText(text)
			columnCenter := i*layout.cellWidth + layout.cellWidth/2
			anchorX := columnCenter - (rotated.Left+rotated.Right)/2
			anchorY := labelAreaBottom - rotated.Bottom
			p.Text(text, anchorX, anchorY)
		}
		p.ClearTextRotation()
		return
	}
	for i, text := range columnLabels {
		box := p.MeasureText(text)
		x := i*layout.cellWidth + (layout.cellWidth-box.Width())/2
		if x < 0 {
			x = 0
		}
		y := layout.gridHeight + labelMargin + box.Height()
		p.Text(text, x, y)
	}
}

// renderCells draws all the heat map cells.
func (h *heatMapChart) renderCells(gridPainter *Painter, layout *heatMapLayout, scale *heatMapScale) {
	theme := h.theme()
	borderColor := h.opt.BorderColor
	if borderColor.IsZero() {
		borderColor = theme.GetAxisSplitLineColor()
	}
	fontSize := h.opt.LabelFontSize
	if fontSize == 0 {
		fontSize = smallLabelFontSize
	}
	nullColor := h.nullColor()
	lightTextColor := drawing.Color{
		R: 51,
		G: 51,
		B: 51,
		A: 255,
	}
	darkTextColor := drawing.ColorWhite

	for row := 0; row < layout.rowCount; row++ {
		for column := 0; column < layout.columnCount; column++ {
			box := layout.cellBox(row, column)
			missing := row >= len(h.opt.Data) ||
				column >= len(h.opt.Data[row]) ||
				isMissingValue(h.opt.Data[row][column])
			var fill Color
			if missing {
				fill = nullColor
			} else {
				fill = scale.color(h.opt.Data[row][column])
			}
			cellStyle := Style{
				FillColor: fill,
			}
			if h.opt.BorderWidth > 0 {
				cellStyle.StrokeColor = borderColor
				cellStyle.StrokeWidth = float64(h.opt.BorderWidth)
			}
			gridPainter.OverrideDrawingStyle(cellStyle).Rect(box)

			if !h.labelShow() {
				continue
			}
			// Hide the label when the cell is too small.
			if box.Width() < 12 || box.Height() < 14 {
				continue
			}
			var text string
			var textColor Color
			if missing {
				text = h.nullText()
				textColor = theme.GetTextColor()
			} else {
				value := h.opt.Data[row][column]
				text = h.valueText(value)
				if isLightColor(fill) {
					textColor = lightTextColor
				} else {
					textColor = darkTextColor
				}
			}
			gridPainter.SetTextStyle(Style{
				Font:      h.font(),
				FontSize:  fontSize,
				FontColor: textColor,
			})
			textBox := gridPainter.MeasureText(text)
			if textBox.Width() > box.Width()-2 {
				continue
			}
			x := box.Left + (box.Width()-textBox.Width())/2
			y := box.Top + (box.Height()+textBox.Height())/2 - 2
			gridPainter.Text(text, x, y)
		}
	}
}

// Render renders the heat map chart.
func (h *heatMapChart) Render() (Box, error) {
	opt := h.opt
	p := h.p
	theme := h.theme()

	rowCount, columnCount, err := h.dimensions()
	if err != nil {
		return BoxZero, err
	}
	// Empty matrix: render background only, it is a deterministic result.
	if rowCount == 0 || columnCount == 0 {
		p.SetBackground(p.Width(), p.Height(), theme.GetBackgroundColor())
		return p.box, nil
	}

	// Title.
	contentTop := 0
	if opt.Title.Text != "" || opt.Title.Subtext != "" {
		titleOpt := opt.Title
		if titleOpt.Theme == nil {
			titleOpt.Theme = theme
		}
		titleBox, titleErr := NewTitlePainter(p, titleOpt).Render()
		if titleErr != nil {
			return BoxZero, titleErr
		}
		contentTop = titleBox.Height() + 10
	}

	contentPainter := p.Child(PainterPaddingOption(Box{
		Top: contentTop,
	}))

	min, max, ok := h.finiteMinMax()
	var scale *heatMapScale
	if ok {
		scale = newHeatMapScale(min, max, opt.Min, opt.Max, opt.Colors)
	} else {
		// All values are missing, the scale is only used for ticks which
		// are never drawn; use a zero range to keep it deterministic.
		scale = newHeatMapScale(0, 0, opt.Min, opt.Max, opt.Colors)
	}

	// Legend occupies the bottom (or right) of the content area.
	legendWidth, legendHeight := h.legendMetrics(contentPainter, true)
	gridAreaPainter := contentPainter
	legendPadding := Box{}
	if legendHeight > 0 || legendWidth > 0 {
		if opt.Legend.Orient == OrientVertical {
			legendPadding = Box{
				Right: legendWidth + 10,
			}
		} else {
			legendPadding = Box{
				Bottom: legendHeight + 10,
			}
		}
		gridAreaPainter = contentPainter.Child(PainterPaddingOption(legendPadding))
	}

	layout := h.computeLayout(gridAreaPainter, rowCount, columnCount)

	// Draw axis labels on the grid area painter.
	h.renderAxisLabels(gridAreaPainter, layout)

	// Draw cells on the grid painter.
	gridPainter := gridAreaPainter.Child(PainterPaddingOption(Box{
		Left: layout.gridLeft,
		Top:  layout.gridTop,
	}))
	gridPainter.SetBackground(layout.gridWidth, layout.gridHeight, theme.GetBackgroundColor(), true)
	h.renderCells(gridPainter, layout, scale)

	// Draw legend.
	if legendHeight > 0 || legendWidth > 0 {
		if opt.Legend.Orient == OrientVertical {
			legendPainter := contentPainter.Child(PainterPaddingOption(Box{
				Left: contentPainter.Width() - legendWidth,
			}))
			h.renderLegend(legendPainter, legendWidth, contentPainter.Height())
		} else {
			legendPainter := contentPainter.Child(PainterPaddingOption(Box{
				Top: contentPainter.Height() - legendHeight,
			}))
			h.renderLegend(legendPainter, gridAreaPainter.Width(), legendHeight)
		}
	}

	return p.box, nil
}

func newHeatMapPainter(opt *HeatMapOption) (*Painter, error) {
	outputType := opt.Type
	if outputType == "" {
		outputType = ChartOutputPNG
	}
	width := opt.Width
	if width <= 0 {
		width = defaultChartWidth
	}
	height := opt.Height
	if height <= 0 {
		height = defaultChartHeight
	}
	font := opt.Font
	if font == nil && opt.FontFamily != "" {
		font, _ = GetFont(opt.FontFamily)
	}
	if font == nil {
		font, _ = GetDefaultFont()
	}
	p, err := NewPainter(PainterOptions{
		Type:   outputType,
		Width:  width,
		Height: height,
		Font:   font,
	}, PainterThemeOption(NewTheme(opt.Theme)))
	if err != nil {
		return nil, err
	}
	if !opt.BackgroundColor.IsZero() {
		p.SetBackground(width, height, opt.BackgroundColor)
	} else {
		p.SetBackground(width, height, p.theme.GetBackgroundColor())
	}
	return p, nil
}

// HeatMapOptionRender renders a heat map chart with the full option.
func HeatMapOptionRender(opt HeatMapOption) (*Painter, error) {
	p, err := newHeatMapPainter(&opt)
	if err != nil {
		return nil, err
	}
	renderPainter := p
	if !opt.Padding.IsZero() {
		renderPainter = p.Child(PainterPaddingOption(opt.Padding))
	}
	_, err = NewHeatMapChart(renderPainter, opt).Render()
	if err != nil {
		return nil, err
	}
	return p, nil
}

// HeatMapRender renders a heat map chart. The data is a value matrix
// (rows are outer slice), rows and columns are the axis labels.
func HeatMapRender(data [][]float64, rows, columns []string, opts ...OptionFunc) (*Painter, error) {
	opt := ChartOption{}
	for _, fn := range opts {
		fn(&opt)
	}
	heatMapOpt := opt.HeatMap
	heatMapOpt.Type = opt.Type
	heatMapOpt.Theme = opt.Theme
	heatMapOpt.FontFamily = opt.FontFamily
	heatMapOpt.Width = opt.Width
	heatMapOpt.Height = opt.Height
	if opt.Padding != (Box{}) {
		heatMapOpt.Padding = opt.Padding
	}
	heatMapOpt.Title = opt.Title
	heatMapOpt.Rows = rows
	heatMapOpt.Columns = columns
	heatMapOpt.Data = data
	if !opt.BackgroundColor.IsZero() {
		heatMapOpt.BackgroundColor = opt.BackgroundColor
	}
	if opt.ValueFormatter != nil {
		heatMapOpt.ValueFormatter = opt.ValueFormatter
	}
	if heatMapOpt.Type == "" {
		heatMapOpt.Type = ChartOutputPNG
	}
	return HeatMapOptionRender(heatMapOpt)
}

// HeatMapColorsOptionFunc sets the color scale of heat map.
func HeatMapColorsOptionFunc(colors ...string) OptionFunc {
	return func(opt *ChartOption) {
		opt.HeatMap.Colors = colors
	}
}

// HeatMapMinMaxOptionFunc sets the min and max value of heat map color scale.
func HeatMapMinMaxOptionFunc(min, max float64) OptionFunc {
	return func(opt *ChartOption) {
		opt.HeatMap.Min = NewFloatPoint(min)
		opt.HeatMap.Max = NewFloatPoint(max)
	}
}

// HeatMapLegendOptionFunc sets the legend option of heat map.
func HeatMapLegendOptionFunc(legend HeatMapLegendOption) OptionFunc {
	return func(opt *ChartOption) {
		opt.HeatMap.Legend = legend
	}
}

// HeatMapNullStyleOptionFunc sets the missing value cell style of heat map.
func HeatMapNullStyleOptionFunc(color Color, text ...string) OptionFunc {
	return func(opt *ChartOption) {
		opt.HeatMap.NullColor = color
		if len(text) != 0 {
			opt.HeatMap.NullText = text[0]
		}
	}
}

// HeatMapLabelShowOptionFunc sets the show flag of heat map cell label.
func HeatMapLabelShowOptionFunc(show bool) OptionFunc {
	return func(opt *ChartOption) {
		opt.HeatMap.LabelShow = &show
	}
}

// HeatMapLabelRotationOptionFunc sets the column label rotation (radians)
// of heat map.
func HeatMapLabelRotationOptionFunc(radians float64) OptionFunc {
	return func(opt *ChartOption) {
		opt.HeatMap.LabelRotation = radians
	}
}
