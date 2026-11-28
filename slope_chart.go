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
	"sort"

	"github.com/dustin/go-humanize"
	"github.com/golang/freetype/truetype"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

const (
	// SlopeSortLeft sorts the rows by the value of the left time point
	SlopeSortLeft = "left"
	// SlopeSortRight sorts the rows by the value of the right time point
	SlopeSortRight = "right"
	// SlopeOrderAsc sorts the rows from small to large
	SlopeOrderAsc = "asc"
	// SlopeOrderDesc sorts the rows from large to small
	SlopeOrderDesc = "desc"
)

const (
	defaultSlopeStrokeWidth = 2.0
	defaultSlopeSymbolSize  = 8.0
	defaultSlopeLabelGap    = 8
	defaultSlopeMissingText = "-"
)

// SlopeChartData is one named row of a slope chart.
// Left and Right are the values measured at the two time points.
type SlopeChartData struct {
	// The name of the data row, it is shown in the label columns on both sides
	Name string
	// The value of the left time point
	Left float64
	// The value of the right time point
	Right float64
	// The style of the row, only StrokeColor and FillColor are supported
	Style Style
}

// SlopeChartOption is the option of a slope chart
type SlopeChartOption struct {
	// The output type of chart, "svg" or "png", default value is "svg"
	Type string
	// The width of chart, default width is 600
	Width int
	// The height of chart, default height is 400
	Height int
	// The theme of chart, "light" and "dark".
	// The default theme is "light"
	Theme string
	// The font family, which should be installed first
	FontFamily string
	// The padding for chart, default padding is [20, 10, 10, 10]
	Padding Box
	// The title option
	Title TitleOption
	// The legend option
	Legend LegendOption
	// The data rows of the slope chart
	Data []SlopeChartData
	// The label of the left time point, drawn under the left value column
	LeftLabel string
	// The label of the right time point, drawn under the right value column
	RightLabel string
	// The minimun value of the shared value axis
	Min *float64
	// The maximum value of the shared value axis
	Max *float64
	// Number of segments that the shared value axis is split into
	DivideCount int
	// Show the name/value labels of every row on both sides
	LabelShow bool
	// Sort the rows stably, it can be SlopeSortLeft or SlopeSortRight.
	// Empty value means keeping the input order
	SortBy string
	// The sort order, it can be SlopeOrderAsc or SlopeOrderDesc.
	// The default order is SlopeOrderDesc
	SortOrder string
	// The colors of rows, the zero value color is ignored and the theme color is used
	Colors []Color
	// The stroke width of the connecting line, default value is 2
	StrokeWidth float64
	// The diameter of the endpoint symbol, default value is 8
	SymbolSize float64
	// The gap between the label column and the value column, default value is 8
	LabelGap int
	// The text rendered in the label when one time point is missing, default value is "-"
	MissingText string
	// The value formatter of labels and the value axis
	ValueFormatter ValueFormatter

	theme ColorPalette
	font  *truetype.Font
}

// SlopeOptionFunc option function of slope chart
type SlopeOptionFunc func(opt *SlopeChartOption)

// SlopeTypeOptionFunc set type of slope chart's output
func SlopeTypeOptionFunc(t string) SlopeOptionFunc {
	return func(opt *SlopeChartOption) {
		opt.Type = t
	}
}

// SlopeWidthOptionFunc set width of slope chart
func SlopeWidthOptionFunc(width int) SlopeOptionFunc {
	return func(opt *SlopeChartOption) {
		opt.Width = width
	}
}

// SlopeHeightOptionFunc set height of slope chart
func SlopeHeightOptionFunc(height int) SlopeOptionFunc {
	return func(opt *SlopeChartOption) {
		opt.Height = height
	}
}

// SlopeThemeOptionFunc set theme of slope chart
func SlopeThemeOptionFunc(theme string) SlopeOptionFunc {
	return func(opt *SlopeChartOption) {
		opt.Theme = theme
	}
}

// SlopePaddingOptionFunc set padding of slope chart
func SlopePaddingOptionFunc(padding Box) SlopeOptionFunc {
	return func(opt *SlopeChartOption) {
		opt.Padding = padding
	}
}

// SlopeTitleOptionFunc set title of slope chart
func SlopeTitleOptionFunc(title TitleOption) SlopeOptionFunc {
	return func(opt *SlopeChartOption) {
		opt.Title = title
	}
}

// SlopeLegendOptionFunc set legend of slope chart
func SlopeLegendOptionFunc(legend LegendOption) SlopeOptionFunc {
	return func(opt *SlopeChartOption) {
		opt.Legend = legend
	}
}

// SlopeTimeLabelOptionFunc set the labels of the two time points
func SlopeTimeLabelOptionFunc(left, right string) SlopeOptionFunc {
	return func(opt *SlopeChartOption) {
		opt.LeftLabel = left
		opt.RightLabel = right
	}
}

// SlopeMinMaxOptionFunc set the min and max value of the shared value axis
func SlopeMinMaxOptionFunc(min, max float64) SlopeOptionFunc {
	return func(opt *SlopeChartOption) {
		opt.Min = NewFloatPoint(min)
		opt.Max = NewFloatPoint(max)
	}
}

// SlopeLabelShowOptionFunc set the show flag of the data labels
func SlopeLabelShowOptionFunc(show bool) SlopeOptionFunc {
	return func(opt *SlopeChartOption) {
		opt.LabelShow = show
	}
}

// SlopeSortOptionFunc set the stable sort option
func SlopeSortOptionFunc(sortBy, sortOrder string) SlopeOptionFunc {
	return func(opt *SlopeChartOption) {
		opt.SortBy = sortBy
		opt.SortOrder = sortOrder
	}
}

// SlopeColorsOptionFunc set the colors of rows
func SlopeColorsOptionFunc(colors ...Color) SlopeOptionFunc {
	return func(opt *SlopeChartOption) {
		opt.Colors = colors
	}
}

// SlopeStrokeWidthOptionFunc set the stroke width of connecting line
func SlopeStrokeWidthOptionFunc(strokeWidth float64) SlopeOptionFunc {
	return func(opt *SlopeChartOption) {
		opt.StrokeWidth = strokeWidth
	}
}

// SlopeValueFormatterOptionFunc set the value formatter
func SlopeValueFormatterOptionFunc(formatter ValueFormatter) SlopeOptionFunc {
	return func(opt *SlopeChartOption) {
		opt.ValueFormatter = formatter
	}
}

func (o *SlopeChartOption) fillDefault() error {
	o.theme = NewTheme(o.Theme)
	o.Width = getDefaultInt(o.Width, defaultChartWidth)
	o.Height = getDefaultInt(o.Height, defaultChartHeight)
	if o.Type == "" {
		o.Type = ChartOutputSVG
	}
	if o.Padding.IsZero() {
		o.Padding = Box{
			Top:    20,
			Right:  20,
			Bottom: 20,
			Left:   20,
		}
	}
	if o.DivideCount <= 0 {
		o.DivideCount = defaultAxisDivideCount
	}
	if o.StrokeWidth <= 0 {
		o.StrokeWidth = defaultSlopeStrokeWidth
	}
	if o.SymbolSize <= 0 {
		o.SymbolSize = defaultSlopeSymbolSize
	}
	if o.LabelGap <= 0 {
		o.LabelGap = defaultSlopeLabelGap
	}
	if o.MissingText == "" {
		o.MissingText = defaultSlopeMissingText
	}
	if o.SortOrder == "" {
		o.SortOrder = SlopeOrderDesc
	}
	if o.SortBy != "" &&
		o.SortBy != SlopeSortLeft &&
		o.SortBy != SlopeSortRight {
		return errors.New("SortBy should be SlopeSortLeft or SlopeSortRight")
	}
	if o.SortOrder != SlopeOrderAsc && o.SortOrder != SlopeOrderDesc {
		return errors.New("SortOrder should be SlopeOrderAsc or SlopeOrderDesc")
	}
	var err error
	o.font, err = GetFont(o.FontFamily)
	if err != nil || o.font == nil {
		o.font, _ = GetDefaultFont()
	} else {
		o.theme.SetFont(o.font)
	}
	if len(o.Data) == 0 {
		return errors.New("The data of slope chart can not be empty")
	}
	for _, item := range o.Data {
		if !isFiniteNumber(item.Left) && !isFiniteNumber(item.Right) {
			return errors.New("Both left and right value are missing for " + item.Name)
		}
	}
	return nil
}

// isFiniteNumber reports whether v is a plottable finite number.
// NaN, Infinity and the library null value are treated as missing data.
func isFiniteNumber(v float64) bool {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return false
	}
	return v != nullValue
}

type slopeChart struct {
	p   *Painter
	opt *SlopeChartOption
}

// NewSlopeChart returns a slope chart renderer
func NewSlopeChart(p *Painter, opt SlopeChartOption) *slopeChart {
	if opt.theme == nil {
		opt.theme = NewTheme(opt.Theme)
	}
	return &slopeChart{
		p:   p,
		opt: &opt,
	}
}

// slopeRow is one internally ordered data row.
// index is the original index of the data, which keeps following the entity's color.
type slopeRow struct {
	index int
	data  SlopeChartData
}

type slopeEndpoint struct {
	// The row index of the data
	Row int
	// The x pixel of the endpoint, it is always xLeft or xRight
	X int
	// The y pixel converted from the real value
	Y int
	// The real value of the endpoint
	Value float64
	// The endpoint is missing, no symbol or connecting line is drawn
	Missing bool
}

type slopeLabelItem struct {
	// The row index of the data
	Row int
	// The y pixel of the real endpoint
	anchorY int
	text    string
	textW   int
	textH   int
	// The final top position after collision avoidance
	top int
}

type slopeLayout struct {
	plotPainter *Painter
	plotBox     Box
	r           axisRange
	xLeft       int
	xRight      int
	height      int
	rows        []slopeRow
	leftPoints  []slopeEndpoint
	rightPoints []slopeEndpoint
	leftLabels  []*slopeLabelItem
	rightLabels []*slopeLabelItem
}

func (c *slopeChart) sortedRows() []slopeRow {
	rows := make([]slopeRow, len(c.opt.Data))
	for index, item := range c.opt.Data {
		rows[index] = slopeRow{
			index: index,
			data:  item,
		}
	}
	if c.opt.SortBy == "" {
		return rows
	}
	less := func(i, j int) bool {
		var vi, vj float64
		if c.opt.SortBy == SlopeSortRight {
			vi, vj = rows[i].data.Right, rows[j].data.Right
		} else {
			vi, vj = rows[i].data.Left, rows[j].data.Left
		}
		// Missing values are always ranked after finite values
		iMissing, jMissing := !isFiniteNumber(vi), !isFiniteNumber(vj)
		if iMissing || jMissing {
			if iMissing == jMissing {
				return false
			}
			return jMissing
		}
		if vi == vj {
			return false
		}
		if c.opt.SortOrder == SlopeOrderAsc {
			return vi < vj
		}
		return vi > vj
	}
	// SliceStable keeps the input order of equal elements, including equal values
	sort.SliceStable(rows, less)
	return rows
}

func (c *slopeChart) valueText(v float64) string {
	if c.opt.ValueFormatter != nil {
		return c.opt.ValueFormatter(v)
	}
	return humanize.FtoaWithDigits(v, 2)
}

// layout calculates every pixel position. It never changes the data coordinates:
// label collision avoidance only adjusts the label top position, the endpoint
// symbols always stay at the value converted positions.
func (c *slopeChart) layout(content *Painter) (*slopeLayout, error) {
	opt := c.opt
	theme := opt.theme
	rows := c.sortedRows()

	// The min and max only count finite values
	min := math.MaxFloat64
	max := -math.MaxFloat64
	for _, row := range rows {
		if isFiniteNumber(row.data.Left) {
			min = math.Min(min, row.data.Left)
			max = math.Max(max, row.data.Left)
		}
		if isFiniteNumber(row.data.Right) {
			min = math.Min(min, row.data.Right)
			max = math.Max(max, row.data.Right)
		}
	}

	rangeHeight := content.Height() - defaultXAxisHeight
	r := NewRange(AxisRangeOption{
		Painter:     content,
		Min:         min,
		Max:         max,
		Size:        rangeHeight,
		DivideCount: opt.DivideCount,
	})
	if opt.Min != nil && *opt.Min <= min {
		r.min = *opt.Min
	}
	if opt.Max != nil && *opt.Max >= max {
		r.max = *opt.Max
	}
	// A degenerate range (all values are equal, e.g. all zero) would collapse
	// every endpoint to the same pixel, so pad it into a plottable range
	if r.max <= r.min {
		span := math.Abs(r.min)
		if span == 0 {
			span = 1
		}
		r.min -= span
		r.max += span
	}

	yAxisOption := YAxisOption{
		Theme:       theme,
		Font:        opt.font,
		Data:        r.Values(),
		DivideCount: opt.DivideCount,
	}
	// The left axis renders the split lines, the right axis shares the same range
	leftAxis := NewLeftYAxis(content, yAxisOption)
	leftBox, err := leftAxis.Render()
	if err != nil {
		return nil, err
	}
	leftWidth := leftBox.Width()
	rightAxis := NewRightYAxis(content.Child(PainterPaddingOption(Box{
		Left: leftWidth,
	})), yAxisOption)
	rightBox, err := rightAxis.Render()
	if err != nil {
		return nil, err
	}
	rightWidth := rightBox.Width()

	plot := content.Child(PainterPaddingOption(Box{
		Left:   leftWidth,
		Right:  rightWidth,
		Bottom: defaultXAxisHeight,
	}))
	width := plot.Width()
	height := plot.Height()
	if width <= 0 || height <= 0 {
		return nil, errors.New("The size of slope chart is too small")
	}

	plot.SetTextStyle(Style{
		Font:      opt.font,
		FontSize:  labelFontSize,
		FontColor: theme.GetTextColor(),
	})

	yFor := func(v float64) int {
		return height - r.getHeight(v)
	}

	leftPoints := make([]slopeEndpoint, 0, len(rows))
	rightPoints := make([]slopeEndpoint, 0, len(rows))
	leftLabels := make([]*slopeLabelItem, 0, len(rows))
	rightLabels := make([]*slopeLabelItem, 0, len(rows))
	maxLeftLabelWidth := 0
	maxRightLabelWidth := 0

	for order, row := range rows {
		leftMissing := !isFiniteNumber(row.data.Left)
		rightMissing := !isFiniteNumber(row.data.Right)

		var leftPoint, rightPoint slopeEndpoint
		if leftMissing {
			// The missing endpoint has no data coordinate, it borrows the y of
			// the existing opposite endpoint only for label placement
			leftPoint = slopeEndpoint{
				Row:     order,
				X:       0,
				Y:       yFor(row.data.Right),
				Value:   row.data.Left,
				Missing: true,
			}
		} else {
			leftPoint = slopeEndpoint{
				Row:   order,
				X:     0,
				Y:     yFor(row.data.Left),
				Value: row.data.Left,
			}
		}
		if rightMissing {
			rightPoint = slopeEndpoint{
				Row:     order,
				X:       0,
				Y:       yFor(row.data.Left),
				Value:   row.data.Right,
				Missing: true,
			}
		} else {
			rightPoint = slopeEndpoint{
				Row:   order,
				X:     0,
				Y:     yFor(row.data.Right),
				Value: row.data.Right,
			}
		}
		leftPoints = append(leftPoints, leftPoint)
		rightPoints = append(rightPoints, rightPoint)

		if opt.LabelShow {
			leftValueText := opt.MissingText
			if !leftMissing {
				leftValueText = c.valueText(row.data.Left)
			}
			rightValueText := opt.MissingText
			if !rightMissing {
				rightValueText = c.valueText(row.data.Right)
			}
			leftText := row.data.Name + " " + leftValueText
			rightText := row.data.Name + " " + rightValueText
			leftBox := plot.MeasureText(leftText)
			rightBox := plot.MeasureText(rightText)
			if leftBox.Width() > maxLeftLabelWidth {
				maxLeftLabelWidth = leftBox.Width()
			}
			if rightBox.Width() > maxRightLabelWidth {
				maxRightLabelWidth = rightBox.Width()
			}
			leftLabels = append(leftLabels, &slopeLabelItem{
				Row:     order,
				anchorY: leftPoint.Y,
				text:    leftText,
				textW:   leftBox.Width(),
				textH:   leftBox.Height(),
			})
			rightLabels = append(rightLabels, &slopeLabelItem{
				Row:     order,
				anchorY: rightPoint.Y,
				text:    rightText,
				textW:   rightBox.Width(),
				textH:   rightBox.Height(),
			})
		}
	}

	gap := opt.LabelGap
	xLeft := maxLeftLabelWidth + 2*gap
	xRight := width - maxRightLabelWidth - 2*gap
	if xRight <= xLeft {
		return nil, errors.New("The width of slope chart is too small for the labels")
	}
	for index := range leftPoints {
		leftPoints[index].X = xLeft
		rightPoints[index].X = xRight
	}

	// Adjust the labels on each side independently, the symbols are not moved
	if opt.LabelShow {
		adjustSlopeLabels(leftLabels, 0, height, gap)
		adjustSlopeLabels(rightLabels, 0, height, gap)
	}

	return &slopeLayout{
		plotPainter: plot,
		plotBox:     plot.box,
		r:           r,
		xLeft:       xLeft,
		xRight:      xRight,
		height:      height,
		rows:        rows,
		leftPoints:  leftPoints,
		rightPoints: rightPoints,
		leftLabels:  leftLabels,
		rightLabels: rightLabels,
	}, nil
}

// adjustSlopeLabels resolves label overlaps with a stable top-to-bottom push.
// It only modifies the label top position, the anchor y of the real endpoint
// is kept untouched so the data coordinates never change.
func adjustSlopeLabels(labels []*slopeLabelItem, minY, maxY, gap int) {
	if len(labels) == 0 {
		return
	}
	sort.SliceStable(labels, func(i, j int) bool {
		return labels[i].anchorY < labels[j].anchorY
	})
	for _, item := range labels {
		item.top = item.anchorY - item.textH/2
	}
	// Push the lower label down when it overlaps the previous one
	for i := 1; i < len(labels); i++ {
		prev := labels[i-1]
		lowestTop := prev.top + prev.textH + gap
		if labels[i].top < lowestTop {
			labels[i].top = lowestTop
		}
	}
	last := labels[len(labels)-1]
	if overflow := last.top + last.textH - maxY; overflow > 0 {
		for _, item := range labels {
			item.top -= overflow
		}
	}
	// The whole stack may be moved above the top boundary, push it back and
	// resolve the overlaps again
	if labels[0].top < minY {
		overflow := minY - labels[0].top
		for _, item := range labels {
			item.top += overflow
		}
		for i := 1; i < len(labels); i++ {
			prev := labels[i-1]
			lowestTop := prev.top + prev.textH + gap
			if labels[i].top < lowestTop {
				labels[i].top = lowestTop
			}
		}
	}
	for _, item := range labels {
		if item.top < minY {
			item.top = minY
		}
		if item.top+item.textH > maxY {
			item.top = maxY - item.textH
		}
	}
}

func (c *slopeChart) rowColor(row slopeRow) Color {
	color := c.opt.theme.GetSeriesColor(row.index)
	if row.index < len(c.opt.Colors) && !c.opt.Colors[row.index].IsZero() {
		color = c.opt.Colors[row.index]
	}
	if !row.data.Style.StrokeColor.IsZero() {
		color = row.data.Style.StrokeColor
	}
	return color
}

func (c *slopeChart) drawLabels(layout *slopeLayout, labels []*slopeLabelItem, points []slopeEndpoint, isLeft bool) {
	opt := c.opt
	theme := opt.theme
	p := layout.plotPainter
	p.SetTextStyle(Style{
		Font:      opt.font,
		FontSize:  labelFontSize,
		FontColor: theme.GetTextColor(),
	})
	columnX := layout.xRight
	if isLeft {
		columnX = layout.xLeft
	}
	for _, item := range labels {
		point := points[item.Row]
		centerY := item.top + item.textH/2
		// A short leader line is drawn only when the label was moved away from
		// its endpoint, the missing side has no endpoint so it has no leader
		if !point.Missing && absInt(centerY-point.Y) > item.textH/2 {
			p.OverrideDrawingStyle(Style{
				StrokeColor: theme.GetAxisSplitLineColor(),
				StrokeWidth: 1,
			})
			if isLeft {
				p.LineStroke([]Point{
					{X: columnX - opt.LabelGap, Y: centerY},
					{X: columnX, Y: point.Y},
				})
			} else {
				p.LineStroke([]Point{
					{X: columnX + opt.LabelGap, Y: centerY},
					{X: columnX, Y: point.Y},
				})
			}
		}
		x := columnX + opt.LabelGap
		if isLeft {
			// Right aligned, the label ends before the left column
			x = columnX - opt.LabelGap - item.textW
		}
		p.Text(item.text, x, item.top+item.textH)
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// Render renders the slope chart
func (c *slopeChart) Render() (Box, error) {
	opt := c.opt
	if opt.theme == nil {
		err := opt.fillDefault()
		if err != nil {
			return BoxZero, err
		}
	}
	p := c.p
	theme := opt.theme
	p.SetBackground(p.Width(), p.Height(), theme.GetBackgroundColor())

	content := p
	if !opt.Padding.IsZero() {
		content = p.Child(PainterPaddingOption(opt.Padding))
	}
	if opt.ValueFormatter != nil {
		content.valueFormatter = opt.ValueFormatter
	}

	legendHeight := 0
	if !opt.Legend.IsEmpty() && !isFalse(opt.Legend.Show) {
		if opt.Legend.Theme == nil {
			opt.Legend.Theme = theme
		}
		legendResult, err := NewLegendPainter(content, opt.Legend).Render()
		if err != nil {
			return BoxZero, err
		}
		legendHeight = legendResult.Height()
	}
	if opt.Title.Text != "" {
		if opt.Title.Theme == nil {
			opt.Title.Theme = theme
		}
		titleBox, err := NewTitlePainter(content, opt.Title).Render()
		if err != nil {
			return BoxZero, err
		}
		top := maxInt(legendHeight, titleBox.Height())
		if opt.Legend.Orient == OrientVertical {
			top = titleBox.Height()
		}
		content = content.Child(PainterPaddingOption(Box{
			Top: top + 20,
		}))
	}

	layout, err := c.layout(content)
	if err != nil {
		return BoxZero, err
	}
	plot := layout.plotPainter

	// Draw the connecting lines first, a line exists only when both endpoints
	// have finite values so it always connects two real data positions
	for order, row := range layout.rows {
		leftPoint := layout.leftPoints[order]
		rightPoint := layout.rightPoints[order]
		if leftPoint.Missing || rightPoint.Missing {
			continue
		}
		color := c.rowColor(row)
		plot.OverrideDrawingStyle(Style{
			StrokeColor: color,
			StrokeWidth: opt.StrokeWidth,
		}).LineStroke([]Point{
			{X: leftPoint.X, Y: leftPoint.Y},
			{X: rightPoint.X, Y: rightPoint.Y},
		})
	}

	// Draw the endpoint symbols at the exact value converted positions
	symbolRadius := opt.SymbolSize / 2
	for order, row := range layout.rows {
		color := c.rowColor(row)
		fillColor := drawing.ColorWhite
		if theme.IsDark() {
			fillColor = color
		}
		if !row.data.Style.FillColor.IsZero() {
			fillColor = row.data.Style.FillColor
		}
		plot.OverrideDrawingStyle(Style{
			StrokeColor: color,
			StrokeWidth: 1,
			FillColor:   fillColor,
		})
		drawSymbol := func(point slopeEndpoint) {
			if point.Missing {
				return
			}
			plot.Circle(symbolRadius, point.X, point.Y)
			plot.FillStroke()
		}
		drawSymbol(layout.leftPoints[order])
		drawSymbol(layout.rightPoints[order])
	}

	if opt.LabelShow {
		c.drawLabels(layout, layout.leftLabels, layout.leftPoints, true)
		c.drawLabels(layout, layout.rightLabels, layout.rightPoints, false)
	}

	// Draw the time point labels under the two columns
	if opt.LeftLabel != "" || opt.RightLabel != "" {
		plot.SetTextStyle(Style{
			Font:      opt.font,
			FontSize:  labelFontSize,
			FontColor: theme.GetTextColor(),
		})
		bottomY := layout.height + defaultXAxisHeight - 8
		if opt.LeftLabel != "" {
			box := plot.MeasureText(opt.LeftLabel)
			plot.Text(opt.LeftLabel, layout.xLeft-box.Width()/2, bottomY)
		}
		if opt.RightLabel != "" {
			box := plot.MeasureText(opt.RightLabel)
			plot.Text(opt.RightLabel, layout.xRight-box.Width()/2, bottomY)
		}
	}

	return p.box, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// SlopeRender renders a slope chart with named paired values.
// The left and right time points share one value axis.
func SlopeRender(data []SlopeChartData, opts ...SlopeOptionFunc) (*Painter, error) {
	opt := SlopeChartOption{
		Data: data,
	}
	for _, fn := range opts {
		fn(&opt)
	}
	err := opt.fillDefault()
	if err != nil {
		return nil, err
	}
	p, err := NewPainter(PainterOptions{
		Type:   opt.Type,
		Width:  opt.Width,
		Height: opt.Height,
		Font:   opt.font,
	})
	if err != nil {
		return nil, err
	}
	_, err = NewSlopeChart(p, opt).Render()
	if err != nil {
		return nil, err
	}
	return p, nil
}
