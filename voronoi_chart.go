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

	"github.com/golang/freetype/truetype"
)

// VoronoiSite is a site (e.g. a watchtower) of a bounded Voronoi chart.
type VoronoiSite struct {
	// X is the x coordinate in the data coordinate space.
	X float64
	// Y is the y coordinate in the data coordinate space.
	Y float64
	// Name is the label rendered next to the site. It is also the legend name.
	Name string
	// FillColor overrides the fill color of the site region. The theme
	// series color is used when it is zero.
	FillColor Color
	// StrokeColor overrides the region border color of this site. The chart
	// level border color is used when it is zero.
	StrokeColor Color
}

// VoronoiChartOption contains all options of a bounded Voronoi chart.
type VoronoiChartOption struct {
	// The theme
	Theme ColorPalette
	// The font
	Font *truetype.Font
	// The output type of chart, "svg" or "png", default value is "svg"
	Type string
	// The width of chart, default width is 600
	Width int
	// The height of chart, default height is 400
	Height int
	// The padding for chart
	Padding Box
	// The option of title
	Title TitleOption
	// The legend option
	Legend LegendOption
	// The sites whose nearest regions are rendered.
	Sites []VoronoiSite
	// Rect is the visible data rectangle. Every region polygon is clipped
	// by this rectangle.
	Rect VoronoiRect
	// FillAlpha is the alpha of the region fill color, 0-255.
	// The default value is 90.
	FillAlpha uint8
	// BorderColor is the shared color of the region borders. The axis
	// stroke color of the theme is used when it is zero.
	BorderColor Color
	// BorderWidth is the stroke width of the region borders, default 1.
	BorderWidth float64
	// ShowBorder controls whether the region borders are rendered.
	ShowBorder bool
	// ShowSite controls whether the site points are rendered.
	ShowSite bool
	// SiteRadius is the radius of the site point in pixels, default 4.
	SiteRadius float64
	// SiteColor overrides the fill color of every site point. The theme
	// series color is used when it is zero.
	SiteColor Color
	// ShowLabel controls whether the site names are rendered.
	ShowLabel bool
	// LabelColor overrides the color of the labels. The theme text color is
	// used when it is zero.
	LabelColor Color
	// LabelFontSize is the font size of the labels, default 10.
	LabelFontSize float64
	// The background color of chart
	BackgroundColor Color
}

type voronoiChart struct {
	p   *Painter
	opt *VoronoiChartOption
}

// NewVoronoiChart returns a bounded Voronoi chart renderer.
func NewVoronoiChart(p *Painter, opt VoronoiChartOption) *voronoiChart {
	if opt.Theme == nil {
		opt.Theme = defaultTheme
	}
	return &voronoiChart{
		p:   p,
		opt: &opt,
	}
}

// NewVoronoiSeriesList builds a series list from the sites so that titles,
// legends and the rest of the rendering pipeline can reuse the common logic.
func NewVoronoiSeriesList(sites []VoronoiSite) SeriesList {
	seriesList := make(SeriesList, len(sites))
	for index, site := range sites {
		seriesList[index] = Series{
			Type: ChartTypeVoronoi,
			Name: site.Name,
			Data: []SeriesData{
				{
					Value: 0,
				},
			},
		}
	}
	return seriesList
}

// toPixel converts a data coordinate point into pixel coordinates relative
// to the given painter, which must be the series painter so that the chart
// padding and the hidden axis area are taken into account.
func (v *voronoiChart) toPixel(p *Painter, point VPoint) Point {
	rect := v.opt.Rect
	width := p.Width()
	height := p.Height()
	x := int(math.Round((point.X - rect.X0) / rect.Width() * float64(width)))
	// The data y axis points upwards while the pixel axis points downwards.
	y := int(math.Round((rect.Y1 - point.Y) / rect.Height() * float64(height)))
	if x < 0 {
		x = 0
	} else if x > width {
		x = width
	}
	if y < 0 {
		y = 0
	} else if y > height {
		y = height
	}
	return Point{
		X: x,
		Y: y,
	}
}

func (v *voronoiChart) render(result *defaultRenderResult, cells []VoronoiCell) (Box, error) {
	opt := v.opt
	theme := opt.Theme
	p := result.seriesPainter

	fillAlpha := opt.FillAlpha
	if fillAlpha == 0 {
		fillAlpha = 90
	}
	borderWidth := opt.BorderWidth
	if borderWidth == 0 {
		borderWidth = 1
	}
	borderColor := opt.BorderColor
	if borderColor.IsZero() {
		borderColor = theme.GetAxisStrokeColor()
	}
	siteRadius := opt.SiteRadius
	if siteRadius == 0 {
		siteRadius = 4
	}
	fontSize := opt.LabelFontSize
	if fontSize == 0 {
		fontSize = labelFontSize
	}
	labelColor := opt.LabelColor
	if labelColor.IsZero() {
		labelColor = theme.GetTextColor()
	}

	// First pass: fill every region. Adjacent regions share their borders,
	// so the fills alone cover the whole view without overlap or gaps.
	for _, cell := range cells {
		if cell.Duplicate || len(cell.Polygon) < 3 {
			continue
		}
		site := opt.Sites[cell.Index]
		fillColor := site.FillColor
		if fillColor.IsZero() {
			fillColor = theme.GetSeriesColor(cell.Index)
		}
		points := make([]Point, 0, len(cell.Polygon))
		for _, vertex := range cell.Polygon {
			points = append(points, v.toPixel(p, vertex))
		}
		p.OverrideDrawingStyle(Style{
			FillColor: fillColor.WithAlpha(fillAlpha),
		}).FillArea(points)
	}

	// Second pass: draw the border of every region. The borders are shared
	// geometrically and are therefore overdrawn on top of each other, which
	// produces a single deterministic line on both PNG and SVG outputs.
	if opt.ShowBorder {
		for _, cell := range cells {
			if cell.Duplicate || len(cell.Polygon) < 3 {
				continue
			}
			site := opt.Sites[cell.Index]
			color := borderColor
			if !site.StrokeColor.IsZero() {
				color = site.StrokeColor
			}
			points := make([]Point, 0, len(cell.Polygon)+1)
			for _, vertex := range cell.Polygon {
				points = append(points, v.toPixel(p, vertex))
			}
			points = append(points, points[0])
			p.OverrideDrawingStyle(Style{
				StrokeColor: color,
				StrokeWidth: borderWidth,
			}).LineStroke(points)
		}
	}

	// Third pass: render the site points on top of the regions.
	if opt.ShowSite {
		for _, cell := range cells {
			if cell.Duplicate {
				continue
			}
			color := opt.SiteColor
			if color.IsZero() {
				site := opt.Sites[cell.Index]
				color = site.FillColor
				if color.IsZero() {
					color = theme.GetSeriesColor(cell.Index)
				}
			}
			point := v.toPixel(p, cell.Site)
			p.OverrideDrawingStyle(Style{
				FillColor:   color,
				StrokeColor: color,
				StrokeWidth: 1,
			})
			p.Circle(siteRadius, point.X, point.Y)
			p.FillStroke()
		}
	}

	// Fourth pass: render the labels on top of everything. Duplicate sites
	// have no region, hence no label, which also avoids overlapping text.
	if opt.ShowLabel {
		p.OverrideTextStyle(Style{
			FontColor: labelColor,
			FontSize:  fontSize,
			Font:      opt.Font,
		})
		for _, cell := range cells {
			if cell.Duplicate {
				continue
			}
			site := opt.Sites[cell.Index]
			if site.Name == "" {
				continue
			}
			point := v.toPixel(p, cell.Site)
			textBox := p.MeasureText(site.Name)
			textX := point.X - textBox.Width()/2
			textY := point.Y + textBox.Height()/2
			// Keep the label inside of the series area so that sites on the
			// view boundary never render a clipped label.
			if textX < 0 {
				textX = 0
			} else if maxX := p.Width() - textBox.Width(); textX > maxX {
				textX = maxX
			}
			if textY < textBox.Height() {
				textY = textBox.Height()
			} else if maxY := p.Height(); textY > maxY {
				textY = maxY
			}
			p.Text(site.Name, textX, textY)
		}
	}

	return v.p.box, nil
}

// Render renders the bounded Voronoi chart.
func (v *voronoiChart) Render() (Box, error) {
	opt := v.opt
	sites := make([]VPoint, len(opt.Sites))
	for index, site := range opt.Sites {
		sites[index] = VPoint{
			X: site.X,
			Y: site.Y,
		}
	}
	cells, err := BoundedVoronoi(sites, opt.Rect)
	if err != nil {
		return BoxZero, err
	}
	renderResult, err := defaultRender(v.p, defaultRenderOption{
		Theme:      opt.Theme,
		Padding:    opt.Padding,
		SeriesList: NewVoronoiSeriesList(opt.Sites),
		XAxis: XAxisOption{
			Show: FalseFlag(),
		},
		YAxisOptions: []YAxisOption{
			{
				Show: FalseFlag(),
			},
		},
		TitleOption:  opt.Title,
		LegendOption: opt.Legend,
	})
	if err != nil {
		return BoxZero, err
	}
	return v.render(renderResult, cells)
}

// VoronoiRender renders a bounded Voronoi chart from the sites and rect.
func VoronoiRender(sites []VoronoiSite, rect VoronoiRect, opts ...OptionFunc) (*Painter, error) {
	opt := ChartOption{}
	for _, fn := range opts {
		fn(&opt)
	}
	if opt.Type == "" {
		opt.Type = ChartOutputSVG
	}
	opt.fillDefault()
	if opt.BackgroundColor.IsZero() {
		opt.BackgroundColor = opt.theme.GetBackgroundColor()
	}
	p, err := NewPainter(PainterOptions{
		Type:   opt.Type,
		Width:  opt.Width,
		Height: opt.Height,
		Font:   opt.font,
	})
	if err != nil {
		return nil, err
	}
	p.SetBackground(p.Width(), p.Height(), opt.BackgroundColor)
	_, err = NewVoronoiChart(p, VoronoiChartOption{
		Theme:           opt.theme,
		Font:            opt.font,
		Padding:         opt.Padding,
		Title:           opt.Title,
		Legend:          opt.Legend,
		Sites:           sites,
		Rect:            rect,
		BackgroundColor: opt.BackgroundColor,
	}).Render()
	if err != nil {
		return nil, err
	}
	return p, nil
}

// VoronoiOptionRender renders a bounded Voronoi chart from the full option.
func VoronoiOptionRender(opt VoronoiChartOption) (*Painter, error) {
	if opt.Type == "" {
		opt.Type = ChartOutputSVG
	}
	if opt.Width <= 0 {
		opt.Width = defaultChartWidth
	}
	if opt.Height <= 0 {
		opt.Height = defaultChartHeight
	}
	if opt.Font == nil {
		opt.Font, _ = GetDefaultFont()
	}
	if opt.Theme == nil {
		opt.Theme = defaultTheme
	}
	if opt.Padding.IsZero() {
		opt.Padding = Box{
			Top:    20,
			Right:  20,
			Bottom: 20,
			Left:   20,
		}
	}
	p, err := NewPainter(PainterOptions{
		Type:   opt.Type,
		Width:  opt.Width,
		Height: opt.Height,
		Font:   opt.Font,
	})
	if err != nil {
		return nil, err
	}
	if opt.BackgroundColor.IsZero() {
		opt.BackgroundColor = opt.Theme.GetBackgroundColor()
	}
	p.SetBackground(p.Width(), p.Height(), opt.BackgroundColor)
	_, err = NewVoronoiChart(p, opt).Render()
	if err != nil {
		return nil, err
	}
	return p, nil
}
