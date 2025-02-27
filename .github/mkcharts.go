package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"andy.dev/redo/backoff"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

const (
	minD = time.Second
	// maxD = 5 * time.Second
	maxSeconds = 5
	maxD       = maxSeconds * time.Second

	totalSamples = 200_000
	slotsPerSec  = 10
	totalSlots   = maxSeconds * slotsPerSec
	triesPer     = 5
)

type (
	backoffFn    func() time.Duration
	backoffGraph struct {
		fn      func() backoffFn
		short   string
		algo    string
		samples []float64
	}
)

func rndBetween(a, b float64) float64 {
	return a + rand.Float64()*(b-a)
}

func ALGExpoJitter(initialDelay, maxDelay time.Duration) func() time.Duration {
	seed := float64(initialDelay)
	max := float64(maxDelay)
	try := 0
	return func() time.Duration {
		try++
		return time.Duration(rndBetween(0, min(max, math.Pow(seed*2, float64(try)))))
		// return time.Duration(rndBetween(0, seed*math.Pow(2, float64(try))))
	}
}

func ALGDecorrJitter(initialDelay, maxDelay time.Duration) func() time.Duration {
	seed := float64(initialDelay.Milliseconds())
	maxD := float64(maxDelay.Milliseconds())
	current := seed
	return func() time.Duration {
		current = min(maxD, rndBetween(seed, current*3*rand.Float64()))
		// return time.Duration(current * float64(time.Millisecond))
		return time.Duration(current * float64(time.Millisecond))
	}
}

func ALGSoftExpSoftBackoff(initial time.Duration, max time.Duration) func() time.Duration {
	iter := backoff.New(initial, max, false)
	return func() time.Duration {
		return iter()
	}
}

func main() {
	rand.Seed(4321)
	log.SetFlags(log.Lshortfile)
	bg := []backoffGraph{
		{
			algo:  "SoftExpSoftDelay",
			short: "softexpd",
			fn: func() backoffFn {
				return ALGSoftExpSoftBackoff(minD, maxD)
			},
		},
		{
			algo:  "Decorrelated Jitter",
			short: "decorr",
			fn: func() backoffFn {
				return ALGDecorrJitter(minD, maxD)
			},
		},
		{
			algo:  "Exponential Backoff w/ Jitter",
			short: "expo",
			fn: func() backoffFn {
				return ALGExpoJitter(minD, maxD)
			},
		},
	}

	// iter := backoff.New(1*time.Second, 20*time.Minute, true)
	// for {
	// 	dur := iter()
	// 	fmt.Printf("%v\n", dur)
	// 	// if dur >= 10*time.Minute {
	// 	// 	os.Exit(0)
	// 	// }
	// 	time.Sleep(500 * time.Millisecond)
	// }

	makeLines(bg)
	makeHistograms(bg)
}

func makeLines(bgs []backoffGraph) {
	p := plot.New()
	p.Title.Text = ""
	p.X.Label.Text = fmt.Sprintf(
		"Delays Across %d retries with cutoff at %d seconds",
		triesPer, maxSeconds,
	)
	p.X.AutoRescale = true
	p.Y.AutoRescale = true
	p.Y.Max = 0.15
	p.Y.Label.Text = "Percentage of total calls"
	p.Y.Tick.Marker = pctTicks{}
	p.X.Tick.Marker = secTics()

	for gi, g := range bgs {
		samples := make(samplePlotter, totalSlots)
		for i := 0; i < totalSamples; i++ {
			t := 0.0
			backoff := g.fn()
			for j := 0; j < 30; j++ {
				t += backoff().Seconds()
				x := int(t*slotsPerSec + 1)
				if x >= 0 && x < totalSlots {
					samples[x] += 1.0 / totalSamples
				}
			}
		}

		l, err := plotter.NewLine(samples)
		if err != nil {
			log.Fatal(err)
		}
		l.LineStyle.Width = vg.Points(1)
		// l.LineStyle.Dashes = []vg.Length{vg.Points(5), vg.Points(5)}
		l.LineStyle.Color = plotutil.Color(gi)

		p.Add(l)
		p.Legend.Add(g.algo, l)
		p.Legend.Top = true
		ms := localMaxima(l, 0.0055)
		if ms != nil {
			pts, err := plotter.NewScatter(ms)
			if err != nil {
				log.Fatal(err)
			}
			pts.GlyphStyle = draw.GlyphStyle{
				Color: color.RGBA{
					R: 0,
					G: 0,
					B: 0,
					A: 255,
				},
				Radius: 4,
				Shape:  draw.TriangleGlyph{},
			}
			p.Add(pts)
		}
	}
	file := chartname("dists")
	fmt.Println(file)
	if err := p.Save(8*vg.Inch, 4*vg.Inch, file); err != nil {
		log.Fatal(err)
	}
}

func makeHistograms(bgs []backoffGraph) {
	const cols = 4

	samples := make(valuePlotter, totalSamples*triesPer)

	// maybe try full hist again, but cutoff at 20s?
	// also try and make all x axes the same scale for tries hists
	for _, g := range bgs {
		for i := 0; i < totalSamples; i++ {
			subset := samples[i*triesPer : i*triesPer+triesPer]
			backoff := g.fn()
			for j := 0; j < triesPer; j++ {
				subset[j] = backoff().Seconds()
			}
		}

		fullHist, err := plotter.NewHist(samples, totalSlots)
		if err != nil {
			log.Fatal(err)
		}
		fullHist.Normalize(1)
		fullplot := plot.New()
		fullplot.Add(fullHist)
		file := chartname(g.short, "hist", "full")
		if err := fullplot.Save(8*vg.Inch, 4*vg.Inch, file); err != nil {
			log.Fatal(err)
		}

		plots := make([][]*plot.Plot, numRows(cols, triesPer))
		for i := range plots {
			plots[i] = make([]*plot.Plot, cols)
		}
		for i := 0; i < triesPer; i++ {
			trySamples := samples.Subset(triesPer, i)
			h, err := plotter.NewHist(trySamples, totalSlots)
			if err != nil {
				log.Fatal(err)
			}
			h.Normalize(100)
			p := plot.New()
			p.Add(h)
			// norm := plotter.NewFunction(distuv.UnitNormal.Prob)
			// norm.Color = color.RGBA{R: 255, A: 255}
			// norm.Width = vg.Points(2)
			// p.Add(norm)
			row := i / cols
			col := i % cols
			plots[row][col] = p
		}
		img := vgimg.New(cols*4*vg.Inch, font.Length(len(plots))*4*vg.Inch)
		dc := draw.New(img)
		t := draw.Tiles{
			Rows: numRows(cols, triesPer),
			Cols: cols,
		}
		canvases := plot.Align(plots, t, dc)
		for j := 0; j < t.Rows; j++ {
			for i := 0; i < t.Cols; i++ {
				if plots[j][i] != nil {
					plots[j][i].Draw(canvases[j][i])
				}
			}
		}

		file = chartname(g.short, "hist", "tries")
		fmt.Println(file)
		w, err := os.Create(file)
		if err != nil {
			log.Fatalf("os.Create: %v", err)
		}
		_, err = vgimg.PngCanvas{Canvas: img}.WriteTo(w)
		if err != nil {
			log.Fatalf("PngCanvas.WriteTo(): %v", err)
		}
	}
}

type samplePlotter []float64

func (sp samplePlotter) Len() int {
	return len(sp)
}

func (sp samplePlotter) XY(idx int) (x, y float64) {
	return float64(idx), sp[idx]
}

type valuePlotter []float64

func (vp valuePlotter) Len() int {
	return len(vp)
}

func (vp valuePlotter) Value(i int) float64 {
	return vp[i]
}

func (vp valuePlotter) Subset(divisor, offset int) subsetValuePlotter {
	return subsetValuePlotter{
		vp: vp,
		d:  divisor,
		o:  offset,
	}
}

type subsetValuePlotter struct {
	vp valuePlotter
	d  int
	o  int
}

func (sp subsetValuePlotter) Len() int {
	return len(sp.vp) / sp.d
}

func (sp subsetValuePlotter) Value(i int) float64 {
	return sp.vp[i*sp.d+sp.o]
}

type pointPlotter []plotter.XY

func (pp pointPlotter) Len() int {
	return len(pp)
}

func (pp pointPlotter) XY(idx int) (x, y float64) {
	return pp[idx].X, pp[idx].Y
}

type pctTicks struct{}

// Ticks computes the default tick marks, but inserts commas
// into the labels for the major tick marks.
func (pctTicks) Ticks(min, max float64) []plot.Tick {
	tks := plot.DefaultTicks{}.Ticks(min, max)
	for i, t := range tks {
		if t.Label != "" {
			tks[i].Label = fmt.Sprintf("%s%%", strconv.FormatFloat(t.Value*100, 'G', -1, 64))
		}
	}
	return tks
}

type bySeconds struct{}

func secTics() plot.ConstantTicks {
	ticks := make([]plot.Tick, 0, totalSlots/slotsPerSec+1)
	for i := 1; i <= totalSlots/slotsPerSec; i++ {
		if i <= 5 || i%5 == 0 {
			ticks = append(ticks, plot.Tick{
				Value: float64(i * slotsPerSec),
				Label: fmt.Sprintf("%ds", i),
			})
		}
	}
	return ticks
}

// crude local maxima based on delta-y
func localMaxima(l *plotter.Line, dThreshold float64) plotter.XYer {
	ms := pointPlotter{}

	for i, p := range l.XYs {
		ismax := false
		cy := p.Y
		switch i {
		case 0:
			ismax = cy > l.XYs[i+1].Y
		case len(l.XYs) - 1:
			continue
		default:
			if cy > l.XYs[i-1].Y && cy > l.XYs[i+1].Y {
				diff := (cy - l.XYs[i-1].Y) + (cy - l.XYs[i+1].Y)
				if diff >= dThreshold {
					ismax = true
				}
			}
		}
		if ismax {
			pt := plotter.XY{}
			pt.X, pt.Y = l.XY(i)
			ms = append(ms, pt)
		}
	}

	if len(ms) > 0 {
		return ms
	}
	return nil
}

func chartname(parts ...any) string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("os.Getwd():", err)
	}
	fname := []byte(filepath.Join(cwd, "charts") + string(filepath.Separator))
	for i, p := range parts {
		fname = fmt.Append(fname, p)
		if i < len(parts)-1 {
			fname = fmt.Append(fname, "_")
		}
	}
	fname = fmt.Append(fname, ".png")
	return string(fname)
}

func numRows(columns, total int) int {
	m := 0
	if total%columns > 0 {
		m++
	}
	return total/columns + m
}
