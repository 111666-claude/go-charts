package main

import (
	"io/ioutil"
	"os"
	"path/filepath"

	charts "github.com/vicanso/go-charts/v2"
)

func writeFile(name string, buf []byte) error {
	tmpPath := "./tmp"
	err := os.MkdirAll(tmpPath, 0700)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(tmpPath, name), buf, 0600)
}

func main() {
	// Acceptance case: A->B=2, B->A=1, plus self loop, one-way edge,
	// zero row and an asymmetric matrix.
	matrix := charts.FlowMatrix{
		Names: []string{"A", "B", "C", "D"},
		Data: [][]float64{
			// A    B    C    D
			{1.0, 2.0, 0.0, 1.0}, // A: self loop + A->B + A->D
			{1.0, 0.0, 3.0, 0.0}, // B: B->A + B->C
			{0.0, 0.0, 0.0, 0.0}, // C: zero row (only incoming)
			{0.0, 1.0, 1.0, 0.0}, // D
		},
	}

	for _, outputType := range []string{charts.ChartOutputPNG, charts.ChartOutputSVG} {
		p, err := charts.ChordRender(
			matrix,
			charts.TypeOptionFunc(outputType),
			charts.TitleTextOptionFunc("Directed Chord", "A->B=2, B->A=1"),
			charts.ChordOptionFunc(charts.ChordOption{
				Matrix:    matrix,
				ShowArrow: true,
			}),
			charts.LegendOptionFunc(charts.LegendOption{
				Data: matrix.Names,
				Top:  "10",
			}),
		)
		if err != nil {
			panic(err)
		}
		buf, err := p.Bytes()
		if err != nil {
			panic(err)
		}
		ext := "png"
		if outputType == charts.ChartOutputSVG {
			ext = "svg"
		}
		if err := writeFile("chord-chart."+ext, buf); err != nil {
			panic(err)
		}
	}

	// small canvas to check scale alignment
	p, err := charts.ChordRender(
		matrix,
		charts.PNGTypeOption(),
		charts.WidthOptionFunc(320),
		charts.HeightOptionFunc(240),
	)
	if err != nil {
		panic(err)
	}
	buf, err := p.Bytes()
	if err != nil {
		panic(err)
	}
	if err := writeFile("chord-chart-small.png", buf); err != nil {
		panic(err)
	}
}
