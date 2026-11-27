package main

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/vicanso/go-charts/v2"
)

func writeFile(buf []byte) error {
	tmpPath := "./tmp"
	err := os.MkdirAll(tmpPath, 0700)
	if err != nil {
		return err
	}

	file := filepath.Join(tmpPath, "mosaic-chart.png")
	err = ioutil.WriteFile(file, buf, 0600)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	// The frequency table of file format (row) and source channel (column).
	// The width of each column follows its column total, and the height of
	// a tile follows its proportion inside the column.
	data := [][]float64{
		{30, 10},
		{10, 50},
	}
	p, err := charts.MosaicOptionRender(charts.MosaicChartOption{
		Data:         data,
		ColumnLabels: []string{"Search", "Direct"},
		RowLabels:    []string{"PNG", "SVG"},
		ShowValue:    true,
		Title: charts.TitleOption{
			Text: "Asset Library",
			Left: charts.PositionCenter,
		},
		Legend: charts.LegendOption{
			Orient: charts.OrientVertical,
			Left:   charts.PositionLeft,
		},
		Padding: charts.Box{
			Top:    20,
			Right:  20,
			Bottom: 20,
			Left:   20,
		},
	})
	if err != nil {
		panic(err)
	}

	buf, err := p.Bytes()
	if err != nil {
		panic(err)
	}
	err = writeFile(buf)
	if err != nil {
		panic(err)
	}
}
