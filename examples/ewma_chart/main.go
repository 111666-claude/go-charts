package main

import (
	"os"
	"path/filepath"

	"github.com/vicanso/go-charts/v2"
)

func writeFile(file string, buf []byte) error {
	tmpPath := "./tmp"
	err := os.MkdirAll(tmpPath, 0700)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(tmpPath, file), buf, 0600)
}

func main() {
	// 渲染帧时延(ms)，基线均值10、标准差2
	values := []float64{
		9.5, 10.2, 11.1, 9.8, 10.5, 12.8, 13.6, 14.2, 13.9, 14.5,
		15.1, 14.8, 10.3, 9.9, 10.6, 10.1, 9.7, 10.4, 11.0, 10.8,
	}
	p, err := charts.EWMARender(charts.EWMAChartOption{
		Title: charts.TitleOption{
			Text:    "Frame Latency EWMA",
			Subtext: "mean=10, stdDev=2, lambda=0.3",
		},
		Legend: charts.LegendOption{
			// 避免与标题重叠
			Padding: charts.Box{
				Top: 40,
			},
		},
		EWMA: charts.EWMAOption{
			Values: values,
			Mean:   10,
			StdDev: 2,
			Lambda: 0.3,
		},
	})
	if err != nil {
		panic(err)
	}
	buf, err := p.Bytes()
	if err != nil {
		panic(err)
	}
	if err := writeFile("ewma-chart.svg", buf); err != nil {
		panic(err)
	}
}
