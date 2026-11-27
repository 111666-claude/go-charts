package main

import (
	"io/ioutil"
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
	err = ioutil.WriteFile(file, buf, 0600)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	// Each item is a pair of angle and weight: [angle, weight].
	// The angle is in degrees [0, 360): 0 means north and it increases
	// clockwise, e.g. 359 and 1 are both next to north and are aggregated
	// into the same north centered sector.
	values := [][2]float64{
		{359, 1},
		{1, 1},
		{20, 2},
		{45, 3},
		{80, 4},
		{100, 5},
		{135, 2},
		{170, 3},
		{200, 6},
		{230, 4},
		{260, 5},
		{290, 3},
		{320, 2},
		{340, 1},
	}

	p, err := charts.WindRoseRender(
		values,
		charts.TitleTextOptionFunc("Wind Rose"),
		// The sector width is 30 degrees, which divides 360 evenly (12 sectors)
		charts.WindRoseSectorWidthOptionFunc(30),
		charts.LegendLabelsOptionFunc([]string{
			"Wind Frequency",
		}),
	)
	if err != nil {
		panic(err)
	}

	buf, err := p.Bytes()
	if err != nil {
		panic(err)
	}
	err = writeFile(buf, "wind-rose-chart.png")
	if err != nil {
		panic(err)
	}
}
