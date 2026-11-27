// MIT License

// Copyright (c) 2022 Tree Xie

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package charts

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEWMAProcess(t *testing.T) {
	assert := assert.New(t)

	// 手算序列：mean=10, lambda=0.5
	// z(1) = 0.5*18 + 0.5*10 = 14
	// z(2) = 0.5*18 + 0.5*14 = 16
	points, err := EWMAProcess(EWMAOption{
		Values: []float64{18, 18},
		Mean:   10,
		StdDev: 2,
		Lambda: 0.5,
	})
	assert.Nil(err)
	assert.Equal(2, len(points))
	assert.Equal(14.0, points[0].Value)
	assert.Equal(16.0, points[1].Value)

	// 手算序列：mean=10, stdDev=2, lambda=0.5, limitSigma=3
	// z: 14, 16, 13, 11.5
	// 控制限：10 ± 3*2*sqrt(0.5/1.5*(1-0.5^(2n)))
	// n=1: 10±3, n=2: 10±3.3541, n=3: 10±3.4369, n=4: 10±3.4573
	points, err = EWMAProcess(EWMAOption{
		Values: []float64{18, 18, 10, 10},
		Mean:   10,
		StdDev: 2,
		Lambda: 0.5,
	})
	assert.Nil(err)
	assert.Equal(4, len(points))
	expectedValues := []float64{14, 16, 13, 11.5}
	expectedUCL := []float64{13, 13.3541, 13.4369, 13.4573}
	expectedOOC := []bool{true, true, false, false}
	for i, point := range points {
		assert.False(point.Missing)
		assert.InDelta(expectedValues[i], point.Value, 0.0001)
		assert.InDelta(expectedUCL[i], point.UCL, 0.0001)
		assert.InDelta(20-expectedUCL[i], point.LCL, 0.0001)
		assert.Equal(10.0, point.Center)
		assert.Equal(expectedOOC[i], point.OutOfControl, "index %d", i)
	}

	// 缺测点(nullValue/NaN/Inf)不更新EWMA状态与样本数
	points, err = EWMAProcess(EWMAOption{
		Values: []float64{18, nullValue, math.NaN(), math.Inf(1), 18},
		Mean:   10,
		StdDev: 2,
		Lambda: 0.5,
	})
	assert.Nil(err)
	assert.Equal(5, len(points))
	assert.Equal(14.0, points[0].Value)
	for i := 1; i <= 3; i++ {
		assert.True(points[i].Missing, "index %d", i)
		assert.True(math.IsNaN(points[i].Value), "index %d", i)
		assert.False(points[i].OutOfControl, "index %d", i)
		// 缺测点的控制限与前一个有效点一致(n不变)
		assert.InDelta(13.0, points[i].UCL, 0.0001, "index %d", i)
	}
	assert.False(points[4].Missing)
	// 第5个观测是第2个有效样本：z = 0.5*18 + 0.5*14 = 16，n=2
	assert.Equal(16.0, points[4].Value)
	assert.InDelta(13.3541, points[4].UCL, 0.0001)
	assert.True(points[4].OutOfControl)

	// 零方差：控制限收缩到中心线，任何偏离均值的EWMA均越界
	points, err = EWMAProcess(EWMAOption{
		Values: []float64{10, 18},
		Mean:   10,
		StdDev: 0,
		Lambda: 0.5,
	})
	assert.Nil(err)
	assert.Equal(10.0, points[0].Value)
	assert.False(points[0].OutOfControl)
	assert.Equal(14.0, points[1].Value)
	assert.True(points[1].OutOfControl)
	for _, point := range points {
		assert.Equal(10.0, point.UCL)
		assert.Equal(10.0, point.LCL)
	}

	// lambda=1时EWMA退化为原始观测
	points, err = EWMAProcess(EWMAOption{
		Values: []float64{5, 7},
		Mean:   0,
		StdDev: 1,
		Lambda: 1,
	})
	assert.Nil(err)
	assert.Equal(5.0, points[0].Value)
	assert.Equal(7.0, points[1].Value)
	// lambda=1时控制限退化为恒定值 mean±limitSigma*stdDev
	assert.InDelta(3.0, points[0].UCL, 0.0001)
	assert.InDelta(3.0, points[1].UCL, 0.0001)

	// 重置分段：第3个观测处重置，EWMA状态与样本数重新开始
	points, err = EWMAProcess(EWMAOption{
		Values:       []float64{18, 18, 18, 18},
		Mean:         10,
		StdDev:       2,
		Lambda:       0.5,
		ResetIndexes: []int{2},
	})
	assert.Nil(err)
	assert.Equal([]float64{14, 16, 14, 16}, []float64{
		points[0].Value, points[1].Value, points[2].Value, points[3].Value,
	})
	// 样本数序列为1,2,1,2，控制限随之重新收窄
	assert.InDelta(13.0, points[0].UCL, 0.0001)
	assert.InDelta(13.3541, points[1].UCL, 0.0001)
	assert.InDelta(13.0, points[2].UCL, 0.0001)
	assert.InDelta(13.3541, points[3].UCL, 0.0001)

	// limitSigma默认值与显式指定3一致
	pointsDefault, err := EWMAProcess(EWMAOption{
		Values: []float64{18},
		Mean:   10,
		StdDev: 2,
		Lambda: 0.5,
	})
	assert.Nil(err)
	pointsExplicit, err := EWMAProcess(EWMAOption{
		Values:     []float64{18},
		Mean:       10,
		StdDev:     2,
		Lambda:     0.5,
		LimitSigma: 3,
	})
	assert.Nil(err)
	assert.Equal(pointsExplicit[0].UCL, pointsDefault[0].UCL)
	assert.Equal(pointsExplicit[0].LCL, pointsDefault[0].LCL)

	// 空观测返回空结果
	points, err = EWMAProcess(EWMAOption{
		Mean:   10,
		StdDev: 2,
		Lambda: 0.5,
	})
	assert.Nil(err)
	assert.Equal(0, len(points))
}

func TestEWMAProcessError(t *testing.T) {
	assert := assert.New(t)

	valid := EWMAOption{
		Values: []float64{18, 18},
		Mean:   10,
		StdDev: 2,
		Lambda: 0.5,
	}
	// 非法lambda
	for _, lambda := range []float64{0, -0.1, 1.5, math.NaN(), math.Inf(1)} {
		opt := valid
		opt.Lambda = lambda
		_, err := EWMAProcess(opt)
		assert.NotNil(err, "lambda %v", lambda)
	}
	// 非有限均值
	for _, mean := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		opt := valid
		opt.Mean = mean
		_, err := EWMAProcess(opt)
		assert.NotNil(err, "mean %v", mean)
	}
	// 非法标准差
	for _, stdDev := range []float64{-1, math.NaN(), math.Inf(1)} {
		opt := valid
		opt.StdDev = stdDev
		_, err := EWMAProcess(opt)
		assert.NotNil(err, "stdDev %v", stdDev)
	}
	// 非法limitSigma
	for _, limitSigma := range []float64{-1, math.NaN(), math.Inf(1)} {
		opt := valid
		opt.LimitSigma = limitSigma
		_, err := EWMAProcess(opt)
		assert.NotNil(err, "limitSigma %v", limitSigma)
	}
	// 越界的重置索引
	for _, resetIndexes := range [][]int{{-1}, {2}, {10}} {
		opt := valid
		opt.ResetIndexes = resetIndexes
		_, err := EWMAProcess(opt)
		assert.NotNil(err, "resetIndexes %v", resetIndexes)
	}
}

func TestEWMAChartRender(t *testing.T) {
	assert := assert.New(t)

	newOption := func() EWMAChartOption {
		return EWMAChartOption{
			Title: TitleOption{
				Text: "EWMA",
			},
			Padding: Box{
				Top:    10,
				Right:  10,
				Bottom: 10,
				Left:   10,
			},
			EWMA: EWMAOption{
				Values: []float64{18, 18, 10, 10, 12, 9},
				Mean:   10,
				StdDev: 2,
				Lambda: 0.5,
			},
		}
	}

	// SVG渲染，y轴刻度由全部曲线(原始值、EWMA、控制限、中心线)共同决定
	p, err := NewPainter(PainterOptions{
		Type:   ChartOutputSVG,
		Width:  600,
		Height: 400,
	}, PainterThemeOption(defaultTheme))
	assert.Nil(err)
	_, err = NewEWMAChart(p, newOption()).Render()
	assert.Nil(err)
	svgData, err := p.Bytes()
	assert.Nil(err)
	svg := string(svgData)
	// 中心线10、上控制限(约13.5)与原始观测18均应落在y轴刻度范围内
	for _, tick := range []string{">8<", ">12<", ">16<", ">20<", ">24<", ">28<"} {
		assert.True(strings.Contains(svg, tick), "tick %s", tick)
	}
	// 图例区分原始折线与各计算曲线
	for _, legend := range []string{">Raw<", ">EWMA<", ">UCL<", ">LCL<", ">Center<"} {
		assert.True(strings.Contains(svg, legend), "legend %s", legend)
	}
	// 控制限与中心线为虚线
	assert.True(strings.Contains(svg, "stroke-dasharray"))
	// 前两个EWMA点(14、16)越上界，应有2个越界标记
	assert.Equal(2, strings.Count(svg, "r=\"4\""))

	// PNG与SVG共用同一渲染流程与刻度计算，渲染结果尺寸一致
	pngPainter, err := EWMARender(func() EWMAChartOption {
		opt := newOption()
		opt.Type = ChartOutputPNG
		return opt
	}())
	assert.Nil(err)
	pngData, err := pngPainter.Bytes()
	assert.Nil(err)
	assert.True(len(pngData) > 8)
	// PNG magic number
	assert.Equal([]byte{0x89, 0x50, 0x4e, 0x47}, pngData[:4])

	// 相同的y轴刻度：SVG与PNG均使用defaultRender计算的axisRange，
	// 再次渲染SVG刻度应保持一致
	svgPainter2, err := EWMARender(newOption())
	assert.Nil(err)
	svgData2, err := svgPainter2.Bytes()
	assert.Nil(err)
	assert.Equal(svg, string(svgData2))

	// 缺测点与重置分段的渲染
	_, err = EWMARender(EWMAChartOption{
		EWMA: EWMAOption{
			Values:       []float64{18, nullValue, 18, 18, 10},
			Mean:         10,
			StdDev:       2,
			Lambda:       0.5,
			ResetIndexes: []int{3},
		},
	})
	assert.Nil(err)

	// 非法参数与空数据应返回错误
	_, err = EWMARender(EWMAChartOption{
		EWMA: EWMAOption{
			Values: []float64{18},
			Mean:   10,
			StdDev: 2,
			Lambda: 0,
		},
	})
	assert.NotNil(err)
	_, err = EWMARender(EWMAChartOption{
		EWMA: EWMAOption{
			Mean:   10,
			StdDev: 2,
			Lambda: 0.5,
		},
	})
	assert.NotNil(err)
}
