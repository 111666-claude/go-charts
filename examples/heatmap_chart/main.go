package main

import (
	"math"
	"os"
	"path/filepath"

	"github.com/vicanso/go-charts/v2"
)

func writeFile(name string, buf []byte) error {
	tmpPath := "./tmp"
	err := os.MkdirAll(tmpPath, 0700)
	if err != nil {
		return err
	}
	file := filepath.Join(tmpPath, name)
	return os.WriteFile(file, buf, 0600)
}

func main() {
	// Level difficulty matrix: rows are roles, columns are chapters.
	data := [][]float64{
		{-10, -5, 0, 3},
		{-7.5, 2, 5, 8},
		{math.NaN(), 1, 10, 6},
	}
	p, err := charts.HeatMapRender(
		data,
		[]string{"Warrior", "Mage", "Rogue"},
		[]string{"Chapter 1", "Chapter 2", "Chapter 3", "Chapter 4"},
		charts.TitleTextOptionFunc("Level Difficulty Heat Map"),
		charts.WidthOptionFunc(800),
		charts.HeightOptionFunc(500),
	)
	if err != nil {
		panic(err)
	}
	buf, err := p.Bytes()
	if err != nil {
		panic(err)
	}
	if err = writeFile("heatmap-chart.png", buf); err != nil {
		panic(err)
	}
}
