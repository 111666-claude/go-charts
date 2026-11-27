package charts

import (
	"bytes"
	"image/png"
	"testing"
)

// TestWindRoseCompassGeometry verifies that aggregated sectors land in the
// correct compass quadrant with deterministic pixel coordinates.
func TestWindRoseCompassGeometry(t *testing.T) {
	p, err := NewPainter(PainterOptions{
		Type:   ChartOutputPNG,
		Width:  400,
		Height: 400,
	}, PainterThemeOption(defaultTheme))
	if err != nil {
		t.Fatal(err)
	}
	// One datum per quadrant with distinct weights so the radial length
	// differs, making each wedge distinguishable.
	_, err = NewWindRoseChart(p, WindRoseChartOption{
		SeriesList: windRoseDataList([][2]float64{
			{0, 8},   // north
			{90, 4},  // east
			{180, 6}, // south
			{270, 2}, // west
		}),
		SectorWidth:     90,
		AngularAxisShow: FalseFlag(),
		RadialAxis:      WindRoseRadialAxisOption{Show: FalseFlag()},
	}).Render()
	if err != nil {
		t.Fatal(err)
	}
	buf, err := p.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	cx, cy := 200, 200
	// isFilled reports whether a pixel is not the white background
	isFilled := func(x, y int) bool {
		r, g, b, _ := img.At(x, y).RGBA()
		return !(r>>8 == 255 && g>>8 == 255 && b>>8 == 255)
	}
	// Sample points well inside each quadrant wedge (45deg diagonals)
	type probe struct {
		name         string
		x, y         int
		expectFilled bool
	}
	probes := []probe{
		// north quadrant (up), radius ~40
		{"north", cx, cy - 40, true},
		// east quadrant (right), radius ~25
		{"east", cx + 25, cy, true},
		// south quadrant (down), radius ~30
		{"south", cx, cy + 30, true},
		// west quadrant (left), radius ~15
		{"west", cx - 15, cy, true},
	}
	for _, pr := range probes {
		got := isFilled(pr.x, pr.y)
		if got != pr.expectFilled {
			t.Errorf("%s probe (%d,%d): filled=%v want %v", pr.name, pr.x, pr.y, got, pr.expectFilled)
		}
	}
	// Points beyond each wedge's radial extent must stay background.
	// West has the smallest weight (2 of max 8 -> the shortest radius), so
	// a far-west point must be empty while the far-east point is filled.
	if isFilled(cx-70, cy) {
		t.Errorf("far west point should be background (west wedge is the shortest)")
	}
	if !isFilled(cx+70, cy) {
		t.Errorf("far east point should be filled (east wedge is longer)")
	}
}
