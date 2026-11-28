package main

import (
	"io/ioutil"
	"math"
	"os"
	"path/filepath"

	"github.com/vicanso/go-charts/v2"
)

func writeFile(buf []byte, name string) error {
	tmpPath := "./tmp"
	err := os.MkdirAll(tmpPath, 0700)
	if err != nil {
		return err
	}
	file := filepath.Join(tmpPath, name)
	return ioutil.WriteFile(file, buf, 0600)
}

func main() {
	// The frame latency ranking before and after a release
	data := []charts.SlopeChartData{
		{Name: "Login", Left: 48, Right: 32},
		{Name: "Feed", Left: 35, Right: 41},
		{Name: "Player", Left: 29, Right: 18},
		{Name: "Chat", Left: 22, Right: 26},
		{Name: "Search", Left: 15, Right: 24},
		{Name: "Pay", Left: 12, Right: 9},
	}
	p, err := charts.SlopeRender(
		data,
		charts.SlopeTypeOptionFunc(charts.ChartOutputPNG),
		charts.SlopeTitleOptionFunc(charts.TitleOption{
			Text: "Frame Latency Ranking (ms)",
		}),
		charts.SlopeTimeLabelOptionFunc("Before Release", "After Release"),
		charts.SlopeLabelShowOptionFunc(true),
		// Stable sort by the left time point, descending
		charts.SlopeSortOptionFunc(charts.SlopeSortLeft, charts.SlopeOrderDesc),
	)
	if err != nil {
		panic(err)
	}
	buf, err := p.Bytes()
	if err != nil {
		panic(err)
	}
	err = writeFile(buf, "slope-chart.png")
	if err != nil {
		panic(err)
	}

	// SVG output with missing values
	missing := []charts.SlopeChartData{
		{Name: "A", Left: 1, Right: 5},
		{Name: "B", Left: 4, Right: 2},
		{Name: "C", Left: 3, Right: math.NaN()},
	}
	svgPainter, err := charts.SlopeRender(
		missing,
		charts.SlopeTypeOptionFunc(charts.ChartOutputSVG),
		charts.SlopeTimeLabelOptionFunc("Before", "After"),
		charts.SlopeLabelShowOptionFunc(true),
		charts.SlopeMinMaxOptionFunc(0, 6),
	)
	if err != nil {
		panic(err)
	}
	svgBuf, err := svgPainter.Bytes()
	if err != nil {
		panic(err)
	}
	err = writeFile(svgBuf, "slope-chart.svg")
	if err != nil {
		panic(err)
	}

	// Acceptance case: A goes 1 -> 5 and B goes 4 -> 2,
	// the two lines must cross at value 3 in the middle
	acceptance := []charts.SlopeChartData{
		{Name: "A", Left: 1, Right: 5},
		{Name: "B", Left: 4, Right: 2},
	}
	acceptPainter, err := charts.SlopeRender(
		acceptance,
		charts.SlopeTypeOptionFunc(charts.ChartOutputPNG),
		charts.SlopeTitleOptionFunc(charts.TitleOption{
			Text: "Slope Chart Acceptance",
		}),
		charts.SlopeTimeLabelOptionFunc("Before", "After"),
		charts.SlopeLabelShowOptionFunc(true),
		charts.SlopeMinMaxOptionFunc(0, 6),
	)
	if err != nil {
		panic(err)
	}
	acceptBuf, err := acceptPainter.Bytes()
	if err != nil {
		panic(err)
	}
	err = writeFile(acceptBuf, "slope-chart-acceptance.png")
	if err != nil {
		panic(err)
	}
}
