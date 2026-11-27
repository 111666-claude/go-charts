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
)

// MosaicCell describes a single tile of the mosaic chart,
// which is identified by its column index and row index.
type MosaicCell struct {
	// The index of column
	Column int
	// The index of row
	Row int
	// The rect of the tile in the plot area
	Box Box
	// The frequency value of the tile
	Value float64
}

// MosaicLayout is the deterministic geometry result of a mosaic chart.
// Every non-zero tile covers the plot area exactly once,
// so the tiles neither overlap nor leave any gap.
type MosaicLayout struct {
	// The width of plot area
	Width int
	// The height of plot area
	Height int
	// The x coordinate list of column boundary,
	// the first value is 0 and the last value is Width
	ColumnX []int
	// The y coordinate list of row boundary of each column,
	// the first value is 0 and the last value is Height
	RowY [][]int
	// All non-zero tiles
	Cells []MosaicCell
}

type mosaicChart struct {
	p   *Painter
	opt *MosaicChartOption
}

// MosaicChartOption is the option of mosaic chart.
// Data is a non-negative row x column frequency table:
// Data[row][column], each column's width follows its column total,
// and the height of a cell follows its proportion inside the column.
type MosaicChartOption struct {
	// The output type of chart, "svg" or "png", default value is "png"
	Type string
	// The width of chart, default width is 600
	Width int
	// The height of chart, default height is 400
	Height int
	// The theme
	Theme ColorPalette
	// The theme name of chart
	ThemeName string
	// The font family, which should be installed first
	FontFamily string
	// The font
	Font *truetype.Font
	// The non-negative frequency table, Data[row][column]
	Data [][]float64
	// The labels of columns (the channels or sources of data)
	ColumnLabels []string
	// The labels of rows (the categories shown in legend)
	RowLabels []string
	// The padding of chart
	Padding Box
	// The option of title
	Title TitleOption
	// The legend option
	Legend LegendOption
	// Show the frequency value inside each non-zero tile
	ShowValue bool
	// The formatter of the tile value, it receives the row index,
	// column index and the original frequency value
	ValueFormatter func(row, column int, value float64) string
	// background is filled
	backgroundIsFilled bool
}

// NewMosaicChart returns a mosaic chart renderer
func NewMosaicChart(p *Painter, opt MosaicChartOption) *mosaicChart {
	if opt.Theme == nil {
		opt.Theme = defaultTheme
	}
	return &mosaicChart{
		p:   p,
		opt: &opt,
	}
}

// largestRemainderDivide divides total into count integer parts,
// each part is proportional to its weight. The largest remainder
// method is used so that the sum of parts always equals total,
// which guarantees that tiles fill the plot area exactly.
func largestRemainderDivide(total int, weights []float64) []int {
	count := len(weights)
	result := make([]int, count)
	if count == 0 || total <= 0 {
		return result
	}
	weightTotal := float64(0)
	for _, w := range weights {
		weightTotal += w
	}
	if weightTotal <= 0 {
		return result
	}
	remainder := total
	type item struct {
		index     int
		remainder float64
	}
	items := make([]item, count)
	for index, w := range weights {
		exact := float64(total) * w / weightTotal
		base := int(exact)
		result[index] = base
		remainder -= base
		items[index] = item{
			index:     index,
			remainder: exact - float64(base),
		}
	}
	// The larger remainder gets one more pixel,
	// the original order breaks a tie deterministically.
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].remainder > items[j].remainder
	})
	for i := 0; i < remainder; i++ {
		result[items[i%count].index]++
	}
	return result
}

// validateMosaicData validates the frequency table and returns
// the column totals and the grand total.
func validateMosaicData(data [][]float64) (columnTotals []float64, grandTotal float64, err error) {
	rowCount := len(data)
	if rowCount == 0 {
		err = errors.New("The data of mosaic chart can not be empty")
		return
	}
	columnCount := len(data[0])
	if columnCount == 0 {
		err = errors.New("The data of mosaic chart can not be empty")
		return
	}
	columnTotals = make([]float64, columnCount)
	for _, row := range data {
		if len(row) != columnCount {
			err = errors.New("The row count of mosaic chart data should be consistent")
			return
		}
		for columnIndex, value := range row {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				err = errors.New("The value of mosaic chart should be finite")
				return
			}
			if value < 0 {
				err = errors.New("The value of mosaic chart should be non-negative")
				return
			}
			columnTotals[columnIndex] += value
			grandTotal += value
		}
	}
	if grandTotal <= 0 {
		err = errors.New("The sum value of mosaic chart should gt 0")
	}
	return
}

// NewMosaicLayout calculates the deterministic geometry of the mosaic.
// The width of each column follows its column total, and the height of
// each non-zero cell follows its proportion inside the column.
func NewMosaicLayout(data [][]float64, width, height int) (*MosaicLayout, error) {
	columnTotals, _, err := validateMosaicData(data)
	if err != nil {
		return nil, err
	}
	rowCount := len(data)
	columnCount := len(data[0])

	layout := &MosaicLayout{
		Width:   width,
		Height:  height,
		ColumnX: make([]int, columnCount+1),
		RowY:    make([][]int, columnCount),
	}
	// The width of column follows the column total
	columnWidths := largestRemainderDivide(width, columnTotals)
	x := 0
	for index, w := range columnWidths {
		layout.ColumnX[index] = x
		x += w
	}
	layout.ColumnX[columnCount] = width

	for columnIndex := 0; columnIndex < columnCount; columnIndex++ {
		rowY := make([]int, rowCount+1)
		columnTotal := columnTotals[columnIndex]
		// A zero column has no tile and all rows start from 0
		if columnTotal <= 0 {
			layout.RowY[columnIndex] = rowY
			continue
		}
		rowHeights := make([]float64, rowCount)
		for rowIndex := 0; rowIndex < rowCount; rowIndex++ {
			rowHeights[rowIndex] = data[rowIndex][columnIndex]
		}
		// The height of tile follows its proportion in the column
		heights := largestRemainderDivide(height, rowHeights)
		y := 0
		for rowIndex, h := range heights {
			rowY[rowIndex] = y
			y += h
		}
		rowY[rowCount] = height
		layout.RowY[columnIndex] = rowY

		for rowIndex := 0; rowIndex < rowCount; rowIndex++ {
			if data[rowIndex][columnIndex] <= 0 {
				continue
			}
			layout.Cells = append(layout.Cells, MosaicCell{
				Column: columnIndex,
				Row:    rowIndex,
				Value:  data[rowIndex][columnIndex],
				Box: Box{
					Left:   layout.ColumnX[columnIndex],
					Right:  layout.ColumnX[columnIndex+1],
					Top:    rowY[rowIndex],
					Bottom: rowY[rowIndex+1],
				},
			})
		}
	}
	return layout, nil
}

func (m *mosaicChart) render(result *defaultRenderResult) (Box, error) {
	opt := m.opt
	theme := opt.Theme
	seriesPainter := result.seriesPainter
	width := seriesPainter.Width()
	height := seriesPainter.Height()
	// The bottom area is reserved for column labels
	xAxisHeight := defaultXAxisHeight
	plotHeight := height - xAxisHeight
	// The left area is reserved for the percentage axis
	const yAxisLabelWidth = 30
	axisLabelGap := 5
	plotWidth := width - yAxisLabelWidth - axisLabelGap
	if plotWidth <= 0 || plotHeight <= 0 {
		return m.p.box, nil
	}

	data := opt.Data
	columnCount := len(data[0])
	columnLabels := opt.ColumnLabels

	layout, err := NewMosaicLayout(data, plotWidth, plotHeight)
	if err != nil {
		return BoxZero, err
	}

	plotPainter := seriesPainter.Child(PainterPaddingOption(Box{
		Left:   yAxisLabelWidth + axisLabelGap,
		Bottom: xAxisHeight,
	}))

	// Draw the tiles, fill only so adjacent tiles share the same edge
	// without any overlap or gap
	for _, cell := range layout.Cells {
		color := theme.GetSeriesColor(cell.Row)
		plotPainter.OverrideDrawingStyle(Style{
			FillColor: color,
		})
		plotPainter.MoveTo(cell.Box.Left, cell.Box.Top)
		plotPainter.LineTo(cell.Box.Right, cell.Box.Top)
		plotPainter.LineTo(cell.Box.Right, cell.Box.Bottom)
		plotPainter.LineTo(cell.Box.Left, cell.Box.Bottom)
		plotPainter.Close()
		plotPainter.Fill()

		if !opt.ShowValue {
			continue
		}
		text := ""
		if opt.ValueFormatter != nil {
			text = opt.ValueFormatter(cell.Row, cell.Column, cell.Value)
		} else {
			text = humanize.FtoaWithDigits(cell.Value, 2)
		}
		if text == "" {
			continue
		}
		// Pick a readable text color for the tile
		fontColor := Color{
			R: 255,
			G: 255,
			B: 255,
			A: 255,
		}
		if isLightColor(color) {
			fontColor = theme.GetTextColor()
		}
		plotPainter.OverrideTextStyle(Style{
			Font:      opt.Font,
			FontColor: fontColor,
			FontSize:  smallLabelFontSize,
		})
		textBox := plotPainter.MeasureText(text)
		tileWidth := cell.Box.Width()
		tileHeight := cell.Box.Height()
		// Skip the value label when the tile is too narrow to hold it
		if textBox.Width() >= tileWidth || textBox.Height() >= tileHeight {
			continue
		}
		textX := cell.Box.Left + (tileWidth-textBox.Width())>>1
		textY := cell.Box.Top + (tileHeight+textBox.Height())>>1
		plotPainter.Text(text, textX, textY)
	}

	// Draw the axis lines (left and bottom)
	seriesPainter.OverrideDrawingStyle(Style{
		StrokeColor: theme.GetAxisStrokeColor(),
		StrokeWidth: 1,
	})
	axisLeft := yAxisLabelWidth + axisLabelGap
	seriesPainter.LineStroke([]Point{
		{
			X: axisLeft,
			Y: 0,
		},
		{
			X: axisLeft,
			Y: plotHeight,
		},
		{
			X: width,
			Y: plotHeight,
		},
	})

	// Draw the y axis percentage labels.
	// The value 0 is at the bottom and 100 is at the top, and edge labels
	// are clamped inside the plot area so that they never overlap.
	seriesPainter.OverrideTextStyle(Style{
		Font:      opt.Font,
		FontColor: theme.GetTextColor(),
		FontSize:  smallLabelFontSize,
	})
	percentValues := []int{100, 75, 50, 25, 0}
	for _, value := range percentValues {
		label := humanize.Comma(int64(value)) + "%"
		textBox := seriesPainter.MeasureText(label)
		textHeight := textBox.Height()
		// The baseline is at the tick position, clamp it inside the plot
		textY := plotHeight - value*plotHeight/100 + textHeight>>1
		minY := textHeight
		maxY := plotHeight - textHeight>>1
		if textY < minY {
			textY = minY
		}
		if textY > maxY {
			textY = maxY
		}
		textX := yAxisLabelWidth - axisLabelGap - textBox.Width()
		seriesPainter.Text(label, textX, textY)
	}

	// Draw the column labels below the plot area,
	// the label is centered on its column even for a zero width column
	for columnIndex := 0; columnIndex < columnCount; columnIndex++ {
		label := ""
		if columnIndex < len(columnLabels) {
			label = columnLabels[columnIndex]
		}
		if label == "" {
			label = humanize.Comma(int64(columnIndex + 1))
		}
		textBox := seriesPainter.MeasureText(label)
		centerX := (layout.ColumnX[columnIndex] + layout.ColumnX[columnIndex+1]) >> 1
		textX := axisLeft + centerX - textBox.Width()>>1
		textY := plotHeight + axisLabelGap + textBox.Height()
		seriesPainter.Text(label, textX, textY)
	}

	return m.p.box, nil
}

// Render renders the mosaic chart
func (m *mosaicChart) Render() (Box, error) {
	opt := m.opt
	if _, _, err := validateMosaicData(opt.Data); err != nil {
		return BoxZero, err
	}
	rowCount := len(opt.Data)
	// The row labels are also used by the legend
	if len(opt.Legend.Data) == 0 {
		opt.Legend.Data = make([]string, rowCount)
		for index := 0; index < rowCount; index++ {
			if index < len(opt.RowLabels) && opt.RowLabels[index] != "" {
				opt.Legend.Data[index] = opt.RowLabels[index]
			} else {
				opt.Legend.Data[index] = humanize.Comma(int64(index + 1))
			}
		}
	}
	renderResult, err := defaultRender(m.p, defaultRenderOption{
		Theme:      opt.Theme,
		Padding:    opt.Padding,
		SeriesList: SeriesList{},
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
	return m.render(renderResult)
}

// MosaicOptionRender renders a mosaic chart with option
func MosaicOptionRender(opt MosaicChartOption) (*Painter, error) {
	if opt.Type == "" {
		opt.Type = ChartOutputPNG
	}
	if opt.Width <= 0 {
		opt.Width = defaultChartWidth
	}
	if opt.Height <= 0 {
		opt.Height = defaultChartHeight
	}
	if opt.Theme == nil {
		opt.Theme = NewTheme(opt.ThemeName)
	}
	if opt.FontFamily != "" {
		opt.Font, _ = GetFont(opt.FontFamily)
	}
	if opt.Font == nil {
		opt.Font, _ = GetDefaultFont()
	}
	// Validate the data before creating the canvas
	if _, _, err := validateMosaicData(opt.Data); err != nil {
		return nil, err
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
	p.SetBackground(p.Width(), p.Height(), opt.Theme.GetBackgroundColor())
	opt.backgroundIsFilled = true
	_, err = NewMosaicChart(p, opt).Render()
	if err != nil {
		return nil, err
	}
	return p, nil
}
