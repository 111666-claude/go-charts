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

	file := filepath.Join(tmpPath, name)
	err = ioutil.WriteFile(file, buf, 0600)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	sites := []charts.VoronoiSite{
		{X: 0, Y: 0, Name: "watchtower A"},
		{X: 2, Y: 0, Name: "watchtower B"},
	}
	rect := charts.VoronoiRect{
		X0: 0,
		Y0: -1,
		X1: 2,
		Y1: 1,
	}
	opts := []charts.OptionFunc{
		charts.TitleTextOptionFunc("Bounded Voronoi"),
		charts.LegendLabelsOptionFunc([]string{
			"watchtower A",
			"watchtower B",
		}),
		charts.WidthOptionFunc(600),
		charts.HeightOptionFunc(400),
	}

	svgPainter, err := charts.VoronoiRender(sites, rect, opts...)
	if err != nil {
		panic(err)
	}
	svgBuf, err := svgPainter.Bytes()
	if err != nil {
		panic(err)
	}
	err = writeFile("voronoi-chart.svg", svgBuf)
	if err != nil {
		panic(err)
	}

	pngPainter, err := charts.VoronoiOptionRender(charts.VoronoiChartOption{
		Type:   charts.ChartOutputPNG,
		Width:  600,
		Height: 400,
		Title: charts.TitleOption{
			Text: "Bounded Voronoi",
		},
		Legend: charts.NewLegendOption([]string{
			"watchtower A",
			"watchtower B",
		}),
		Sites:      sites,
		Rect:       rect,
		ShowBorder: true,
		ShowSite:   true,
		ShowLabel:  true,
	})
	if err != nil {
		panic(err)
	}
	pngBuf, err := pngPainter.Bytes()
	if err != nil {
		panic(err)
	}
	err = writeFile("voronoi-chart.png", pngBuf)
	if err != nil {
		panic(err)
	}
}
