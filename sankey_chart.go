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
	"fmt"
	"sort"

	"github.com/golang/freetype/truetype"
)

const (
	defaultSankeyNodeWidth = 20
	defaultSankeyNodeGap   = 10
	// the alpha of link fill color
	sankeyLinkColorAlpha = 90
)

// SankeyNode is the node of sankey chart
type SankeyNode struct {
	// The unique name of node
	Name string
	// The layer(depth) of node, nodes of the same layer are aligned vertically.
	// If the value is less than 0, the layer is calculated automatically
	// by the longest path from the source nodes.
	Layer int
	// The color of node, if not set, the theme series color is used
	Color Color
}

// SankeyLink is the directed link of sankey chart
type SankeyLink struct {
	// The name of source node
	Source string
	// The name of target node
	Target string
	// The weight of link, it should be greater than or equal to 0
	Value float64
	// The color of link, if not set, the color of source node(with alpha) is used
	Color Color
}

// SankeyChartOption is the option of sankey chart
type SankeyChartOption struct {
	// The theme
	Theme ColorPalette
	// The font size
	Font *truetype.Font
	// The nodes of sankey chart
	Nodes []SankeyNode
	// The directed links of sankey chart
	Links []SankeyLink
	// The padding of sankey chart
	Padding Box
	// The option of title
	Title TitleOption
	// The width of node, default value is 20
	NodeWidth int
	// The vertical gap between nodes of the same layer, default value is 10
	NodeGap int
	// The show flag of node label, default is true
	LabelShow *bool
	// The font size of node label, default value is 10
	LabelFontSize float64
}

type sankeyChart struct {
	p   *Painter
	opt *SankeyChartOption
}

// NewSankeyChart returns a sankey chart renderer
func NewSankeyChart(p *Painter, opt SankeyChartOption) *sankeyChart {
	if opt.Theme == nil {
		opt.Theme = defaultTheme
	}
	return &sankeyChart{
		p:   p,
		opt: &opt,
	}
}

// sankeyLayoutNode is the node with calculated layout geometry
type sankeyLayoutNode struct {
	SankeyNode
	// the index of node in the option
	index int
	// the total flow value of node(max of in and out flow)
	value float64
	// the geometry of node
	x0 int
	y0 int
	x1 int
	y1 int
	// the vertical offset already used by out links
	outOffset int
	// the vertical offset already used by in links
	inOffset int
}

// sankeyLayoutLink is the link with calculated layout geometry
type sankeyLayoutLink struct {
	SankeyLink
	// the geometry of link band
	sy0 int
	sy1 int
	ty0 int
	ty1 int
}

type sankeyLayout struct {
	nodes   []*sankeyLayoutNode
	links   []*sankeyLayoutLink
	layers  [][]*sankeyLayoutNode
	scale   float64
	nodeMap map[string]*sankeyLayoutNode
}

// valueOf returns the total flow value of the node with the specified name
func (l *sankeyLayout) valueOf(name string) float64 {
	if n, ok := l.nodeMap[name]; ok {
		return n.value
	}
	return 0
}

// layoutSankey calculates the geometry of nodes and links
func layoutSankey(opt *SankeyChartOption, width, height int) (*sankeyLayout, error) {
	if len(opt.Nodes) == 0 {
		return nil, errors.New("sankey: nodes can not be empty")
	}
	nodeWidth := opt.NodeWidth
	if nodeWidth <= 0 {
		nodeWidth = defaultSankeyNodeWidth
	}
	nodeGap := opt.NodeGap
	if nodeGap <= 0 {
		nodeGap = defaultSankeyNodeGap
	}

	nodes := make([]*sankeyLayoutNode, len(opt.Nodes))
	nodeMap := make(map[string]*sankeyLayoutNode)
	for index, item := range opt.Nodes {
		if item.Name == "" {
			return nil, errors.New("sankey: node name can not be empty")
		}
		if _, ok := nodeMap[item.Name]; ok {
			return nil, fmt.Errorf("sankey: duplicate node %q", item.Name)
		}
		n := &sankeyLayoutNode{
			SankeyNode: item,
			index:      index,
		}
		nodes[index] = n
		nodeMap[item.Name] = n
	}

	inFlow := make(map[string]float64)
	outFlow := make(map[string]float64)
	for _, link := range opt.Links {
		if link.Value < 0 {
			return nil, fmt.Errorf("sankey: link %q -> %q has negative value", link.Source, link.Target)
		}
		if _, ok := nodeMap[link.Source]; !ok {
			return nil, fmt.Errorf("sankey: link source node %q not found", link.Source)
		}
		if _, ok := nodeMap[link.Target]; !ok {
			return nil, fmt.Errorf("sankey: link target node %q not found", link.Target)
		}
		outFlow[link.Source] += link.Value
		inFlow[link.Target] += link.Value
	}
	for _, n := range nodes {
		n.value = outFlow[n.Name]
		if inFlow[n.Name] > n.value {
			n.value = inFlow[n.Name]
		}
	}

	// assign layers
	autoLayer := false
	for _, n := range nodes {
		if n.Layer < 0 {
			autoLayer = true
			break
		}
	}
	if autoLayer {
		// the layer is the longest path from the source nodes
		layerOf := make(map[string]int)
		visiting := make(map[string]bool)
		var resolve func(name string) (int, error)
		resolve = func(name string) (int, error) {
			if layer, ok := layerOf[name]; ok {
				return layer, nil
			}
			if visiting[name] {
				return 0, fmt.Errorf("sankey: cycle detected at node %q", name)
			}
			visiting[name] = true
			layer := 0
			for _, link := range opt.Links {
				if link.Target != name {
					continue
				}
				l, err := resolve(link.Source)
				if err != nil {
					return 0, err
				}
				if l+1 > layer {
					layer = l + 1
				}
			}
			visiting[name] = false
			layerOf[name] = layer
			return layer, nil
		}
		for _, n := range nodes {
			layer, err := resolve(n.Name)
			if err != nil {
				return nil, err
			}
			n.Layer = layer
		}
	}
	// a link must flow from a lower layer to a higher layer
	for _, link := range opt.Links {
		if nodeMap[link.Source].Layer >= nodeMap[link.Target].Layer {
			return nil, fmt.Errorf("sankey: layer conflict, link %q(layer %d) -> %q(layer %d)",
				link.Source, nodeMap[link.Source].Layer,
				link.Target, nodeMap[link.Target].Layer)
		}
	}

	// group nodes by layer, the order in the same layer follows the input order
	maxLayer := 0
	for _, n := range nodes {
		if n.Layer > maxLayer {
			maxLayer = n.Layer
		}
	}
	layers := make([][]*sankeyLayoutNode, maxLayer+1)
	for _, n := range nodes {
		layers[n.Layer] = append(layers[n.Layer], n)
	}

	// the scale is limited by the tallest layer,
	// so that the links on both sides of a node never exceed its capacity
	scale := float64(0)
	for _, layerNodes := range layers {
		total := float64(0)
		for _, n := range layerNodes {
			total += n.value
		}
		if total == 0 {
			continue
		}
		available := float64(height - nodeGap*(len(layerNodes)-1))
		if available < 0 {
			available = 0
		}
		s := available / total
		if scale == 0 || s < scale {
			scale = s
		}
	}

	// the x position of each layer
	layerX := func(layer int) int {
		if maxLayer == 0 {
			return 0
		}
		return layer * (width - nodeWidth) / maxLayer
	}
	// the y position of nodes, nodes of the same layer are stacked and centered
	for layer, layerNodes := range layers {
		total := float64(0)
		for _, n := range layerNodes {
			total += n.value
		}
		used := int(total*scale) + nodeGap*(len(layerNodes)-1)
		y := (height - used) / 2
		if y < 0 {
			y = 0
		}
		x := layerX(layer)
		for _, n := range layerNodes {
			n.x0 = x
			n.x1 = x + nodeWidth
			n.y0 = y
			n.y1 = y + int(n.value*scale)
			y = n.y1 + nodeGap
		}
	}

	links := make([]*sankeyLayoutLink, len(opt.Links))
	for index, link := range opt.Links {
		links[index] = &sankeyLayoutLink{
			SankeyLink: link,
		}
	}
	// the position of node in its layer, used to sort links for less crossing
	positionOf := make(map[string]int)
	for _, layerNodes := range layers {
		for index, n := range layerNodes {
			positionOf[n.Name] = index
		}
	}
	// assign the slots of links on the source side,
	// links of the same source node are sorted by the target position
	sourceSide := make([]*sankeyLayoutLink, len(links))
	copy(sourceSide, links)
	sort.SliceStable(sourceSide, func(i, j int) bool {
		a, b := sourceSide[i], sourceSide[j]
		if a.Source != b.Source {
			return positionOf[a.Source] < positionOf[b.Source]
		}
		return positionOf[a.Target] < positionOf[b.Target]
	})
	for _, link := range sourceSide {
		source := nodeMap[link.Source]
		thickness := int(link.Value * scale)
		link.sy0 = source.y0 + source.outOffset
		link.sy1 = link.sy0 + thickness
		source.outOffset += thickness
	}
	// assign the slots of links on the target side,
	// links of the same target node are sorted by the source position
	targetSide := make([]*sankeyLayoutLink, len(links))
	copy(targetSide, links)
	sort.SliceStable(targetSide, func(i, j int) bool {
		a, b := targetSide[i], targetSide[j]
		if a.Target != b.Target {
			return positionOf[a.Target] < positionOf[b.Target]
		}
		return positionOf[a.Source] < positionOf[b.Source]
	})
	for _, link := range targetSide {
		target := nodeMap[link.Target]
		thickness := int(link.Value * scale)
		link.ty0 = target.y0 + target.inOffset
		link.ty1 = link.ty0 + thickness
		target.inOffset += thickness
	}

	return &sankeyLayout{
		nodes:   nodes,
		links:   links,
		layers:  layers,
		scale:   scale,
		nodeMap: nodeMap,
	}, nil
}

func (s *sankeyChart) render(result *defaultRenderResult) (Box, error) {
	opt := s.opt
	seriesPainter := result.seriesPainter
	width := seriesPainter.Width()
	height := seriesPainter.Height()
	layout, err := layoutSankey(opt, width, height)
	if err != nil {
		return BoxZero, err
	}
	theme := opt.Theme

	// the color of node
	nodeColor := func(n *sankeyLayoutNode) Color {
		if !n.Color.IsZero() {
			return n.Color
		}
		return theme.GetSeriesColor(n.index)
	}

	// draw the link bands first, so that the nodes cover the ends of bands
	for _, link := range layout.links {
		// zero flow link is ignored
		if link.sy1 <= link.sy0 && link.ty1 <= link.ty0 {
			continue
		}
		source := layout.nodeMap[link.Source]
		target := layout.nodeMap[link.Target]
		color := link.Color
		if color.IsZero() {
			color = nodeColor(source).WithAlpha(sankeyLinkColorAlpha)
		}
		x0 := source.x1
		x1 := target.x0
		cx := (x0 + x1) >> 1
		seriesPainter.OverrideDrawingStyle(Style{
			FillColor: color,
		})
		seriesPainter.MoveTo(x0, link.sy0)
		seriesPainter.QuadCurveTo(cx, link.sy0, x1, link.ty0)
		seriesPainter.LineTo(x1, link.ty1)
		seriesPainter.QuadCurveTo(cx, link.ty1, x0, link.sy1)
		seriesPainter.Close()
		seriesPainter.Fill()
	}

	// draw the nodes
	for _, n := range layout.nodes {
		seriesPainter.OverrideDrawingStyle(Style{
			FillColor: nodeColor(n),
		})
		y1 := n.y1
		if y1 == n.y0 {
			// the node with zero flow is drawn as a thin bar
			y1 = n.y0 + 1
		}
		seriesPainter.MoveTo(n.x0, n.y0)
		seriesPainter.LineTo(n.x1, n.y0)
		seriesPainter.LineTo(n.x1, y1)
		seriesPainter.LineTo(n.x0, y1)
		seriesPainter.Close()
		seriesPainter.Fill()
	}

	// draw the labels of nodes
	if isFalse(opt.LabelShow) {
		return s.p.box, nil
	}
	fontSize := opt.LabelFontSize
	if fontSize == 0 {
		fontSize = labelFontSize
	}
	seriesPainter.OverrideTextStyle(Style{
		FontColor: theme.GetTextColor(),
		FontSize:  fontSize,
		Font:      opt.Font,
	})
	for _, n := range layout.nodes {
		text := n.Name
		textBox := seriesPainter.MeasureText(text)
		textY := (n.y0+n.y1)>>1 + textBox.Height()>>1
		// the label is placed on the right side of node if there is enough space,
		// otherwise it is placed on the left side
		textX := n.x1 + 4
		if textX+textBox.Width() > width {
			textX = n.x0 - 4 - textBox.Width()
			if textX < 0 {
				textX = 0
			}
		}
		seriesPainter.Text(text, textX, textY)
	}

	return s.p.box, nil
}

// Render renders the sankey chart
func (s *sankeyChart) Render() (Box, error) {
	p := s.p
	opt := s.opt
	renderResult, err := defaultRender(p, defaultRenderOption{
		Theme:   opt.Theme,
		Padding: opt.Padding,
		XAxis: XAxisOption{
			Show: FalseFlag(),
		},
		YAxisOptions: []YAxisOption{
			{
				Show: FalseFlag(),
			},
		},
		TitleOption: opt.Title,
	})
	if err != nil {
		return BoxZero, err
	}
	return s.render(renderResult)
}
