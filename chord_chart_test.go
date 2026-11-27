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
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChordLayoutValidation(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name    string
		matrix  FlowMatrix
		wantErr string
	}{
		{
			name: "node count lt 2",
			matrix: FlowMatrix{
				Names: []string{"A"},
				Data:  [][]float64{{0}},
			},
			wantErr: ">= 2",
		},
		{
			name: "row count mismatch",
			matrix: FlowMatrix{
				Names: []string{"A", "B"},
				Data:  [][]float64{{0, 0}},
			},
			wantErr: "row count",
		},
		{
			name: "not square",
			matrix: FlowMatrix{
				Names: []string{"A", "B"},
				Data: [][]float64{
					{0, 0, 0},
					{0, 0, 0},
				},
			},
			wantErr: "square",
		},
		{
			name: "negative weight",
			matrix: FlowMatrix{
				Names: []string{"A", "B"},
				Data: [][]float64{
					{0, -1},
					{0, 0},
				},
			},
			wantErr: "non-negative",
		},
		{
			name: "nan weight",
			matrix: FlowMatrix{
				Names: []string{"A", "B"},
				Data: [][]float64{
					{0, math.NaN()},
					{0, 0},
				},
			},
			wantErr: "non-finite",
		},
		{
			name: "infinite weight",
			matrix: FlowMatrix{
				Names: []string{"A", "B"},
				Data: [][]float64{
					{0, math.Inf(1)},
					{0, 0},
				},
			},
			wantErr: "non-finite",
		},
		{
			name: "all zero",
			matrix: FlowMatrix{
				Names: []string{"A", "B"},
				Data: [][]float64{
					{0, 0},
					{0, 0},
				},
			},
			wantErr: "gt 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewPainter(PainterOptions{
				Type:   ChartOutputSVG,
				Width:  600,
				Height: 400,
			}, PainterThemeOption(defaultTheme))
			assert.Nil(err)
			_, err = NewChordChart(p, ChordChartOption{
				Chord: ChordOption{
					Matrix: tt.matrix,
				},
			}).Render()
			assert.NotNil(err)
			assert.Contains(err.Error(), tt.wantErr)
		})
	}
}

func TestChordLayoutDirectionAndTotals(t *testing.T) {
	assert := assert.New(t)

	// A->B = 2, B->A = 1: the two reverse edges must stay distinct,
	// and both nodes have a total adjacency weight of 3.
	matrix := FlowMatrix{
		Names: []string{"A", "B"},
		Data: [][]float64{
			{0, 2},
			{1, 0},
		},
	}
	p, err := NewPainter(PainterOptions{
		Type:   ChartOutputSVG,
		Width:  600,
		Height: 400,
	}, PainterThemeOption(defaultTheme))
	assert.Nil(err)

	c := NewChordChart(p, ChordChartOption{
		Chord: ChordOption{
			Matrix: matrix,
		},
	})
	nodes, edges, err := c.validateAndBuild()
	assert.Nil(err)

	// distinct reverse edges, never merged into one
	assert.Equal(2, len(edges))
	assert.Equal(0, edges[0].source)
	assert.Equal(1, edges[0].target)
	assert.Equal(2.0, edges[0].weight)
	assert.Equal(1, edges[1].source)
	assert.Equal(0, edges[1].target)
	assert.Equal(1.0, edges[1].weight)

	// total adjacency weight of each node is 3 (out + in)
	assert.Equal(3.0, nodes[0].total)
	assert.Equal(3.0, nodes[1].total)
	assert.Equal(2.0, nodes[0].out)
	assert.Equal(1.0, nodes[0].in)
	assert.Equal(1.0, nodes[1].out)
	assert.Equal(2.0, nodes[1].in)

	// the source slot of each edge is inside the source node's outgoing half,
	// and the target slot is inside the target node's incoming half
	assert.True(edges[0].srcStart >= nodes[0].start)
	assert.True(edges[0].srcEnd <= nodes[0].mid())
	assert.True(edges[0].targetStart >= nodes[1].mid())
	assert.True(edges[0].targetEnd <= nodes[1].end)

	// fixed angle allocation: first node always starts at 12 o'clock
	assert.InDelta(-math.Pi/2, nodes[0].start, 1e-9)
	// nodes are placed in input order
	assert.Equal("A", nodes[0].name)
	assert.Equal("B", nodes[1].name)

	// the two reverse ribbons must be emitted as two distinct ribbon paths
	// (fill uses the ribbon alpha, while node rings use full opacity)
	painter, err := NewPainter(PainterOptions{
		Type:   ChartOutputSVG,
		Width:  500,
		Height: 500,
	}, PainterThemeOption(defaultTheme))
	assert.Nil(err)
	_, err = NewChordChart(painter, ChordChartOption{
		Chord: ChordOption{
			Matrix: matrix,
		},
	}).Render()
	assert.Nil(err)
	svgBytes, err := painter.Bytes()
	assert.Nil(err)
	svg := string(svgBytes)
	// the two ribbons use the translucent ribbon alpha (rendered as 0.7),
	// while node rings use full opacity (1.0); each reverse edge keeps its
	// source color, so two distinct translucent ribbon fills must exist
	assert.Equal(1, strings.Count(svg, "fill:rgba(84,112,198,0.7)"))
	assert.Equal(1, strings.Count(svg, "fill:rgba(145,204,117,0.7)"))
	// quadratic bezier ribbon paths are present (not merged into one shape)
	assert.True(strings.Contains(svg, "Q"))
}

func TestChordLayoutSpecialCases(t *testing.T) {
	assert := assert.New(t)

	t.Run("self loop", func(t *testing.T) {
		matrix := FlowMatrix{
			Names: []string{"A", "B"},
			Data: [][]float64{
				{3, 0},
				{0, 2},
			},
		}
		p, _ := NewPainter(PainterOptions{
			Type:   ChartOutputPNG,
			Width:  400,
			Height: 400,
		})
		c := NewChordChart(p, ChordChartOption{
			Chord: ChordOption{
				Matrix: matrix,
			},
		})
		nodes, edges, err := c.validateAndBuild()
		assert.Nil(err)
		// self loop contributes twice to the node total
		assert.Equal(6.0, nodes[0].total)
		assert.Equal(4.0, nodes[1].total)
		assert.Equal(2, len(edges))
		assert.True(edges[0].selfLoop)
		assert.True(edges[1].selfLoop)
	})

	t.Run("one way edge and zero row", func(t *testing.T) {
		matrix := FlowMatrix{
			Names: []string{"A", "B", "C"},
			Data: [][]float64{
				{0, 5, 0},
				{0, 0, 0},
				{0, 0, 0},
			},
		}
		p, _ := NewPainter(PainterOptions{
			Type:   ChartOutputSVG,
			Width:  500,
			Height: 500,
		})
		c := NewChordChart(p, ChordChartOption{
			Chord: ChordOption{
				Matrix: matrix,
			},
		})
		nodes, edges, err := c.validateAndBuild()
		assert.Nil(err)
		// only one directed edge
		assert.Equal(1, len(edges))
		assert.False(edges[0].selfLoop)
		// zero-row / zero-column nodes still keep a deterministic minimal arc
		for _, node := range nodes {
			assert.True(node.end > node.start)
		}
		// B has no outgoing flow, its outgoing cursor is never advanced
		assert.InDelta(nodes[1].start, nodes[1].outCursor, 1e-9)
		// B carries the incoming flow in its incoming half
		assert.True(nodes[1].inCursor > nodes[1].mid())
		// C has no flow at all but remains visible on the ring
		assert.Equal(0.0, nodes[2].total)
		assert.True(nodes[2].end-nodes[2].start > 0)
	})

	t.Run("asymmetric matrix", func(t *testing.T) {
		matrix := FlowMatrix{
			Names: []string{"A", "B", "C"},
			Data: [][]float64{
				{0, 1, 4},
				{2, 0, 0},
				{0, 0, 0},
			},
		}
		p, _ := NewPainter(PainterOptions{
			Type:   ChartOutputPNG,
			Width:  600,
			Height: 600,
		})
		c := NewChordChart(p, ChordChartOption{
			Chord: ChordOption{
				Matrix: matrix,
			},
		})
		nodes, edges, err := c.validateAndBuild()
		assert.Nil(err)
		assert.Equal(3, len(edges))
		// arcs tile the full circle with gaps, no overlap between adjacent nodes
		for i := 0; i+1 < len(nodes); i++ {
			assert.True(nodes[i].end <= nodes[i+1].start)
		}
		assert.InDelta(2*math.Pi, nodes[len(nodes)-1].end+math.Pi/2+defaultChordGap, 1e-9)
	})

	t.Run("disconnected node keeps minimal arc", func(t *testing.T) {
		matrix := FlowMatrix{
			Names: []string{"A", "B", "C"},
			Data: [][]float64{
				{0, 5, 0},
				{5, 0, 0},
				{0, 0, 0},
			},
		}
		p, _ := NewPainter(PainterOptions{
			Type:   ChartOutputSVG,
			Width:  500,
			Height: 500,
		})
		c := NewChordChart(p, ChordChartOption{
			Chord: ChordOption{
				Matrix: matrix,
			},
		})
		nodes, _, err := c.validateAndBuild()
		assert.Nil(err)
		// the fully disconnected node has zero weight yet keeps a minimal arc
		assert.Equal(0.0, nodes[2].total)
		assert.True(nodes[2].end > nodes[2].start)
	})
}

func TestChordRenderOutputs(t *testing.T) {
	assert := assert.New(t)

	matrix := FlowMatrix{
		Names: []string{"A", "B", "C", "D"},
		Data: [][]float64{
			{1, 2, 0, 1},
			{0, 0, 3, 0},
			{2, 0, 0, 1},
			{0, 1, 1, 0},
		},
	}

	for _, outputType := range []string{ChartOutputPNG, ChartOutputSVG} {
		t.Run(outputType, func(t *testing.T) {
			buf, err := ChordRender(matrix,
				TypeOptionFunc(outputType),
				TitleTextOptionFunc("Directed Chord"),
				ChordOptionFunc(ChordOption{
					Matrix:    matrix,
					ShowArrow: true,
				}),
			)
			assert.Nil(err)
			data, err := buf.Bytes()
			assert.Nil(err)
			assert.NotEmpty(data)
			if outputType == ChartOutputSVG {
				assert.Contains(string(data), "<svg")
				// every node name is rendered as label and legend
				for _, name := range matrix.Names {
					assert.Contains(string(data), name)
				}
			} else {
				// PNG magic header
				assert.Equal([]byte{0x89, 0x50, 0x4e, 0x47}, data[:4])
			}
		})
	}
}

func TestChordRenderScaleAlignment(t *testing.T) {
	assert := assert.New(t)

	matrix := FlowMatrix{
		Names: []string{"A", "B", "C"},
		Data: [][]float64{
			{0, 2, 1},
			{1, 0, 1},
			{1, 1, 0},
		},
	}
	// the angle layout must be identical regardless of canvas size,
	// so arcs and ribbon endpoints stay aligned when scaling the canvas.
	buildLayout := func(width, height int) ([]*chordNode, []*chordEdge) {
		p, err := NewPainter(PainterOptions{
			Type:   ChartOutputSVG,
			Width:  width,
			Height: height,
		})
		assert.Nil(err)
		c := NewChordChart(p, ChordChartOption{
			Chord: ChordOption{
				Matrix: matrix,
			},
		})
		nodes, edges, err := c.validateAndBuild()
		assert.Nil(err)
		return nodes, edges
	}
	smallNodes, smallEdges := buildLayout(300, 200)
	bigNodes, bigEdges := buildLayout(900, 600)
	for i := range smallNodes {
		assert.InDelta(smallNodes[i].start, bigNodes[i].start, 1e-9)
		assert.InDelta(smallNodes[i].end, bigNodes[i].end, 1e-9)
	}
	for i := range smallEdges {
		assert.InDelta(smallEdges[i].srcStart, bigEdges[i].srcStart, 1e-9)
		assert.InDelta(smallEdges[i].targetEnd, bigEdges[i].targetEnd, 1e-9)
	}

	// rendering at both sizes must succeed and produce non-empty output
	for _, size := range [][2]int{{300, 200}, {900, 600}} {
		p, err := NewPainter(PainterOptions{
			Type:   ChartOutputPNG,
			Width:  size[0],
			Height: size[1],
		})
		assert.Nil(err)
		_, err = NewChordChart(p, ChordChartOption{
			Chord: ChordOption{
				Matrix: matrix,
			},
		}).Render()
		assert.Nil(err)
	}
}

func TestChordRenderStableOrder(t *testing.T) {
	assert := assert.New(t)

	matrix := FlowMatrix{
		Names: []string{"Alpha", "Beta", "Gamma"},
		Data: [][]float64{
			{0, 3, 0},
			{0, 0, 2},
			{4, 0, 0},
		},
	}
	render := func() string {
		p, err := ChordRender(matrix, TypeOptionFunc(ChartOutputSVG))
		assert.Nil(err)
		buf, err := p.Bytes()
		assert.Nil(err)
		return string(buf)
	}
	first := render()
	second := render()
	// fixed angle allocation and fixed iteration order give stable output
	assert.Equal(first, second)
}

func TestChardNodeColorOption(t *testing.T) {
	assert := assert.New(t)

	matrix := FlowMatrix{
		Names: []string{"A", "B"},
		Data: [][]float64{
			{0, 2},
			{1, 0},
		},
	}
	p, err := NewPainter(PainterOptions{
		Type:   ChartOutputSVG,
		Width:  400,
		Height: 400,
	})
	assert.Nil(err)
	customColor := parseColor("#123456")
	c := NewChordChart(p, ChordChartOption{
		Chord: ChordOption{
			Matrix:     matrix,
			NodeColors: []Color{customColor},
		},
	})
	nodes, _, err := c.validateAndBuild()
	assert.Nil(err)
	assert.Equal(customColor, nodes[0].color)
	// fallback to theme palette for nodes without explicit color
	assert.False(nodes[1].color.IsZero())
}
