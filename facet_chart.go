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

// FacetChartOption renders multiple charts as small multiples (facets)
// arranged in a row/column grid inside one PNG or SVG.
type FacetChartOption struct {
	// The output type of chart, "svg" or "png", default value is "svg"
	Type string
	// The theme of chart, "light", "dark", "grafana" and "ant".
	// The default theme is "light"
	Theme string
	// The font family, which should be installed first
	FontFamily string
	// The width of chart, default width is 600
	Width int
	// The height of chart, default height is 400
	Height int
	// The padding of the whole canvas
	Padding Box
	// The gap between panels, default value is 10
	Gap int
	// The number of rows. Zero means it is derived from the column count.
	Rows int
	// The number of columns. Zero means it is derived from the row count.
	Columns int
	// Share the same Y scale across all panels.
	// When enabled, every value is positioned against the global min/max,
	// so the same value is drawn at the same height in every panel.
	ShareYAxis bool
	// Share the same X axis across all panels. Only category labels that
	// are identical across every panel are merged; inner panels hide their
	// X axis and only the bottom row keeps its labels.
	ShareXAxis bool
	// An optional title rendered at the top of the whole canvas
	Title TitleOption
	// An optional common legend rendered at the top of the whole canvas.
	// When empty, the series names of all panels are collected automatically.
	Legend LegendOption
	// The panels rendered into the grid, in row-major order. A nil entry
	// (or one without any series) produces an empty panel.
	Charts []ChartOption
}

// FacetChart renders multiple charts as small multiples (facets) and
// returns the painter of the whole canvas.
func FacetChart(opt FacetChartOption) (*Painter, error) {
	panelCount := len(opt.Charts)
	if panelCount == 0 {
		return nil, errors.New("facet chart requires at least one chart")
	}
	rows, columns, err := normalizeFacetGrid(opt.Rows, opt.Columns, panelCount)
	if err != nil {
		return nil, err
	}

	width := getDefaultInt(opt.Width, defaultChartWidth)
	height := getDefaultInt(opt.Height, defaultChartHeight)
	gap := opt.Gap
	if gap <= 0 {
		gap = 10
	}
	padding := opt.Padding
	if padding.IsZero() {
		padding = Box{
			Top:    20,
			Right:  20,
			Bottom: 20,
			Left:   20,
		}
	}
	// Validate the canvas geometry up front so a too small canvas fails
	// instead of silently producing a negative-sized painter.
	contentWidth := width - padding.Left - padding.Right
	contentHeight := height - padding.Top - padding.Bottom
	if contentWidth <= 0 || contentHeight <= 0 {
		return nil, errors.New("facet chart canvas is too small for the panels")
	}

	font, _ := GetFont(opt.FontFamily)
	if font == nil {
		font, _ = GetDefaultFont()
	}
	root, err := NewPainter(PainterOptions{
		Type:   opt.Type,
		Width:  width,
		Height: height,
		Font:   font,
	})
	if err != nil {
		return nil, err
	}
	theme := NewTheme(opt.Theme)
	if font != nil {
		theme.SetFont(font)
	}
	root.theme = theme
	root.SetBackground(width, height, theme.GetBackgroundColor())

	content := root.Child(PainterPaddingOption(padding))

	// Optional overall title
	topOffset := 0
	if opt.Title.Text != "" {
		titleOpt := opt.Title
		titleOpt.Theme = theme
		if titleOpt.Font == nil {
			titleOpt.Font = font
		}
		titlePainter := NewTitlePainter(content, titleOpt)
		titleBox, titleErr := titlePainter.Render()
		if titleErr != nil {
			return nil, titleErr
		}
		topOffset = titleBox.Height() + gap
	}

	// Collect the union of series names for the common legend.
	legendNames := opt.Legend.Data
	if len(legendNames) == 0 || opt.Legend.IsEmpty() {
		legendNames = collectFacetLegendNames(opt.Charts)
	}
	legendHeight := 0
	if len(legendNames) != 0 && !isFalse(opt.Legend.Show) {
		legendOpt := opt.Legend
		legendOpt.Theme = theme
		legendOpt.Data = legendNames
		// The legend occupies its own band at the top of the canvas
		legendPainter := NewLegendPainter(
			content.Child(PainterPaddingOption(Box{Top: topOffset})),
			legendOpt,
		)
		legendBox, legendErr := legendPainter.Render()
		if legendErr != nil {
			return nil, legendErr
		}
		legendHeight = legendBox.Height()
		if legendHeight > 0 {
			legendHeight += gap
		}
	}

	grid := content.Child(PainterPaddingOption(Box{
		Top: topOffset + legendHeight,
	}))
	gridHeight := contentHeight - topOffset - legendHeight
	if grid.Width() <= 0 || gridHeight <= 0 {
		return nil, errors.New("facet chart canvas is too small for the panels")
	}

	panelWidth := (grid.Width() - gap*(columns-1)) / columns
	panelHeight := (grid.Height() - gap*(rows-1)) / rows
	if panelWidth <= 0 || panelHeight <= 0 {
		return nil, errors.New("facet chart canvas is too small for the panels")
	}

	// Stable color for a series name across all panels
	colorIndexDict := make(map[string]int, len(legendNames))
	for index, name := range legendNames {
		colorIndexDict[name] = index
	}

	// Global Y scale shared by all panels
	sharedYRange, err := buildFacetSharedYRange(opt.Charts, grid, opt.ShareYAxis)
	if err != nil {
		return nil, err
	}
	// Common X labels shared by all panels (only if identical)
	sharedXLabels, shareX := buildFacetSharedXLabels(opt.Charts, opt.ShareXAxis)

	// Uniform left gutter so the plot areas of all panels align.
	// Shared scales produce identical labels, hence identical widths, but
	// the width is still measured explicitly to pin the gutter in place.
	var leftGutter int
	if opt.ShareYAxis {
		leftGutter = measureFacetYLabelWidth(grid, sharedYRange.Values(), font)
	} else {
		leftGutter = measureFacetMaxYAxisWidth(grid, opt.Charts, font)
	}
	// Uniform title band and bottom band so plot tops/bottoms align
	titleBand := measureFacetMaxTitleHeight(grid, opt.Charts, font)
	bottomBand := defaultXAxisHeight
	if shareX {
		bottomBand = measureFacetMaxXLabelHeight(grid, opt.Charts, font)
	}
	// Fail clearly when the reserved bands leave no usable plot area.
	// defaultRender subtracts an X axis band of defaultXAxisHeight again,
	// so the minimum accounts for both bands and the panel padding.
	plotHeight := panelHeight - titleBand - bottomBand - defaultXAxisHeight - gap*2
	plotWidth := panelWidth - leftGutter - gap*2
	if plotHeight <= 0 || plotWidth <= 0 {
		return nil, errors.New("facet chart canvas is too small for the panels")
	}

	// The last row that actually contains a panel. Trailing empty slots
	// still occupy grid cells but must not suppress the labels of the last
	// rendered row above them.
	lastRenderedRow := 0
	for index, chartOpt := range opt.Charts {
		if len(chartOpt.SeriesList) != 0 {
			lastRenderedRow = index / columns
		}
	}

	for index, chartOpt := range opt.Charts {
		row := index / columns
		column := index % columns
		boxLeft := column * (panelWidth + gap)
		boxTop := row * (panelHeight + gap)
		panelBox := Box{
			Left:   boxLeft,
			Top:    boxTop,
			Right:  boxLeft + panelWidth,
			Bottom: boxTop + panelHeight,
		}
		isLastRow := row == lastRenderedRow
		isFirstColumn := column == 0

		// Empty panel: keep its slot but render nothing inside it
		if len(chartOpt.SeriesList) == 0 {
			continue
		}

		panelOpt := cloneFacetChartOption(chartOpt)

		// The panel legend is hidden because it is rendered once at the
		// canvas level, but its labels are kept so Render still resolves the
		// series names through fillDefault.
		panelLegendData := panelOpt.Legend.Data
		if len(panelLegendData) == 0 {
			panelLegendData = legendNames
		}
		panelOpt.Legend = LegendOption{
			Show: FalseFlag(),
			Data: panelLegendData,
		}
		// Resolve series names first, then pin colors across panels
		facetResolveSeriesNames(&panelOpt)
		facetPreparePanel(&panelOpt, colorIndexDict)
		panelOpt.Parent = root
		panelOpt.Box = panelBox
		panelOpt.Width = panelWidth
		panelOpt.Height = panelHeight

		applyFacetLayoutPadding(&panelOpt, facetPadding{
			leftGutter: leftGutter,
			titleBand:  titleBand,
			bottomBand: bottomBand,
			gap:        gap,
		})

		if opt.ShareYAxis {
			applyFacetSharedYAxis(&panelOpt, sharedYRange, isFirstColumn)
		} else if !isFirstColumn {
			// Independent scales still align their plot areas: suppress the
			// duplicated Y labels of inner columns while keeping the axis
			// line and the reserved gutter.
			hideFacetYAxisLabels(&panelOpt)
		}

		if shareX {
			panelOpt.XAxis.Data = sharedXLabels
			if !isLastRow {
				panelOpt.XAxis.HideLabel = true
			}
		} else if !isLastRow {
			panelOpt.XAxis.HideLabel = true
		}

		if _, renderErr := Render(panelOpt); renderErr != nil {
			return nil, renderErr
		}
	}

	return root, nil
}

type facetPadding struct {
	leftGutter int
	titleBand  int
	bottomBand int
	gap        int
}

// normalizeFacetGrid resolves the row/column counts of the grid.
func normalizeFacetGrid(rows, columns, panelCount int) (int, int, error) {
	if rows < 0 || columns < 0 {
		return 0, 0, errors.New("facet chart rows and columns can not be negative")
	}
	if rows == 0 && columns == 0 {
		// A single row by default
		columns = panelCount
	}
	if rows == 0 {
		rows = int(math.Ceil(float64(panelCount) / float64(columns)))
	}
	if columns == 0 {
		columns = int(math.Ceil(float64(panelCount) / float64(rows)))
	}
	if rows <= 0 || columns <= 0 || rows*columns < panelCount {
		return 0, 0, errors.New("facet chart grid can not hold all the panels")
	}
	return rows, columns, nil
}

// cloneFacetChartOption returns a shallow copy of the chart option so the
// facet layout never mutates the caller's option. Slices are copied as well.
func cloneFacetChartOption(opt ChartOption) ChartOption {
	clone := opt
	clone.SeriesList = make(SeriesList, len(opt.SeriesList))
	for index, series := range opt.SeriesList {
		cloneSeries := series
		cloneSeries.Data = append([]SeriesData(nil), series.Data...)
		clone.SeriesList[index] = cloneSeries
	}
	clone.YAxisOptions = append([]YAxisOption(nil), opt.YAxisOptions...)
	clone.Children = nil
	clone.Parent = nil
	return clone
}

// collectFacetLegendNames collects the distinct series names of all panels.
// A panel legend carries the names before Render runs, because unnamed
// series are only named during Render's fillDefault step.
func collectFacetLegendNames(charts []ChartOption) []string {
	names := make([]string, 0)
	exists := map[string]bool{}
	add := func(name string) {
		if name == "" || exists[name] {
			return
		}
		exists[name] = true
		names = append(names, name)
	}
	for _, chartOpt := range charts {
		for index, series := range chartOpt.SeriesList {
			if series.Name != "" {
				add(series.Name)
				continue
			}
			// fall back to the panel legend label of the same position
			if index < len(chartOpt.Legend.Data) {
				add(chartOpt.Legend.Data[index])
			}
		}
	}
	return names
}

// facetResolveSeriesNames mirrors the name resolution done by
// ChartOption.fillDefault when a legend is present, without sorting the
// series (the facet keeps the caller's order).
func facetResolveSeriesNames(opt *ChartOption) {
	if len(opt.Legend.Data) == 0 {
		return
	}
	for index, name := range opt.Legend.Data {
		if index < len(opt.SeriesList) && opt.SeriesList[index].Name == "" {
			opt.SeriesList[index].Name = name
		}
	}
}

// facetPreparePanel makes the color of every named series stable across
// panels. Unnamed series keep the color derived from their own position,
// matching the standalone rendering behavior.
func facetPreparePanel(opt *ChartOption, colorIndexDict map[string]int) {
	for index := range opt.SeriesList {
		series := &opt.SeriesList[index]
		if series.Name == "" {
			continue
		}
		if colorIndex, ok := colorIndexDict[series.Name]; ok {
			// Pin the palette color of this series across all panels.
			// SeriesList.init keeps positive ColorIndex values untouched.
			series.ColorIndex = colorIndex
		}
	}
}

// applyFacetLayoutPadding reserves uniform gutters/bands on every panel so
// that all plot areas share the same left, top and bottom offsets.
func applyFacetLayoutPadding(opt *ChartOption, p facetPadding) {
	opt.Padding = Box{
		Top:    p.titleBand + p.gap,
		Right:  p.gap,
		Bottom: p.bottomBand + p.gap,
		Left:   p.leftGutter + p.gap,
	}
}

// hideFacetYAxisLabels suppresses the Y labels of every panel axis while
// leaving the axis line itself visible (used by inner columns with
// independent scales, where the gutter is still reserved).
func hideFacetYAxisLabels(opt *ChartOption) {
	if len(opt.YAxisOptions) == 0 {
		opt.YAxisOptions = []YAxisOption{
			{
				HideLabel: true,
			},
		}
		return
	}
	for index := range opt.YAxisOptions {
		opt.YAxisOptions[index].HideLabel = true
	}
}

// applyFacetSharedYAxis pins every Y axis of a panel to the global range.
// Inner columns suppress their Y labels because the first column shows
// them once.
func applyFacetSharedYAxis(opt *ChartOption, r axisRange, isFirstColumn bool) {
	if len(opt.YAxisOptions) == 0 {
		opt.YAxisOptions = make([]YAxisOption, 1)
	}
	for index := range opt.YAxisOptions {
		yOpt := &opt.YAxisOptions[index]
		minValue := r.min
		maxValue := r.max
		yOpt.Min = &minValue
		yOpt.Max = &maxValue
		if !isFirstColumn {
			yOpt.HideLabel = true
		}
	}
}

// buildFacetSharedYRange computes the global Y range of all panels. It is
// only used when ShareYAxis is enabled.
func buildFacetSharedYRange(charts []ChartOption, p *Painter, enabled bool) (axisRange, error) {
	empty := axisRange{}
	if !enabled {
		return empty, nil
	}
	min := math.MaxFloat64
	max := -math.MaxFloat64
	hasValue := false
	for _, chartOpt := range charts {
		for _, series := range chartOpt.SeriesList {
			for _, item := range series.Data {
				if item.Value == nullValue {
					continue
				}
				hasValue = true
				if item.Value < min {
					min = item.Value
				}
				if item.Value > max {
					max = item.Value
				}
			}
		}
	}
	if !hasValue {
		return empty, errors.New("share y axis requires at least one value")
	}
	if !isFinite(min) || !isFinite(max) || min > max {
		return empty, errors.New("invalid shared y axis range")
	}
	r := NewRange(AxisRangeOption{
		Painter:     p,
		Min:         min,
		Max:         max,
		Size:        p.Height(),
		DivideCount: defaultAxisDivideCount,
	})
	if !isFinite(r.min) || !isFinite(r.max) || r.min >= r.max {
		return empty, errors.New("invalid shared y axis range")
	}
	return r, nil
}

// isFinite reports whether v is neither NaN nor an infinity.
func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// buildFacetSharedXLabels returns the common X labels only when every panel
// defines identical labels, otherwise sharing is disabled.
func buildFacetSharedXLabels(charts []ChartOption, enabled bool) ([]string, bool) {
	if !enabled {
		return nil, false
	}
	var shared []string
	for _, chartOpt := range charts {
		if len(chartOpt.SeriesList) == 0 {
			continue
		}
		labels := chartOpt.XAxis.Data
		if len(labels) == 0 {
			return nil, false
		}
		if shared == nil {
			shared = labels
			continue
		}
		if len(shared) != len(labels) {
			return nil, false
		}
		for index, label := range labels {
			if shared[index] != label {
				return nil, false
			}
		}
	}
	if shared == nil {
		return nil, false
	}
	return shared, true
}

// measureFacetYLabelWidth measures the Y axis block width for a fixed set
// of labels, mirroring the spacing reserved by the axis painter.
func measureFacetYLabelWidth(p *Painter, labels []string, font *truetype.Font) int {
	child := p.Child(PainterFontOption(font))
	textWidth, _ := child.MeasureTextMaxWidthHeight(labels)
	return textWidth + 10
}

// measureFacetMaxYAxisWidth measures the widest Y axis label block of all
// panels so that independent-scale panels still get aligned plot areas.
func measureFacetMaxYAxisWidth(p *Painter, charts []ChartOption, font *truetype.Font) int {
	maxWidth := 0
	for _, chartOpt := range charts {
		if len(chartOpt.SeriesList) == 0 {
			continue
		}
		max, min := chartOpt.SeriesList.GetMaxMin(0)
		if min == math.MaxFloat64 {
			continue
		}
		divideCount := defaultAxisDivideCount
		if len(chartOpt.YAxisOptions) > 0 && chartOpt.YAxisOptions[0].DivideCount > 0 {
			divideCount = chartOpt.YAxisOptions[0].DivideCount
		}
		r := NewRange(AxisRangeOption{
			Painter:     p,
			Min:         min,
			Max:         max,
			Size:        p.Height(),
			DivideCount: divideCount,
		})
		width := measureFacetYLabelWidth(p, r.Values(), font)
		if width > maxWidth {
			maxWidth = width
		}
	}
	return maxWidth
}

// measureFacetMaxTitleHeight measures the tallest panel title band.
func measureFacetMaxTitleHeight(p *Painter, charts []ChartOption, font *truetype.Font) int {
	maxHeight := 0
	for _, chartOpt := range charts {
		if chartOpt.Title.Text == "" {
			continue
		}
		// One text line by default, titles with new lines are split
		lines := splitTitleText(chartOpt.Title.Text)
		if len(lines) == 0 {
			continue
		}
		child := p.Child(PainterFontOption(font))
		_, textHeight := child.MeasureTextMaxWidthHeight(lines)
		if textHeight > maxHeight {
			maxHeight = textHeight
		}
	}
	return maxHeight
}

// measureFacetMaxXLabelHeight measures the tallest X label block so the
// bottom band reserves enough room for long labels. The axis painter always
// works within defaultXAxisHeight, so the measured value never exceeds it.
func measureFacetMaxXLabelHeight(p *Painter, charts []ChartOption, font *truetype.Font) int {
	maxHeight := 0
	for _, chartOpt := range charts {
		if len(chartOpt.XAxis.Data) == 0 {
			continue
		}
		child := p.Child(PainterFontOption(font))
		_, textHeight := child.MeasureTextMaxWidthHeight(chartOpt.XAxis.Data)
		// the axis band adds tick spacing around the label
		height := textHeight + 10
		if height > maxHeight {
			maxHeight = height
		}
	}
	if maxHeight == 0 || maxHeight > defaultXAxisHeight {
		return defaultXAxisHeight
	}
	return maxHeight
}
