package main

import (
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
	err = os.WriteFile(file, buf, 0600)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	// the flow of resource loading: download -> decode -> upload
	opt := charts.SankeyChartOption{
		Title: charts.TitleOption{
			Text: "Texture Loading",
		},
		Nodes: []charts.SankeyNode{
			{Name: "Download", Layer: -1},
			{Name: "Memory Cache", Layer: -1},
			{Name: "Decode", Layer: -1},
			{Name: "Disk Cache", Layer: -1},
			{Name: "GPU Upload", Layer: -1},
			{Name: "Dropped", Layer: -1},
		},
		Links: []charts.SankeyLink{
			{Source: "Download", Target: "Decode", Value: 6},
			{Source: "Download", Target: "Disk Cache", Value: 2},
			{Source: "Memory Cache", Target: "GPU Upload", Value: 3},
			{Source: "Decode", Target: "GPU Upload", Value: 5},
			{Source: "Decode", Target: "Dropped", Value: 1},
			{Source: "Disk Cache", Target: "Decode", Value: 2},
		},
	}

	for _, item := range []struct {
		name       string
		outputType string
	}{
		{
			name:       "sankey-chart.png",
			outputType: charts.ChartOutputPNG,
		},
		{
			name:       "sankey-chart.svg",
			outputType: charts.ChartOutputSVG,
		},
	} {
		p, err := charts.NewPainter(charts.PainterOptions{
			Type:   item.outputType,
			Width:  800,
			Height: 500,
		})
		if err != nil {
			panic(err)
		}
		_, err = charts.NewSankeyChart(p, opt).Render()
		if err != nil {
			panic(err)
		}
		buf, err := p.Bytes()
		if err != nil {
			panic(err)
		}
		err = writeFile(item.name, buf)
		if err != nil {
			panic(err)
		}
	}
}
