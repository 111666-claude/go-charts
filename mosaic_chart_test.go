package charts

import (
	"bytes"
	"image/png"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLargestRemainderDivide(t *testing.T) {
	assert := assert.New(t)

	assert.Equal([]int{}, largestRemainderDivide(10, nil))
	assert.Equal([]int{0, 0}, largestRemainderDivide(0, []float64{1, 2}))
	assert.Equal([]int{0, 0}, largestRemainderDivide(10, []float64{0, 0}))

	// Exact split
	assert.Equal([]int{5, 5}, largestRemainderDivide(10, []float64{1, 1}))
	// The parts always sum to the total
	for _, total := range []int{1, 3, 7, 100, 333} {
		parts := largestRemainderDivide(total, []float64{30, 10, 10, 50})
		sum := 0
		for _, v := range parts {
			assert.GreaterOrEqual(v, 0)
			sum += v
		}
		assert.Equal(total, sum)
	}
	// Deterministic: repeated calls return the same result
	first := largestRemainderDivide(7, []float64{1, 1, 1})
	for i := 0; i < 5; i++ {
		assert.Equal(first, largestRemainderDivide(7, []float64{1, 1, 1}))
	}
}

func TestNewMosaicLayout(t *testing.T) {
	assert := assert.New(t)

	data := [][]float64{
		{30, 10},
		{10, 50},
	}
	const width = 500
	const height = 300
	layout, err := NewMosaicLayout(data, width, height)
	assert.Nil(err)

	// Column totals are 40 and 60, so the widths are 40% / 60%
	assert.Equal([]int{0, 200, 500}, layout.ColumnX)
	assert.Equal(200, layout.ColumnX[1]-layout.ColumnX[0])
	assert.Equal(300, layout.ColumnX[2]-layout.ColumnX[1])

	// In the first column the top cell (30 of 40) takes 75% of the height
	firstColumnY := layout.RowY[0]
	assert.Equal(0, firstColumnY[0])
	assert.Equal(225, firstColumnY[1])
	assert.Equal(300, firstColumnY[2])

	// The second column: 10/60 and 50/60
	secondColumnY := layout.RowY[1]
	assert.Equal(0, secondColumnY[0])
	assert.Equal(50, secondColumnY[1])
	assert.Equal(300, secondColumnY[2])

	// All non-zero cells fill the plot area exactly once without overlap
	areaSum := 0
	for _, cell := range layout.Cells {
		assert.Equal(cell.Column, cell.Column)
		areaSum += cell.Box.Width() * cell.Box.Height()
	}
	assert.Equal(width*height, areaSum)

	// Bounds check: every cell is inside the plot
	for _, cell := range layout.Cells {
		assert.GreaterOrEqual(cell.Box.Left, 0)
		assert.GreaterOrEqual(cell.Box.Top, 0)
		assert.LessOrEqual(cell.Box.Right, width)
		assert.LessOrEqual(cell.Box.Bottom, height)
	}
}

func TestNewMosaicLayoutCoverage(t *testing.T) {
	assert := assert.New(t)

	// A larger random-ish table: the tiles should cover the whole area
	data := [][]float64{
		{12, 0, 7, 3},
		{0, 5, 11, 9},
		{4, 8, 0, 1},
	}
	const width = 640
	const height = 360
	layout, err := NewMosaicLayout(data, width, height)
	assert.Nil(err)

	// Reconstruct the pixel coverage matrix
	covered := make([][]bool, height)
	for i := range covered {
		covered[i] = make([]bool, width)
	}
	for _, cell := range layout.Cells {
		for y := cell.Box.Top; y < cell.Box.Bottom; y++ {
			for x := cell.Box.Left; x < cell.Box.Right; x++ {
				// No overlap
				assert.False(covered[y][x])
				covered[y][x] = true
			}
		}
	}
	// No gap: every pixel is covered by a tile
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			assert.True(covered[y][x])
		}
	}
	// Zero cells are not rendered
	zeroCellCount := 0
	for _, row := range data {
		for _, v := range row {
			if v == 0 {
				zeroCellCount++
			}
		}
	}
	assert.Equal(len(data)*len(data[0])-zeroCellCount, len(layout.Cells))
}

func TestNewMosaicLayoutZeroColumn(t *testing.T) {
	assert := assert.New(t)

	data := [][]float64{
		{0, 10, 0},
		{0, 30, 0},
	}
	layout, err := NewMosaicLayout(data, 100, 200)
	assert.Nil(err)
	// Zero columns get zero width
	assert.Equal(0, layout.ColumnX[1]-layout.ColumnX[0])
	assert.Equal(100, layout.ColumnX[2]-layout.ColumnX[1])
	assert.Equal(0, layout.ColumnX[3]-layout.ColumnX[2])
	// All row boundaries of a zero column are 0
	for _, y := range layout.RowY[0] {
		assert.Equal(0, y)
	}
	for _, y := range layout.RowY[2] {
		assert.Equal(0, y)
	}
	// Only the middle column contributes tiles, and it fills the area
	areaSum := 0
	for _, cell := range layout.Cells {
		assert.Equal(1, cell.Column)
		areaSum += cell.Box.Width() * cell.Box.Height()
	}
	assert.Equal(100*200, areaSum)
}

func TestNewMosaicLayoutValidation(t *testing.T) {
	assert := assert.New(t)

	// Empty table
	_, err := NewMosaicLayout(nil, 100, 100)
	assert.NotNil(err)
	_, err = NewMosaicLayout([][]float64{}, 100, 100)
	assert.NotNil(err)
	_, err = NewMosaicLayout([][]float64{{}}, 100, 100)
	assert.NotNil(err)

	// Ragged table
	_, err = NewMosaicLayout([][]float64{
		{1, 2},
		{3},
	}, 100, 100)
	assert.NotNil(err)

	// Negative value
	_, err = NewMosaicLayout([][]float64{
		{1, -2},
		{3, 4},
	}, 100, 100)
	assert.NotNil(err)

	// Non-finite value
	_, err = NewMosaicLayout([][]float64{
		{1, math.NaN()},
		{3, 4},
	}, 100, 100)
	assert.NotNil(err)
	_, err = NewMosaicLayout([][]float64{
		{1, math.Inf(1)},
		{3, 4},
	}, 100, 100)
	assert.NotNil(err)

	// All zero table
	_, err = NewMosaicLayout([][]float64{
		{0, 0},
		{0, 0},
	}, 100, 100)
	assert.NotNil(err)

	// Duplicate labels do not cause an error
	_, err = MosaicOptionRender(MosaicChartOption{
		Data: [][]float64{
			{1, 2},
			{3, 4},
		},
		RowLabels:    []string{"same", "same"},
		ColumnLabels: []string{"same", "same"},
	})
	assert.Nil(err)
}

func testMosaicOption() MosaicChartOption {
	return MosaicChartOption{
		Width:  600,
		Height: 400,
		Data: [][]float64{
			{30, 10},
			{10, 50},
		},
		ColumnLabels: []string{"PNG", "SVG"},
		RowLabels:    []string{"Image", "Vector"},
		ShowValue:    true,
		Title: TitleOption{
			Text: "Asset Library",
			Left: PositionCenter,
		},
		Padding: Box{
			Top:    20,
			Right:  20,
			Bottom: 20,
			Left:   20,
		},
	}
}

func TestMosaicChartRenderGolden(t *testing.T) {
	assert := assert.New(t)
	opt := testMosaicOption()
	opt.Type = ChartOutputSVG
	p, err := MosaicOptionRender(opt)
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	assert.Equal(goldenMosaicSVG, string(buf))
}

func TestMosaicChartPNGSVGAlignment(t *testing.T) {
	assert := assert.New(t)

	// The two renderers receive the same integer coordinates through
	// the same painter, so the tile rects in SVG and the pixels in PNG
	// must describe identical geometry.
	svgOpt := testMosaicOption()
	svgOpt.Type = ChartOutputSVG
	svgPainter, err := MosaicOptionRender(svgOpt)
	assert.Nil(err)
	svgBuf, err := svgPainter.Bytes()
	assert.Nil(err)

	pngOpt := testMosaicOption()
	pngOpt.Type = ChartOutputPNG
	pngPainter, err := MosaicOptionRender(pngOpt)
	assert.Nil(err)
	pngBuf, err := pngPainter.Bytes()
	assert.Nil(err)

	img, err := png.Decode(bytes.NewReader(pngBuf))
	assert.Nil(err)
	bounds := img.Bounds()
	assert.Equal(600, bounds.Dx())
	assert.Equal(400, bounds.Dy())

	// Every rect path in the SVG must be a solid filled region of the
	// same series color in the PNG at its center pixel.
	seriesColors := []struct{ r, g, b uint32 }{
		{84, 112, 198},
		{145, 204, 117},
	}
	centers := []struct{ x, y int }{
		// First column, top tile (30): x in [55,265), y in [55,254)
		{160, 150},
		// First column, bottom tile (10): y in [254,320)
		{100, 290},
		// Second column, top tile (10): x in [265,580), y in [55,99)
		{420, 77},
		// Second column, bottom tile (50): y in [99,320)
		{420, 200},
	}
	for index, c := range centers {
		r, g, b, _ := img.At(c.x, c.y).RGBA()
		expected := seriesColors[index%2]
		assert.Equal(expected.r, r>>8)
		assert.Equal(expected.g, g>>8)
		assert.Equal(expected.b, b>>8)
	}
	// The SVG describes the same four tile rects with the exact same
	// integer geometry as the PNG pixels
	expectedTilePaths := []string{
		"M 55 55\nL 265 55\nL 265 254\nL 55 254",
		"M 55 254\nL 265 254\nL 265 320\nL 55 320",
		"M 265 55\nL 580 55\nL 580 99\nL 265 99",
		"M 265 99\nL 580 99\nL 580 320\nL 265 320",
	}
	for _, path := range expectedTilePaths {
		assert.True(strings.Contains(string(svgBuf), path))
	}

	// Tile boundaries are shared edges: the pixel just left of the
	// column boundary x=265 belongs to column one, and the pixel at
	// x=265 belongs to column two (no overlap, no gap).
	r1, g1, b1, _ := img.At(264, 200).RGBA()
	r2, g2, b2, _ := img.At(265, 200).RGBA()
	assert.Equal(uint32(84), r1>>8)
	assert.Equal(uint32(112), g1>>8)
	assert.Equal(uint32(198), b1>>8)
	assert.Equal(uint32(145), r2>>8)
	assert.Equal(uint32(204), g2>>8)
	assert.Equal(uint32(117), b2>>8)

	// The background below the plot area stays white
	r, g, b, a := img.At(300, 360).RGBA()
	assert.Equal(uint32(255), r>>8)
	assert.Equal(uint32(255), g>>8)
	assert.Equal(uint32(255), b>>8)
	assert.Equal(uint32(255), a>>8)
}

func TestMosaicChartZeroColumnRender(t *testing.T) {
	assert := assert.New(t)

	for _, typ := range []string{ChartOutputSVG, ChartOutputPNG} {
		p, err := MosaicOptionRender(MosaicChartOption{
			Type:   typ,
			Width:  300,
			Height: 200,
			Data: [][]float64{
				{0, 10, 0},
				{0, 30, 0},
			},
			ColumnLabels: []string{"Z1", "OK", "Z2"},
			RowLabels:    []string{"A", "B"},
		})
		assert.Nil(err)
		buf, err := p.Bytes()
		assert.Nil(err)
		assert.NotEmpty(buf)
		if typ == ChartOutputPNG {
			img, err := png.Decode(bytes.NewReader(buf))
			assert.Nil(err)
			// The zero width columns render no tile, the only non-zero
			// column still fills the whole plot width. Check its center
			// pixel is painted with a series color rather than background.
			bounds := img.Bounds()
			cx := (bounds.Min.X + bounds.Max.X) >> 1
			r, g, b, _ := img.At(cx, bounds.Min.Y+20).RGBA()
			// A series color, not the white background
			assert.False(r == 0xffff && g == 0xffff && b == 0xffff)
		}
	}
}

func TestMosaicChartNarrowCanvas(t *testing.T) {
	assert := assert.New(t)

	// A very narrow canvas must still produce a deterministic result
	// instead of panicking
	for _, typ := range []string{ChartOutputSVG, ChartOutputPNG} {
		p, err := MosaicOptionRender(MosaicChartOption{
			Type:   typ,
			Width:  40,
			Height: 60,
			Data: [][]float64{
				{30, 10},
				{10, 50},
			},
		})
		assert.Nil(err)
		buf, err := p.Bytes()
		assert.Nil(err)
		assert.NotEmpty(buf)
	}
}

func TestMosaicChartNewPainterAPI(t *testing.T) {
	assert := assert.New(t)

	// The renderer should also work when driven directly with a Painter,
	// following the same convention as the pie and funnel chart
	p, err := NewPainter(PainterOptions{
		Type:   ChartOutputSVG,
		Width:  600,
		Height: 400,
	}, PainterThemeOption(defaultTheme))
	assert.Nil(err)
	_, err = NewMosaicChart(p, MosaicChartOption{
		Data: [][]float64{
			{30, 10},
			{10, 50},
		},
		ColumnLabels: []string{"A", "B"},
		RowLabels:    []string{"X", "Y"},
	}).Render()
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	assert.True(strings.HasPrefix(string(buf), "<svg"))
}

func TestMosaicChartValueFormatter(t *testing.T) {
	assert := assert.New(t)

	calls := 0
	p, err := MosaicOptionRender(MosaicChartOption{
		Type:   ChartOutputSVG,
		Width:  600,
		Height: 400,
		Data: [][]float64{
			{30, 10},
			{10, 50},
		},
		ShowValue: true,
		ValueFormatter: func(row, column int, value float64) string {
			calls++
			return "v"
		},
	})
	assert.Nil(err)
	buf, err := p.Bytes()
	assert.Nil(err)
	// One call per non-zero cell
	assert.Equal(4, calls)
	assert.Equal(4, strings.Count(string(buf), ">v<"))
}
