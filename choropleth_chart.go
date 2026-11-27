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
	"strconv"
	"strings"

	"github.com/golang/freetype/truetype"
)

// ChartTypeChoropleth is the chart type of planar choropleth chart
const ChartTypeChoropleth = "choropleth"

// Point2D is a point of the planar coordinate system.
// The coordinates are used directly without any geographic projection.
type Point2D struct {
	X float64
	Y float64
}

// MapRegion is a named closed polygon region with optional holes.
// Polygon is the outer ring and should be closed implicitly (the first
// vertex is joined to the last one). Holes is a list of inner rings,
// each of them is kept as the background color when rendered.
type MapRegion struct {
	// Name is the unique name of the region
	Name string
	// Polygon is the outer ring of the region
	Polygon []Point2D
	// Holes is the inner ring list of the region, e.g. lakes or voids
	Holes [][]Point2D
	// Value is the statistic value of the region, nil means missing value
	Value *float64
}

// VisualMapPiece is a piece of the segmented color scale.
// A value v belongs to the piece when Min <= v < Max.
type VisualMapPiece struct {
	// Min is the inclusive lower bound of the piece
	Min float64
	// Max is the exclusive upper bound of the piece
	Max float64
	// Label is the legend label, it is generated automatically when empty
	Label string
	// Color is the fill color of the piece, generated from the default
	// ramp when empty
	Color Color
}

// VisualMap is the configurable segmented color scale (piecewise visual map).
type VisualMap struct {
	// Min is the minimum value of the scale, the minimum of region values
	// is used when it is nil
	Min *float64
	// Max is the maximum value of the scale, the maximum of region values
	// is used when it is nil
	Max *float64
	// Pieces is the custom piece list. It should be ordered by Min and
	// cover the value range. Equal interval pieces are generated when empty.
	Pieces []VisualMapPiece
	// PieceCount is the count of equal interval pieces, default is 5
	PieceCount int
	// Colors is the ramp used by generated pieces, the default blue
	// sequential ramp is used when empty
	Colors []Color
	// ShowNoData shows a no data entry in the legend when true
	ShowNoData bool
	// NoDataLabel is the label of the no data legend entry
	NoDataLabel string
	// NoDataColor is the fill color of missing value regions, the
	// background color is used when it is zero (the region is not filled)
	NoDataColor Color
}

// ChoroplethChartOption is the option of planar choropleth chart
type ChoroplethChartOption struct {
	// The output type of chart, "svg" or "png", default is "png"
	Type string
	// The theme of chart, "light" and "dark"
	Theme string
	// The font family, which should be installed first
	FontFamily string
	// The font for chart
	Font *truetype.Font
	// The width of chart, default width is 600
	Width int
	// The height of chart, default height is 400
	Height int
	// The padding for chart
	Padding Box
	// The title option
	Title TitleOption
	// The region list
	Regions []MapRegion
	// The segmented color scale
	VisualMap VisualMap
	// The background color of chart
	BackgroundColor Color
	// BorderColor is the stroke color of region boundaries
	BorderColor Color
	// BorderWidth is the stroke width of region boundaries, default is 1
	BorderWidth float64
	// ShowLabel draws the region name at the centroid of each region
	ShowLabel bool
	// LabelColor is the color of region name labels
	LabelColor Color
	// LabelFontSize is the font size of region name labels, default is 10
	LabelFontSize float64
	// KeepAspect keeps the aspect ratio of planar coordinates when true.
	// The default value is false, each axis is scaled independently so
	// that the data bounds fill the plot area.
	KeepAspect bool
	// LegendOrient is the layout orientation of legend,
	// it can be horizontal or vertical, default is horizontal
	LegendOrient string
	// LegendLeft is the distance between legend and the left side,
	// it can be position value: right, center, or a number
	LegendLeft string
	// LegendTop is the distance between legend and the top side
	LegendTop string
	// LegendFontSize is the font size of the legend text
	LegendFontSize float64
}

// choroplethChart renders named polygon regions with a segmented color scale
type choroplethChart struct {
	p   *Painter
	opt *ChoroplethChartOption
}

// NewChoroplethChart returns a choropleth chart renderer
func NewChoroplethChart(p *Painter, opt ChoroplethChartOption) *choroplethChart {
	return &choroplethChart{
		p:   p,
		opt: &opt,
	}
}

// default sequential blue ramp (light to dark)
var defaultChoroplethColors = []Color{
	parseColor("#cde2fb"),
	parseColor("#86b6ef"),
	parseColor("#3987e5"),
	parseColor("#256abf"),
	parseColor("#104281"),
}

// defaultNoDataColor is the default color of missing value regions
var defaultNoDataColor = parseColor("#f0efec")

// maxCanvasCoord limits the coordinate passed to the renderer, a value
// outside the int32 range makes the raster renderer panic
const maxCanvasCoord = 1 << 28

var (
	errEmptyRegion       = errors.New("choropleth region should contain at least 3 vertices")
	errNonFiniteVertex   = errors.New("choropleth region contains a non-finite vertex")
	errSelfIntersectRing = errors.New("choropleth region ring is self-intersecting")
	errEmptyRegions      = errors.New("choropleth regions can not be empty")
	errInvalidVisualMap  = errors.New("choropleth visual map is invalid")
	errCanvasTooSmall    = errors.New("choropleth canvas is too small for the title and legend")
)

// isFinitePoint returns true when both x and y are finite numbers
func isFinitePoint(pt Point2D) bool {
	return !math.IsNaN(pt.X) && !math.IsInf(pt.X, 0) &&
		!math.IsNaN(pt.Y) && !math.IsInf(pt.Y, 0)
}

// ringSignedArea returns the signed area of a ring, positive for counter
// clockwise and negative for clockwise
func ringSignedArea(ring []Point2D) float64 {
	sum := float64(0)
	for i := 0; i < len(ring); i++ {
		a := ring[i]
		b := ring[(i+1)%len(ring)]
		sum += a.X*b.Y - b.X*a.Y
	}
	return sum / 2
}

// segmentsIntersect returns true when segment ab intersects segment cd.
// Coincident segments are treated as intersecting because they make the
// fill result ambiguous.
func segmentsIntersect(ax, ay, bx, by, cx, cy, dx, dy float64) bool {
	orientation := func(px, py, qx, qy, rx, ry float64) float64 {
		return (qx-px)*(ry-py) - (qy-py)*(rx-px)
	}
	onSegment := func(px, py, qx, qy, rx, ry float64) bool {
		return math.Min(px, rx) <= qx && qx <= math.Max(px, rx) &&
			math.Min(py, ry) <= qy && qy <= math.Max(py, ry)
	}
	o1 := orientation(ax, ay, bx, by, cx, cy)
	o2 := orientation(ax, ay, bx, by, dx, dy)
	o3 := orientation(cx, cy, dx, dy, ax, ay)
	o4 := orientation(cx, cy, dx, dy, bx, by)
	epsilon := 1e-10
	if math.Abs(o1) <= epsilon && onSegment(ax, ay, cx, cy, bx, by) {
		return true
	}
	if math.Abs(o2) <= epsilon && onSegment(ax, ay, dx, dy, bx, by) {
		return true
	}
	if math.Abs(o3) <= epsilon && onSegment(cx, cy, ax, ay, dx, dy) {
		return true
	}
	if math.Abs(o4) <= epsilon && onSegment(cx, cy, bx, by, dx, dy) {
		return true
	}
	return (o1 > 0) != (o2 > 0) && (o3 > 0) != (o4 > 0)
}

// isSimpleRing returns false when non adjacent edges of the ring intersect
func isSimpleRing(ring []Point2D) bool {
	count := len(ring)
	for i := 0; i < count; i++ {
		a := ring[i]
		b := ring[(i+1)%count]
		for j := i + 1; j < count; j++ {
			// adjacent edges share a vertex, skip them
			if j == i+1 || (i == 0 && j == count-1) {
				continue
			}
			c := ring[j]
			d := ring[(j+1)%count]
			if segmentsIntersect(a.X, a.Y, b.X, b.Y, c.X, c.Y, d.X, d.Y) {
				return false
			}
		}
	}
	return true
}

// validateRing checks a single ring
func validateRing(ring []Point2D) error {
	if len(ring) < 3 {
		return errEmptyRegion
	}
	for _, pt := range ring {
		if !isFinitePoint(pt) {
			return errNonFiniteVertex
		}
	}
	if !isSimpleRing(ring) {
		return errSelfIntersectRing
	}
	return nil
}

// validateRegions checks all regions and returns the value bounds
func validateRegions(regions []MapRegion) (float64, float64, error) {
	if len(regions) == 0 {
		return 0, 0, errEmptyRegions
	}
	min := math.MaxFloat64
	max := -math.MaxFloat64
	names := map[string]bool{}
	for _, region := range regions {
		if names[region.Name] {
			return 0, 0, errors.New("choropleth region name is duplicated: " + region.Name)
		}
		names[region.Name] = true
		if err := validateRing(region.Polygon); err != nil {
			return 0, 0, err
		}
		for _, hole := range region.Holes {
			if err := validateRing(hole); err != nil {
				return 0, 0, err
			}
		}
		if region.Value == nil || math.IsNaN(*region.Value) || math.IsInf(*region.Value, 0) {
			continue
		}
		if *region.Value < min {
			min = *region.Value
		}
		if *region.Value > max {
			max = *region.Value
		}
	}
	if min == math.MaxFloat64 {
		// no finite value, use a neutral range
		return 0, 1, nil
	}
	return min, max, nil
}

// formatPieceLabel formats the legend label of a piece
func formatPieceLabel(min, max float64) string {
	format := func(v float64) string {
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return "[" + format(min) + ", " + format(max) + ")"
}

// normalizeVisualMap completes the visual map option and returns the pieces
func normalizeVisualMap(vm *VisualMap, dataMin, dataMax float64) ([]VisualMapPiece, error) {
	if len(vm.Pieces) != 0 {
		pieces := make([]VisualMapPiece, len(vm.Pieces))
		copy(pieces, vm.Pieces)
		for i := range pieces {
			if pieces[i].Max <= pieces[i].Min {
				return nil, errInvalidVisualMap
			}
			if pieces[i].Label == "" {
				pieces[i].Label = formatPieceLabel(pieces[i].Min, pieces[i].Max)
			}
			if pieces[i].Color.IsZero() {
				pieces[i].Color = defaultChoroplethColors[i%len(defaultChoroplethColors)]
			}
		}
		// pieces should be ordered, sort by Min to make lookups stable
		sort.Slice(pieces, func(i, j int) bool {
			return pieces[i].Min < pieces[j].Min
		})
		return pieces, nil
	}
	min := dataMin
	max := dataMax
	if vm.Min != nil {
		min = *vm.Min
	}
	if vm.Max != nil {
		max = *vm.Max
	}
	if max <= min {
		return nil, errInvalidVisualMap
	}
	count := vm.PieceCount
	if count <= 0 {
		count = 5
	}
	colors := vm.Colors
	if len(colors) == 0 {
		colors = defaultChoroplethColors
	}
	step := (max - min) / float64(count)
	pieces := make([]VisualMapPiece, count)
	for i := 0; i < count; i++ {
		lo := min + float64(i)*step
		hi := min + float64(i+1)*step
		pieces[i] = VisualMapPiece{
			Min:   lo,
			Max:   hi,
			Label: formatPieceLabel(lo, hi),
			Color: colors[i*len(colors)/count],
		}
	}
	return pieces, nil
}

// pieceColor returns the fill color of a value, the zero color and false
// mean that no piece matches the value. Pieces are ordered by Min.
func pieceColor(pieces []VisualMapPiece, value float64) (Color, bool) {
	// the maximum value belongs to the last piece (its upper bound is equal)
	if len(pieces) != 0 && value == pieces[len(pieces)-1].Max {
		return pieces[len(pieces)-1].Color, true
	}
	// find the last piece whose Min is not greater than the value
	idx := sort.Search(len(pieces), func(i int) bool {
		return pieces[i].Min > value
	}) - 1
	if idx < 0 || value >= pieces[idx].Max {
		return Color{}, false
	}
	return pieces[idx].Color, true
}

// coordTransform converts planar coordinates to painter coordinates
type coordTransform struct {
	scaleX  float64
	scaleY  float64
	minX    float64
	maxY    float64
	offsetX float64
	offsetY float64
}

// bounds is the planar bounding box of all region vertices
type bounds struct {
	minX float64
	maxX float64
	minY float64
	maxY float64
}

func regionBounds(regions []MapRegion) bounds {
	b := bounds{
		minX: math.MaxFloat64,
		maxX: -math.MaxFloat64,
		minY: math.MaxFloat64,
		maxY: -math.MaxFloat64,
	}
	accept := func(pt Point2D) {
		if pt.X < b.minX {
			b.minX = pt.X
		}
		if pt.X > b.maxX {
			b.maxX = pt.X
		}
		if pt.Y < b.minY {
			b.minY = pt.Y
		}
		if pt.Y > b.maxY {
			b.maxY = pt.Y
		}
	}
	for _, region := range regions {
		for _, pt := range region.Polygon {
			accept(pt)
		}
		for _, hole := range region.Holes {
			for _, pt := range hole {
				accept(pt)
			}
		}
	}
	return b
}

// newCoordTransform builds the transform, Y axis is flipped because the
// painter coordinate system grows downward
func newCoordTransform(b bounds, width, height int, keepAspect bool) coordTransform {
	dataWidth := b.maxX - b.minX
	dataHeight := b.maxY - b.minY
	if dataWidth <= 0 {
		dataWidth = 1
	}
	if dataHeight <= 0 {
		dataHeight = 1
	}
	scaleX := float64(width) / dataWidth
	scaleY := float64(height) / dataHeight
	if keepAspect {
		scale := math.Min(scaleX, scaleY)
		scaleX = scale
		scaleY = scale
	}
	usedWidth := dataWidth * scaleX
	usedHeight := dataHeight * scaleY
	return coordTransform{
		scaleX:  scaleX,
		scaleY:  scaleY,
		minX:    b.minX,
		maxY:    b.maxY,
		offsetX: (float64(width) - usedWidth) / 2,
		offsetY: (float64(height) - usedHeight) / 2,
	}
}

// apply converts a planar point to a clipped painter point.
// The Y axis is flipped so that a larger planar Y is drawn higher.
func (t coordTransform) apply(pt Point2D) Point {
	fx := t.offsetX + (pt.X-t.minX)*t.scaleX
	fy := t.offsetY + (t.maxY-pt.Y)*t.scaleY
	return Point{
		X: clampCanvasInt(fx),
		Y: clampCanvasInt(fy),
	}
}

// clampCanvasInt limits a float coordinate to the safe integer range
func clampCanvasInt(v float64) int {
	if math.IsNaN(v) {
		return 0
	}
	if v > maxCanvasCoord {
		return maxCanvasCoord
	}
	if v < -maxCanvasCoord {
		return -maxCanvasCoord
	}
	return int(math.Round(v))
}

// ringCentroid returns the area centroid of a ring
func ringCentroid(ring []Point2D) Point2D {
	area := ringSignedArea(ring)
	if math.Abs(area) < 1e-12 {
		// fallback to the average of vertices
		pt := Point2D{}
		for _, item := range ring {
			pt.X += item.X
			pt.Y += item.Y
		}
		pt.X /= float64(len(ring))
		pt.Y /= float64(len(ring))
		return pt
	}
	var cx, cy float64
	for i := 0; i < len(ring); i++ {
		a := ring[i]
		b := ring[(i+1)%len(ring)]
		factor := a.X*b.Y - b.X*a.Y
		cx += (a.X + b.X) * factor
		cy += (a.Y + b.Y) * factor
	}
	divider := 6 * area
	return Point2D{
		X: cx / divider,
		Y: cy / divider,
	}
}

// pointInRing reports whether pt is inside a simple ring using the
// ray casting algorithm
func pointInRing(pt Point2D, ring []Point2D) bool {
	inside := false
	count := len(ring)
	for i, j := 0, count-1; i < count; j, i = i, i+1 {
		a := ring[j]
		b := ring[i]
		if (b.Y > pt.Y) != (a.Y > pt.Y) &&
			pt.X < (a.X-b.X)*(pt.Y-b.Y)/(a.Y-b.Y)+b.X {
			inside = !inside
		}
	}
	return inside
}

// pointInHole reports whether pt is inside any hole
func pointInHole(pt Point2D, holes [][]Point2D) bool {
	for _, hole := range holes {
		if pointInRing(pt, hole) {
			return true
		}
	}
	return false
}

// regionLabelPoint returns a point inside the region but outside every hole
// for placing the region name label. The area centroid is preferred, and an
// interior point sampled from the bounding box is used when the centroid
// falls in a hole.
func regionLabelPoint(region MapRegion) Point2D {
	center := ringCentroid(region.Polygon)
	if pointInRing(center, region.Polygon) && !pointInHole(center, region.Holes) {
		return center
	}
	// fallback: sample a grid of points inside the bounding box and keep
	// the first one that is inside the region and outside every hole
	b := regionBounds([]MapRegion{region})
	const steps = 8
	for i := 1; i <= steps; i++ {
		for j := 1; j <= steps; j++ {
			pt := Point2D{
				X: b.minX + (b.maxX-b.minX)*float64(i)/float64(steps+1),
				Y: b.minY + (b.maxY-b.minY)*float64(j)/float64(steps+1),
			}
			if pointInRing(pt, region.Polygon) && !pointInHole(pt, region.Holes) {
				return pt
			}
		}
	}
	return center
}

// drawRingPath traces a ring with the given painter, the winding direction
// is adjusted by reverse so that holes are rendered correctly with both
// the nonzero (SVG) and even-odd (PNG) fill rules
func drawRingPath(p *Painter, ring []Point2D, t coordTransform, reverse bool) {
	count := len(ring)
	for i := 0; i < count; i++ {
		idx := i
		if reverse {
			idx = count - 1 - i
		}
		pt := t.apply(ring[idx])
		if i == 0 {
			p.MoveTo(pt.X, pt.Y)
		} else {
			p.LineTo(pt.X, pt.Y)
		}
	}
	p.Close()
}

// drawRegion fills and strokes a region. The outer ring and every hole are
// put into a single path; hole winding is reversed against the outer ring
// so the hole keeps the background color on both renderers.
func (c *choroplethChart) drawRegion(p *Painter, region MapRegion, t coordTransform, fillColor Color, hasFill bool, borderColor Color, borderWidth float64) {
	outerArea := ringSignedArea(region.Polygon)
	outerCCW := outerArea > 0
	if hasFill {
		p.SetDrawingStyle(Style{
			FillColor: fillColor,
		})
		drawRingPath(p, region.Polygon, t, false)
		for _, hole := range region.Holes {
			holeCCW := ringSignedArea(hole) > 0
			// reverse the hole when it has the same winding as the outer ring
			reverse := holeCCW == outerCCW
			drawRingPath(p, hole, t, reverse)
		}
		p.Fill()
	}
	// boundaries are stroked separately so that a missing value region
	// still shows its outline and shared edges stay visible
	p.SetDrawingStyle(Style{
		StrokeColor: borderColor,
		StrokeWidth: borderWidth,
	})
	drawRingPath(p, region.Polygon, t, false)
	p.Stroke()
	for _, hole := range region.Holes {
		drawRingPath(p, hole, t, false)
		p.Stroke()
	}
}

// legendLayout is the measured size of the legend
type legendLayout struct {
	width  int
	height int
}

const (
	legendIconWidth  = 20
	legendIconHeight = 14
	legendItemGap    = 16
	legendTextGap    = 6
	legendRowGap     = 8
)

// measureLegend measures the legend without drawing it
func (c *choroplethChart) measureLegend(p *Painter, pieces []VisualMapPiece, vm VisualMap) (legendLayout, []string) {
	labels := make([]string, 0, len(pieces)+1)
	for _, piece := range pieces {
		labels = append(labels, piece.Label)
	}
	if vm.ShowNoData {
		label := vm.NoDataLabel
		if label == "" {
			label = "No Data"
		}
		labels = append(labels, label)
	}
	fontSize := c.opt.LegendFontSize
	if fontSize == 0 {
		fontSize = labelFontSize
	}
	p.SetTextStyle(Style{
		FontSize: fontSize,
		Font:     c.opt.Font,
	})
	maxTextWidth := 0
	for _, label := range labels {
		box := p.MeasureText(label)
		if box.Width() > maxTextWidth {
			maxTextWidth = box.Width()
		}
	}
	if c.opt.LegendOrient == OrientVertical {
		width := legendIconWidth + legendTextGap + maxTextWidth
		height := len(labels)*(legendIconHeight+legendRowGap) - legendRowGap
		return legendLayout{width: width, height: height}, labels
	}
	width := 0
	for _, label := range labels {
		box := p.MeasureText(label)
		width += legendIconWidth + legendTextGap + box.Width() + legendItemGap
	}
	width -= legendItemGap
	return legendLayout{width: width, height: legendIconHeight}, labels
}

// drawLegend draws the segmented color scale legend
func (c *choroplethChart) drawLegend(p *Painter, pieces []VisualMapPiece, vm VisualMap, labels []string, layout legendLayout) {
	fontSize := c.opt.LegendFontSize
	if fontSize == 0 {
		fontSize = labelFontSize
	}
	theme := p.theme
	textColor := theme.GetTextColor()
	p.SetTextStyle(Style{
		FontSize:  fontSize,
		FontColor: textColor,
		Font:      c.opt.Font,
	})
	left := 0
	switch c.opt.LegendLeft {
	case PositionRight:
		left = p.Width() - layout.width
	case PositionCenter, "":
		left = (p.Width() - layout.width) >> 1
	default:
		if strings.HasSuffix(c.opt.LegendLeft, "%") {
			value, _ := strconv.Atoi(strings.ReplaceAll(c.opt.LegendLeft, "%", ""))
			left = p.Width() * value / 100
		} else {
			value, _ := strconv.Atoi(c.opt.LegendLeft)
			left = value
		}
	}
	top, _ := strconv.Atoi(c.opt.LegendTop)
	if top == 0 {
		top = legendRowGap
	}
	x := left
	y := top
	for index, label := range labels {
		var color Color
		if index < len(pieces) {
			color = pieces[index].Color
		} else {
			color = vm.NoDataColor
			if color.IsZero() {
				color = defaultNoDataColor
			}
		}
		p.SetDrawingStyle(Style{
			FillColor:   color,
			StrokeColor: color,
		})
		p.Rect(Box{
			Left:   x,
			Top:    y,
			Right:  x + legendIconWidth,
			Bottom: y + legendIconHeight,
		})
		p.SetTextStyle(Style{
			FontSize:  fontSize,
			FontColor: textColor,
			Font:      c.opt.Font,
		})
		textBox := p.MeasureText(label)
		p.Text(label, x+legendIconWidth+legendTextGap, y+(legendIconHeight+textBox.Height())/2-1)
		if c.opt.LegendOrient == OrientVertical {
			x = left
			y += legendIconHeight + legendRowGap
		} else {
			x += legendIconWidth + legendTextGap + textBox.Width() + legendItemGap
		}
	}
}

// Render renders the choropleth chart
func (c *choroplethChart) Render() (Box, error) {
	opt := c.opt
	if opt.Width <= 0 {
		opt.Width = defaultChartWidth
	}
	if opt.Height <= 0 {
		opt.Height = defaultChartHeight
	}
	if opt.Font == nil {
		opt.Font, _ = GetFont(opt.FontFamily)
		if opt.Font == nil {
			opt.Font, _ = GetDefaultFont()
		}
	}
	theme := NewTheme(opt.Theme)
	if theme == nil {
		theme = NewTheme(ThemeLight)
	}
	backgroundColor := opt.BackgroundColor
	if backgroundColor.IsZero() {
		backgroundColor = theme.GetBackgroundColor()
	}
	dataMin, dataMax, err := validateRegions(opt.Regions)
	if err != nil {
		return BoxZero, err
	}
	pieces, err := normalizeVisualMap(&opt.VisualMap, dataMin, dataMax)
	if err != nil {
		return BoxZero, err
	}
	p, err := NewPainter(PainterOptions{
		Type:   opt.Type,
		Width:  opt.Width,
		Height: opt.Height,
		Font:   opt.Font,
	})
	if err != nil {
		return BoxZero, err
	}
	c.p = p
	p.SetBackground(opt.Width, opt.Height, backgroundColor)

	// the painter defaults to the light theme, attach the requested theme
	// so legend and label colors follow it (e.g. light text on dark theme)
	root := p.Child(PainterThemeOption(theme))
	if !opt.Padding.IsZero() {
		root = root.Child(PainterPaddingOption(opt.Padding))
	}
	// title
	titleHeight := 0
	if opt.Title.Text != "" {
		opt.Title.Theme = theme
		opt.Title.Font = opt.Font
		titleBox, err := NewTitlePainter(root, opt.Title).Render()
		if err != nil {
			return BoxZero, err
		}
		titleHeight = titleBox.Height() + 20
	}
	// legend measurement reserves the bottom area
	legendChild := root.Child(PainterPaddingOption(Box{
		Top: titleHeight,
	}))
	layout, labels := c.measureLegend(legendChild, pieces, opt.VisualMap)
	legendSpace := layout.height + 20
	plotPainter := root.Child(PainterPaddingOption(Box{
		Top:    titleHeight,
		Bottom: legendSpace,
	}))
	if plotPainter.Width() <= 0 || plotPainter.Height() <= 0 {
		return BoxZero, errCanvasTooSmall
	}
	// draw legend at the bottom
	legendPainter := root.Child(PainterPaddingOption(Box{
		Top: plotPainter.box.Bottom - root.box.Top,
	}))
	c.drawLegend(legendPainter, pieces, opt.VisualMap, labels, layout)

	b := regionBounds(opt.Regions)
	t := newCoordTransform(b, plotPainter.Width(), plotPainter.Height(), opt.KeepAspect)
	borderColor := opt.BorderColor
	if borderColor.IsZero() {
		borderColor = theme.GetAxisStrokeColor()
	}
	borderWidth := opt.BorderWidth
	if borderWidth <= 0 {
		borderWidth = 1
	}
	noDataColor := opt.VisualMap.NoDataColor
	for _, region := range opt.Regions {
		fillColor := Color{}
		hasFill := false
		if region.Value != nil && !math.IsNaN(*region.Value) && !math.IsInf(*region.Value, 0) {
			if color, ok := pieceColor(pieces, *region.Value); ok {
				fillColor = color
				hasFill = true
			}
		}
		if !hasFill && !noDataColor.IsZero() {
			fillColor = noDataColor
			hasFill = true
		}
		c.drawRegion(plotPainter, region, t, fillColor, hasFill, borderColor, borderWidth)
		if opt.ShowLabel {
			center := regionLabelPoint(region)
			pt := t.apply(center)
			labelColor := opt.LabelColor
			if labelColor.IsZero() {
				labelColor = theme.GetTextColor()
			}
			fontSize := opt.LabelFontSize
			if fontSize == 0 {
				fontSize = smallLabelFontSize
			}
			plotPainter.SetTextStyle(Style{
				FontSize:  fontSize,
				FontColor: labelColor,
				Font:      opt.Font,
			})
			box := plotPainter.MeasureText(region.Name)
			plotPainter.Text(region.Name, pt.X-box.Width()/2, pt.Y+box.Height()/2-1)
		}
	}
	return p.box, nil
}

// ChoroplethRender renders a planar choropleth chart and returns the painter
func ChoroplethRender(opt ChoroplethChartOption) (*Painter, error) {
	if opt.Type == "" {
		opt.Type = ChartOutputPNG
	}
	c := NewChoroplethChart(nil, opt)
	_, err := c.Render()
	if err != nil {
		return nil, err
	}
	return c.p, nil
}
