package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	charts "github.com/vicanso/go-charts/v2"
)

func writeFile(buf []byte) error {
	tmpPath := "./tmp"
	err := os.MkdirAll(tmpPath, 0700)
	if err != nil {
		return err
	}

	file := filepath.Join(tmpPath, "facet-chart.svg")
	err = ioutil.WriteFile(file, buf, 0600)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	xLabels := []string{"Mon", "Tue", "Wed", "Thu"}
	panels := []charts.ChartOption{
		{
			Title:  charts.TitleOption{Text: "iOS"},
			XAxis:  charts.NewXAxisOption(xLabels),
			Legend: charts.NewLegendOption([]string{"Active"}),
			SeriesList: charts.SeriesList{
				charts.NewSeriesFromValues([]float64{120, 132, 101, 134}),
			},
		},
		{
			Title:  charts.TitleOption{Text: "Android"},
			XAxis:  charts.NewXAxisOption(xLabels),
			Legend: charts.NewLegendOption([]string{"Active"}),
			SeriesList: charts.SeriesList{
				charts.NewSeriesFromValues([]float64{220, 182, 191, 234}),
			},
		},
	}

	p, err := charts.FacetChart(charts.FacetChartOption{
		Type:       charts.ChartOutputSVG,
		Width:      800,
		Height:     400,
		Rows:       1,
		Columns:    2,
		ShareYAxis: true,
		Charts:     panels,
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
	fmt.Println("facet chart rendered")
}
