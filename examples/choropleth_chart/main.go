package main

import (
	"io/ioutil"
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
	return ioutil.WriteFile(filepath.Join(tmpPath, name), buf, 0600)
}

func value(v float64) *float64 {
	return &v
}

func main() {
	// two adjacent unit squares, the first one contains a lake (hole)
	regions := []charts.MapRegion{
		{
			Name: "North",
			Polygon: []charts.Point2D{
				{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1},
			},
			// a lake of area 1/4, it keeps the background color
			Holes: [][]charts.Point2D{
				{{X: 0.25, Y: 0.25}, {X: 0.75, Y: 0.25}, {X: 0.75, Y: 0.75}, {X: 0.25, Y: 0.75}},
			},
			Value: value(1),
		},
		{
			Name: "South",
			Polygon: []charts.Point2D{
				{X: 1, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 1}, {X: 1, Y: 1},
			},
			Value: value(3),
		},
	}
	opt := charts.ChoroplethChartOption{
		Width:     500,
		Height:    420,
		ShowLabel: true,
		Padding: charts.Box{
			Top:    20,
			Right:  20,
			Bottom: 20,
			Left:   20,
		},
		Title: charts.TitleOption{
			Text:    "Planar Choropleth",
			Subtext: "polygon regions with a lake hole",
			Left:    charts.PositionCenter,
		},
		Regions: regions,
		// planar coordinates are used directly, no geographic projection
		VisualMap: charts.VisualMap{
			Min:        charts.NewFloatPoint(0),
			Max:        charts.NewFloatPoint(4),
			PieceCount: 2,
			ShowNoData: true,
		},
	}

	// PNG output
	pngOpt := opt
	pngOpt.Type = charts.ChartOutputPNG
	pngPainter, err := charts.ChoroplethRender(pngOpt)
	if err != nil {
		panic(err)
	}
	pngBuf, err := pngPainter.Bytes()
	if err != nil {
		panic(err)
	}
	err = writeFile("choropleth-chart.png", pngBuf)
	if err != nil {
		panic(err)
	}

	// SVG output shares the same fill geometry
	svgOpt := opt
	svgOpt.Type = charts.ChartOutputSVG
	svgPainter, err := charts.ChoroplethRender(svgOpt)
	if err != nil {
		panic(err)
	}
	svgBuf, err := svgPainter.Bytes()
	if err != nil {
		panic(err)
	}
	err = writeFile("choropleth-chart.svg", svgBuf)
	if err != nil {
		panic(err)
	}
}
