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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func sankeyTestOption() SankeyChartOption {
	return SankeyChartOption{
		Nodes: []SankeyNode{
			{Name: "A", Layer: -1},
			{Name: "B", Layer: -1},
			{Name: "C", Layer: -1},
			{Name: "D", Layer: -1},
		},
		Links: []SankeyLink{
			{Source: "A", Target: "B", Value: 3},
			{Source: "A", Target: "C", Value: 1},
			{Source: "B", Target: "D", Value: 3},
			{Source: "C", Target: "D", Value: 1},
		},
	}
}

func TestSankeyLayoutFlowValue(t *testing.T) {
	assert := assert.New(t)
	opt := sankeyTestOption()
	layout, err := layoutSankey(&opt, 600, 400)
	assert.Nil(err)
	// A -> B(3), A -> C(1), B -> D(3), C -> D(1)
	// the total flow of A and D should both be 4
	assert.Equal(float64(4), layout.valueOf("A"))
	assert.Equal(float64(3), layout.valueOf("B"))
	assert.Equal(float64(1), layout.valueOf("C"))
	assert.Equal(float64(4), layout.valueOf("D"))
}

func TestSankeyLayoutGeometry(t *testing.T) {
	assert := assert.New(t)
	opt := sankeyTestOption()
	layout, err := layoutSankey(&opt, 600, 400)
	assert.Nil(err)

	// the tallest layer limits the scale: (400 - 10) / 4 = 97.5
	assert.Equal(97.5, layout.scale)

	// nodes are assigned to layers automatically
	nodeOf := func(name string) *sankeyLayoutNode {
		return layout.nodeMap[name]
	}
	assert.Equal(0, nodeOf("A").Layer)
	assert.Equal(1, nodeOf("B").Layer)
	assert.Equal(1, nodeOf("C").Layer)
	assert.Equal(2, nodeOf("D").Layer)

	// fixed node geometry: node width 20, gap 10
	// layer x positions: 0, 290, 580
	assert.Equal([]int{0, 5, 20, 395}, []int{nodeOf("A").x0, nodeOf("A").y0, nodeOf("A").x1, nodeOf("A").y1})
	assert.Equal([]int{290, 0, 310, 292}, []int{nodeOf("B").x0, nodeOf("B").y0, nodeOf("B").x1, nodeOf("B").y1})
	assert.Equal([]int{290, 302, 310, 399}, []int{nodeOf("C").x0, nodeOf("C").y0, nodeOf("C").x1, nodeOf("C").y1})
	assert.Equal([]int{580, 5, 600, 395}, []int{nodeOf("D").x0, nodeOf("D").y0, nodeOf("D").x1, nodeOf("D").y1})

	// fixed link band geometry
	linkOf := func(source, target string) *sankeyLayoutLink {
		for _, link := range layout.links {
			if link.Source == source && link.Target == target {
				return link
			}
		}
		return nil
	}
	ab := linkOf("A", "B")
	assert.Equal([]int{5, 297, 0, 292}, []int{ab.sy0, ab.sy1, ab.ty0, ab.ty1})
	ac := linkOf("A", "C")
	assert.Equal([]int{297, 394, 302, 399}, []int{ac.sy0, ac.sy1, ac.ty0, ac.ty1})
	bd := linkOf("B", "D")
	assert.Equal([]int{0, 292, 5, 297}, []int{bd.sy0, bd.sy1, bd.ty0, bd.ty1})
	cd := linkOf("C", "D")
	assert.Equal([]int{302, 399, 297, 394}, []int{cd.sy0, cd.sy1, cd.ty0, cd.ty1})

	// the total width of links on both sides of a node can not exceed its capacity
	for _, n := range layout.nodes {
		capacity := n.y1 - n.y0
		assert.True(n.outOffset <= capacity, "node %s out flow %d exceeds capacity %d", n.Name, n.outOffset, capacity)
		assert.True(n.inOffset <= capacity, "node %s in flow %d exceeds capacity %d", n.Name, n.inOffset, capacity)
	}
}

func TestSankeyLayoutExplicitLayer(t *testing.T) {
	assert := assert.New(t)
	opt := sankeyTestOption()
	// the explicit layers are the same as the auto calculated layers,
	// so the geometry should be identical
	opt.Nodes[0].Layer = 0
	opt.Nodes[1].Layer = 1
	opt.Nodes[2].Layer = 1
	opt.Nodes[3].Layer = 2
	layout, err := layoutSankey(&opt, 600, 400)
	assert.Nil(err)
	assert.Equal(97.5, layout.scale)
	assert.Equal(5, layout.nodeMap["A"].y0)
	assert.Equal(395, layout.nodeMap["D"].y1)
}

func TestSankeyLayoutError(t *testing.T) {
	assert := assert.New(t)

	// empty nodes
	opt := sankeyTestOption()
	opt.Nodes = nil
	_, err := layoutSankey(&opt, 600, 400)
	assert.Equal("sankey: nodes can not be empty", err.Error())

	// duplicate node
	opt = sankeyTestOption()
	opt.Nodes = append(opt.Nodes, SankeyNode{Name: "A"})
	_, err = layoutSankey(&opt, 600, 400)
	assert.Equal(`sankey: duplicate node "A"`, err.Error())

	// negative link value
	opt = sankeyTestOption()
	opt.Links[0].Value = -1
	_, err = layoutSankey(&opt, 600, 400)
	assert.Equal(`sankey: link "A" -> "B" has negative value`, err.Error())

	// missing source node
	opt = sankeyTestOption()
	opt.Links[0].Source = "X"
	_, err = layoutSankey(&opt, 600, 400)
	assert.Equal(`sankey: link source node "X" not found`, err.Error())

	// missing target node
	opt = sankeyTestOption()
	opt.Links[0].Target = "X"
	_, err = layoutSankey(&opt, 600, 400)
	assert.Equal(`sankey: link target node "X" not found`, err.Error())

	// cycle
	opt = sankeyTestOption()
	opt.Links = append(opt.Links, SankeyLink{Source: "D", Target: "A", Value: 1})
	_, err = layoutSankey(&opt, 600, 400)
	assert.Contains(err.Error(), "cycle detected")

	// layer conflict: the source layer is greater than the target layer
	opt = sankeyTestOption()
	opt.Nodes[0].Layer = 1
	opt.Nodes[1].Layer = 0
	opt.Nodes[2].Layer = 1
	opt.Nodes[3].Layer = 2
	_, err = layoutSankey(&opt, 600, 400)
	assert.Contains(err.Error(), "layer conflict")
}

func TestSankeyLayoutZeroFlow(t *testing.T) {
	assert := assert.New(t)
	opt := SankeyChartOption{
		Nodes: []SankeyNode{
			{Name: "A", Layer: -1},
			{Name: "B", Layer: -1},
			{Name: "C", Layer: -1},
		},
		Links: []SankeyLink{
			{Source: "A", Target: "B", Value: 2},
			// zero flow link
			{Source: "A", Target: "C", Value: 0},
		},
	}
	layout, err := layoutSankey(&opt, 600, 400)
	assert.Nil(err)
	assert.Equal(float64(2), layout.valueOf("A"))
	assert.Equal(float64(2), layout.valueOf("B"))
	assert.Equal(float64(0), layout.valueOf("C"))
	// the zero flow link takes no space
	link := layout.links[1]
	assert.Equal(link.sy0, link.sy1)
	assert.Equal(link.ty0, link.ty1)
	// the node with zero flow is still laid out
	assert.Equal(layout.nodeMap["C"].y0, layout.nodeMap["C"].y1)
}

func TestSankeyChartRender(t *testing.T) {
	assert := assert.New(t)

	opt := sankeyTestOption()
	opt.Title = TitleOption{
		Text: "Sankey",
	}

	// svg output
	p, err := NewPainter(PainterOptions{
		Type:   ChartOutputSVG,
		Width:  600,
		Height: 400,
	}, PainterThemeOption(defaultTheme))
	assert.Nil(err)
	_, err = NewSankeyChart(p, opt).Render()
	assert.Nil(err)
	svgData, err := p.Bytes()
	assert.Nil(err)
	assert.True(strings.HasPrefix(string(svgData), "<svg"))
	// background + 4 links + 4 nodes
	assert.Equal(9, strings.Count(string(svgData), "<path"))
	// title + 4 node labels
	assert.Equal(5, strings.Count(string(svgData), "<text"))

	// png output
	p, err = NewPainter(PainterOptions{
		Type:   ChartOutputPNG,
		Width:  600,
		Height: 400,
	}, PainterThemeOption(defaultTheme))
	assert.Nil(err)
	_, err = NewSankeyChart(p, opt).Render()
	assert.Nil(err)
	pngData, err := p.Bytes()
	assert.Nil(err)
	assert.True(len(pngData) > 0)
	assert.Equal([]byte{0x89, 0x50, 0x4e, 0x47}, pngData[:4])

	// the same option renders to both output types without error,
	// custom node color and link color are supported
	opt.Nodes[0].Color = Color{R: 255, A: 255}
	opt.Links[0].Color = Color{G: 255, A: 255}
	for _, outputType := range []string{ChartOutputSVG, ChartOutputPNG} {
		p, err = NewPainter(PainterOptions{
			Type:   outputType,
			Width:  600,
			Height: 400,
		}, PainterThemeOption(defaultTheme))
		assert.Nil(err)
		_, err = NewSankeyChart(p, opt).Render()
		assert.Nil(err)
		data, err := p.Bytes()
		assert.Nil(err)
		assert.True(len(data) > 0)
	}

	// render error is returned for invalid option
	p, err = NewPainter(PainterOptions{
		Type:   ChartOutputSVG,
		Width:  600,
		Height: 400,
	}, PainterThemeOption(defaultTheme))
	assert.Nil(err)
	invalidOpt := sankeyTestOption()
	invalidOpt.Links[0].Target = "not-exists"
	_, err = NewSankeyChart(p, invalidOpt).Render()
	assert.NotNil(err)
}
