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

// ChartTypeChord is the series type of directed chord chart
const ChartTypeChord = "chord"

const (
	// defaultChordGap is the gap angle (in radians) between two node arcs
	defaultChordGap = 2 * math.Pi / 360
	// defaultChordRingWidth is the default thickness of the node ring
	defaultChordRingWidth = 12.0
	// defaultChordRadiusPercent is the default outer radius percentage
	defaultChordRadiusPercent = 0.40
	// defaultChordRibbonAlpha is the default fill opacity of the directed ribbon
	defaultChordRibbonAlpha uint8 = 170
	// labelCollisionTryCount is the max attempts to avoid label overlap
	labelCollisionTryCount = 5
)

// FlowMatrix describes a directed non-negative flow matrix.
// Data[i][j] means the flow amount from node Names[i] (source) to node Names[j] (target).
type FlowMatrix struct {
	// Names of the matrix nodes, the order is stable and is kept on the ring
	Names []string
	// Data is the square directed flow matrix, rows are sources and columns are targets
	Data [][]float64
}

// ChordOption customizes the directed chord chart.
type ChordOption struct {
	// The directed flow matrix
	Matrix FlowMatrix
	// Colors of the nodes, the theme series colors are used when it is empty
	NodeColors []Color
	// Gap angle (in radians) between node arcs, default is 1 degree
	Gap float64
	// RingWidth is the thickness of the node ring, default is 12px
	RingWidth float64
	// Radius of the outer ring, it can be a percent string ("40%") or pixel value ("120").
	// Default value is "40%" of the shorter canvas side.
	Radius string
	// RibbonAlpha is the fill opacity (0-255) of the directed ribbons, default is 170
	RibbonAlpha uint8
	// ShowLabel shows the node name labels, default is true
	ShowLabel *bool
	// ShowArrow draws an inward arrow head on every directed ribbon, default is false
	ShowArrow bool
}

type ChordChartOption struct {
	// The theme
	Theme ColorPalette
	// The font
	Font *truetype.Font
	// The padding of chord chart
	Padding Box
	// The option of title
	Title TitleOption
	// The legend option
	Legend LegendOption
	// The chord option
	Chord ChordOption
	// background is filled
	backgroundIsFilled bool
}

// chordNode is the layout result of a single node on the ring
type chordNode struct {
	name  string
	color Color
	// start/end angle (math convention: 0 at 3 o'clock, counterclockwise) of the whole arc
	start float64
	end   float64
	// total incident weight (out + in, self loop counted twice)
	total float64
	out   float64
	in    float64
	// current cursor for outgoing / incoming slots
	outCursor float64
	inCursor  float64
}

// mid returns the middle angle of the node arc
func (n *chordNode) mid() float64 {
	return (n.start + n.end) / 2
}

// chordEdge is one directed ribbon
type chordEdge struct {
	source      int
	target      int
	weight      float64
	srcStart    float64
	srcEnd      float64
	targetStart float64
	targetEnd   float64
	selfLoop    bool
	color       Color
}

type chordChart struct {
	p   *Painter
	opt *ChordChartOption
}

// NewChordChart returns a directed chord chart renderer
func NewChordChart(p *Painter, opt ChordChartOption) *chordChart {
	if opt.Theme == nil {
		opt.Theme = defaultTheme
	}
	return &chordChart{
		p:   p,
		opt: &opt,
	}
}

// validateAndBuild validates the flow matrix and builds node / edge layouts.
// The angle allocation is fixed: every node starts at -Pi/2 (12 o'clock) and
// nodes are placed counterclockwise in their input order, which guarantees a
// stable layout for the same matrix.
func (c *chordChart) validateAndBuild() ([]*chordNode, []*chordEdge, error) {
	opt := c.opt.Chord
	names := opt.Matrix.Names
	data := opt.Matrix.Data
	nodeCount := len(names)
	if nodeCount < 2 {
		return nil, nil, errors.New("The count of chord nodes should be >= 2")
	}
	if len(data) != nodeCount {
		return nil, nil, errors.New("The flow matrix row count should match the node count")
	}
	for _, row := range data {
		if len(row) != nodeCount {
			return nil, nil, errors.New("The flow matrix should be a square matrix")
		}
		for _, v := range row {
			// non-finite weights are rejected instead of corrupting the layout
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, nil, errors.New("The flow matrix contains non-finite weight")
			}
			if v < 0 {
				return nil, nil, errors.New("The flow matrix should contain non-negative weight only")
			}
		}
	}

	nodes := make([]*chordNode, nodeCount)
	totalWeight := float64(0)
	for i := 0; i < nodeCount; i++ {
		outSum, inSum := float64(0), float64(0)
		for j := 0; j < nodeCount; j++ {
			outSum += data[i][j]
			inSum += data[j][i]
		}
		// self loop participates in both sums, therefore counted twice,
		// which keeps the adjacency weight of A/B both equal to 3 for A->B=2, B->A=1
		nodes[i] = &chordNode{
			name:  names[i],
			out:   outSum,
			in:    inSum,
			total: outSum + inSum,
		}
		totalWeight += outSum + inSum
	}
	if totalWeight <= 0 {
		return nil, nil, errors.New("The total weight of flow matrix should be gt 0")
	}

	// resolve node colors
	for i := range nodes {
		if i < len(opt.NodeColors) && !opt.NodeColors[i].IsZero() {
			nodes[i].color = opt.NodeColors[i]
		} else {
			nodes[i].color = c.opt.Theme.GetSeriesColor(i)
		}
	}

	// fixed angle allocation
	gap := opt.Gap
	if gap <= 0 {
		gap = defaultChordGap
	}
	// the gaps should never consume the whole circle
	if gap*float64(nodeCount) >= 2*math.Pi {
		gap = (2 * math.Pi) / float64(nodeCount) * 0.1
	}
	usableAngle := 2*math.Pi - gap*float64(nodeCount)
	// every node keeps a minimal visible arc, so zero-row / fully disconnected
	// nodes still occupy a deterministic slot on the ring instead of vanishing
	minArc := usableAngle * 0.02
	weightedAngle := usableAngle - minArc*float64(nodeCount)
	if weightedAngle < 0 {
		minArc = usableAngle / float64(nodeCount) * 0.2
		weightedAngle = usableAngle - minArc*float64(nodeCount)
	}
	cursor := -math.Pi / 2
	for _, node := range nodes {
		span := minArc
		if totalWeight > 0 {
			span += weightedAngle * (node.total / totalWeight)
		}
		node.start = cursor
		node.end = cursor + span
		// outgoing slots occupy the first half of the arc, incoming slots the second half
		node.outCursor = node.start
		node.inCursor = node.start + span/2
		cursor += span + gap
	}

	// allocate directed edge slots in a fixed iteration order (source first,
	// then target) so reverse edges never merge and the layout is deterministic
	edges := make([]*chordEdge, 0)
	for i := 0; i < nodeCount; i++ {
		for j := 0; j < nodeCount; j++ {
			w := data[i][j]
			if w <= 0 {
				continue
			}
			src, target := nodes[i], nodes[j]
			srcSpan := (src.end - src.start) / 2 * (w / src.out)
			targetSpan := (target.end - target.start) / 2 * (w / target.in)
			edge := &chordEdge{
				source:      i,
				target:      j,
				weight:      w,
				selfLoop:    i == j,
				srcStart:    src.outCursor,
				srcEnd:      src.outCursor + srcSpan,
				targetStart: target.inCursor,
				targetEnd:   target.inCursor + targetSpan,
				color:       src.color,
			}
			src.outCursor += srcSpan
			target.inCursor += targetSpan
			edges = append(edges, edge)
		}
	}
	return nodes, edges, nil
}

// polarPoint converts an angle (math convention) and radius to a canvas point
func polarPoint(cx, cy int, radius, angle float64) Point {
	return Point{
		X: cx + int(math.Round(radius*math.Cos(angle))),
		Y: cy + int(math.Round(radius*math.Sin(angle))),
	}
}

// arcPoints samples an arc into polyline points
func arcPoints(cx, cy int, radius, startAngle, endAngle float64) []Point {
	delta := endAngle - startAngle
	if delta <= 0 {
		return []Point{polarPoint(cx, cy, radius, startAngle)}
	}
	// at least 2 segments, roughly one sample per 10 degrees
	count := int(math.Ceil(delta / (math.Pi / 18)))
	if count < 2 {
		count = 2
	}
	points := make([]Point, 0, count+1)
	for i := 0; i <= count; i++ {
		angle := startAngle + delta*float64(i)/float64(count)
		points = append(points, polarPoint(cx, cy, radius, angle))
	}
	return points
}

// ringSectorPath builds the closed annular sector path of a node
func ringSectorPath(p *Painter, cx, cy int, innerRadius, outerRadius float64, startAngle, endAngle float64) {
	outer := arcPoints(cx, cy, outerRadius, startAngle, endAngle)
	inner := arcPoints(cx, cy, innerRadius, endAngle, startAngle)
	p.MoveTo(outer[0].X, outer[0].Y)
	for _, point := range outer[1:] {
		p.LineTo(point.X, point.Y)
	}
	for _, point := range inner {
		p.LineTo(point.X, point.Y)
	}
	p.Close()
}

// ribbonPath builds the closed path of a directed ribbon.
// Both the source slot and the target slot stay on the inner ring, while the
// ribbon dips toward the center through quadratic bezier control points at
// the origin, which is the classic chord band shape. Because every endpoint
// is derived from the same radius and the fixed angle slots, arcs and ribbon
// endpoints remain aligned when the canvas is scaled.
func ribbonPath(p *Painter, cx, cy int, radius float64, e *chordEdge) {
	origin := Point{X: cx, Y: cy}
	if e.selfLoop {
		// self loop: a petal contained inside its own node, never crossing
		// the ribbons of other nodes
		midAngle := (e.srcStart + e.srcEnd) / 2
		edgeA := polarPoint(cx, cy, radius, e.srcStart)
		edgeB := polarPoint(cx, cy, radius, e.srcEnd)
		// the petal dips toward the center but stays within its own sector
		deepRadius := radius * 0.35
		ctrlA := polarPoint(cx, cy, deepRadius, midAngle-0.02)
		ctrlB := polarPoint(cx, cy, deepRadius, midAngle+0.02)
		tip := polarPoint(cx, cy, deepRadius*0.6, midAngle)
		p.MoveTo(edgeA.X, edgeA.Y)
		p.QuadCurveTo(ctrlA.X, ctrlA.Y, tip.X, tip.Y)
		p.QuadCurveTo(ctrlB.X, ctrlB.Y, edgeB.X, edgeB.Y)
		p.Close()
		return
	}
	srcStart := polarPoint(cx, cy, radius, e.srcStart)
	srcEnd := polarPoint(cx, cy, radius, e.srcEnd)
	targetStart := polarPoint(cx, cy, radius, e.targetStart)

	// side A: source start -> target start, control point at the origin
	p.MoveTo(srcStart.X, srcStart.Y)
	p.QuadCurveTo(origin.X, origin.Y, targetStart.X, targetStart.Y)
	// follow the target slot along the ring (target start -> target end)
	for _, point := range arcPoints(cx, cy, radius, e.targetStart, e.targetEnd)[1:] {
		p.LineTo(point.X, point.Y)
	}
	// side B: target end -> source end, control point at the origin
	p.QuadCurveTo(origin.X, origin.Y, srcEnd.X, srcEnd.Y)
	// close along the source slot in reverse (source end -> source start)
	for _, point := range arcPoints(cx, cy, radius, e.srcEnd, e.srcStart)[1:] {
		p.LineTo(point.X, point.Y)
	}
	p.Close()
}

// drawArrowHead draws a small inward arrow head at the target end of a ribbon
func (c *chordChart) drawArrowHead(p *Painter, cx, cy int, radius float64, e *chordEdge, color Color) {
	if e.selfLoop {
		return
	}
	angle := (e.targetStart + e.targetEnd) / 2
	// the arrow tip touches the target slot on the inner ring, the base
	// points inward toward the center
	tipRadius := radius * 0.99
	baseRadius := radius * 0.82
	halfWidth := math.Max(2.0, (e.targetEnd-e.targetStart)*radius*0.45)
	tip := polarPoint(cx, cy, tipRadius, angle)
	baseCenter := polarPoint(cx, cy, baseRadius, angle)
	dx := math.Cos(angle + math.Pi/2)
	dy := math.Sin(angle + math.Pi/2)
	p.OverrideDrawingStyle(Style{
		FillColor:   color,
		StrokeColor: color,
		StrokeWidth: 1,
	})
	p.MoveTo(tip.X, tip.Y)
	p.LineTo(baseCenter.X+int(dx*halfWidth), baseCenter.Y+int(dy*halfWidth))
	p.LineTo(baseCenter.X-int(dx*halfWidth), baseCenter.Y-int(dy*halfWidth))
	p.Close()
	p.FillStroke()
}

// drawLabels renders the node labels with collision avoidance
func (c *chordChart) drawLabels(p *Painter, cx, cy int, outerRadius float64, nodes []*chordNode) {
	opt := c.opt
	p.OverrideTextStyle(Style{
		FontColor: opt.Theme.GetTextColor(),
		FontSize:  labelFontSize,
		Font:      opt.Font,
	})
	labelLineWidth := 12
	if outerRadius < 50 {
		labelLineWidth = 8
	}
	labelRadius := outerRadius + float64(labelLineWidth)
	prevPoints := make([]Point, 0)
	isOverride := func(x, y int) bool {
		for _, prev := range prevPoints {
			if math.Abs(float64(prev.Y-y)) > labelFontSize {
				continue
			}
			if (prev.X <= cx && x <= cx) || (prev.X > cx && x > cx) {
				return true
			}
		}
		return false
	}
	for _, node := range nodes {
		angle := node.mid()
		start := polarPoint(cx, cy, outerRadius, angle)
		end := polarPoint(cx, cy, labelRadius, angle)
		// fixed amount of collision avoidance attempts, then give up shifting
		for i := 0; i < labelCollisionTryCount; i++ {
			if !isOverride(end.X, end.Y) {
				break
			}
			end.Y -= labelFontSize << 1
		}
		prevPoints = append(prevPoints, end)

		p.OverrideDrawingStyle(Style{
			StrokeColor: node.color,
			StrokeWidth: 1,
		})
		p.MoveTo(start.X, start.Y)
		p.LineTo(end.X, end.Y)
		offset := labelLineWidth
		if end.X < cx {
			offset = -labelLineWidth
		}
		p.MoveTo(end.X, end.Y)
		lineEndX := end.X + offset
		p.LineTo(lineEndX, end.Y)
		p.Stroke()

		textBox := p.MeasureText(node.name)
		textMargin := 3
		x := lineEndX + textMargin
		y := end.Y + textBox.Height()>>1 - 1
		if offset < 0 {
			x = lineEndX - textBox.Width() - textMargin
		}
		p.Text(node.name, x, y)
	}
}

func (c *chordChart) render(result *defaultRenderResult, seriesList SeriesList) (Box, error) {
	p := result.seriesPainter
	opt := c.opt

	nodes, edges, err := c.validateAndBuild()
	if err != nil {
		return BoxZero, err
	}

	width := p.Width()
	height := p.Height()
	cx := width >> 1
	cy := height >> 1
	diameter := float64(minInt(width, height))
	outerRadius := getRadius(diameter, opt.Chord.Radius)
	if outerRadius <= 0 {
		outerRadius = diameter * defaultChordRadiusPercent
	}
	// reserve room for the label leader lines and text outside the ring
	if showChordLabel(opt.Chord.ShowLabel) {
		outerRadius -= float64(labelLineWidthFor(outerRadius)) + 22
	}
	if outerRadius < 10 {
		outerRadius = 10
	}
	ringWidth := opt.Chord.RingWidth
	if ringWidth <= 0 {
		ringWidth = defaultChordRingWidth
	}
	if ringWidth >= outerRadius {
		ringWidth = outerRadius * 0.25
	}
	innerRadius := outerRadius - ringWidth

	ribbonAlpha := opt.Chord.RibbonAlpha
	if ribbonAlpha == 0 {
		ribbonAlpha = defaultChordRibbonAlpha
	}

	// ribbons first, node rings cover their endpoints afterwards
	for _, edge := range edges {
		fillColor := edge.color.WithAlpha(ribbonAlpha)
		p.OverrideDrawingStyle(Style{
			FillColor:   fillColor,
			StrokeColor: edge.color.WithAlpha(ribbonAlpha),
			StrokeWidth: 1,
		})
		ribbonPath(p, cx, cy, innerRadius, edge)
		p.FillStroke()
		if opt.Chord.ShowArrow {
			c.drawArrowHead(p, cx, cy, innerRadius, edge, edge.color)
		}
	}

	// node annular sectors
	for _, node := range nodes {
		p.OverrideDrawingStyle(Style{
			FillColor:   node.color,
			StrokeColor: node.color,
			StrokeWidth: 1,
		})
		ringSectorPath(p, cx, cy, innerRadius, outerRadius, node.start, node.end)
		p.FillStroke()
	}

	if showChordLabel(opt.Chord.ShowLabel) {
		c.drawLabels(p, cx, cy, outerRadius, nodes)
	}
	return c.p.box, nil
}

func showChordLabel(show *bool) bool {
	return show == nil || *show
}

func labelLineWidthFor(radius float64) int {
	if radius < 50 {
		return 8
	}
	return 12
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Render renders the directed chord chart
func (c *chordChart) Render() (Box, error) {
	p := c.p
	opt := c.opt
	renderResult, err := defaultRender(p, defaultRenderOption{
		Theme:      opt.Theme,
		Padding:    opt.Padding,
		SeriesList: NewChordSeriesList(opt.Chord.Matrix),
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
	seriesList := NewChordSeriesList(opt.Chord.Matrix)
	return c.render(renderResult, seriesList)
}

// NewChordSeriesList wraps a flow matrix into a marker series list so that the
// generic Render pipeline recognizes the chord chart.
func NewChordSeriesList(matrix FlowMatrix) SeriesList {
	return SeriesList{
		{
			Type: ChartTypeChord,
			Name: "chord",
			Data: []SeriesData{
				{
					Value: 1,
				},
			},
		},
	}
}

// ChordRender renders a directed chord chart from a named flow matrix
func ChordRender(matrix FlowMatrix, opts ...OptionFunc) (*Painter, error) {
	legendData := make([]string, len(matrix.Names))
	copy(legendData, matrix.Names)
	opt := ChartOption{
		SeriesList: NewChordSeriesList(matrix),
		Chord: ChordOption{
			Matrix: matrix,
		},
		Legend: LegendOption{
			Data: legendData,
		},
	}
	return Render(opt, opts...)
}
