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
	return ioutil.WriteFile(file, buf, 0600)
}

func main() {
	// several non-negative resource consumption series sharing the
	// same time axis
	values := [][]float64{
		{120, 132, 101, 134, 90, 230, 210},
		{220, 182, 191, 234, 290, 330, 310},
		{150, 232, 201, 154, 190, 330, 410},
		{320, 332, 301, 334, 390, 330, 320},
	}
	names := []string{
		"Email",
		"Union Ads",
		"Video Ads",
		"Direct",
	}
	xLabels := []string{
		"Mon",
		"Tue",
		"Wed",
		"Thu",
		"Fri",
		"Sat",
		"Sun",
	}

	// centered streamgraph
	p, err := charts.StreamRender(
		values,
		charts.TitleTextOptionFunc("Stream (center)"),
		charts.XAxisDataOptionFunc(xLabels, charts.FalseFlag()),
		charts.LegendLabelsOptionFunc(names, "50"),
		charts.PNGTypeOption(),
	)
	if err != nil {
		panic(err)
	}
	buf, err := p.Bytes()
	if err != nil {
		panic(err)
	}
	err = writeFile("stream-chart-center.png", buf)
	if err != nil {
		panic(err)
	}

	// wiggle baseline
	p, err = charts.StreamRender(
		values,
		charts.TitleTextOptionFunc("Stream (wiggle)"),
		charts.XAxisDataOptionFunc(xLabels, charts.FalseFlag()),
		charts.LegendLabelsOptionFunc(names, "50"),
		charts.StreamBaselineOptionFunc(charts.StreamBaselineWiggle),
		charts.PNGTypeOption(),
	)
	if err != nil {
		panic(err)
	}
	buf, err = p.Bytes()
	if err != nil {
		panic(err)
	}
	err = writeFile("stream-chart-wiggle.png", buf)
	if err != nil {
		panic(err)
	}

	// non-equidistant time series: the samples are positioned by
	// their numeric x values (e.g. unix timestamps)
	p, err = charts.StreamRender(
		values,
		charts.TitleTextOptionFunc("Stream (non-equidistant time)"),
		charts.XAxisDataOptionFunc(xLabels, charts.FalseFlag()),
		charts.LegendLabelsOptionFunc(names, "50"),
		charts.StreamBaselineOptionFunc(charts.StreamBaselineWiggle),
		charts.StreamXValuesOptionFunc([]float64{0, 1, 3, 6, 10, 15, 21}),
		charts.SVGTypeOption(),
	)
	if err != nil {
		panic(err)
	}
	buf, err = p.Bytes()
	if err != nil {
		panic(err)
	}
	err = writeFile("stream-chart-time.svg", buf)
	if err != nil {
		panic(err)
	}
}
