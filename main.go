package main

import (
	"bufio"
	"fmt"
	"image/color"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

func main() {
	if err := run(os.Args, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) != 2 {
		return fmt.Errorf("Usage: %s <filename>", args[0])
	}

	filename := args[1]
	title := stripExtension(filename)

	input, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer input.Close()

	fmt.Println("Calculating unigrams...")

	uniData, err := unigrams(input)
	if err != nil {
		return err
	}

	uniPath := fmt.Sprintf("%s_unigrams.png", title)
	uniTitle := fmt.Sprintf("%s unigrams", title)

	uniOut, err := os.Create(uniPath)
	if err != nil {
		return err
	}
	defer uniOut.Close()

	err = graph(uniTitle, uniData, uniOut)
	if err != nil {
		return err
	}

	fmt.Printf("Graph %s has been generated\n", uniPath)

	fmt.Println("Calculating bigrams...")

	biData, err := bigrams(input)
	if err != nil {
		return err
	}

	fmt.Println("heeere")

	biPath := fmt.Sprintf("%s_bigrams.png", title)
	biTitle := fmt.Sprintf("%s bigrams", title)

	biOut, err := os.Create(biPath)
	if err != nil {
		return err
	}
	defer biOut.Close()

	err = graph(biTitle, biData, biOut)
	if err != nil {
		return err
	}

	fmt.Printf("Graph %s has been generated\n", biPath)

	return nil
}

func stripExtension(filename string) string {
	return filepath.Base(filename[:len(filename)-len(filepath.Ext(filename))])
}

func isSymbol(c rune) bool {
	return strings.ContainsRune("^<>$%{()}=~[]_#@&*'`\\+-/\"|!;:?", c)
}

func unigrams(f io.Reader) (map[string]int, error) {
	r := bufio.NewReader(f)

	res := make(map[string]int)

	for {
		c, _, err := r.ReadRune()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		if isSymbol(c) {
			res[string(c)] += 1
		}
	}

	return res, nil
}

func bigrams(f io.Reader) (map[string]int, error) {
	r := bufio.NewReader(f)

	res := make(map[string]int)
	var prev *rune

	for {
		c, _, err := r.ReadRune()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		if unicode.IsSpace(c) && c != '\n' && c != '\r' {
			continue
		}

		if !isSymbol(c) && !unicode.IsDigit(c) {
			prev = nil // reset
			continue
		}

		if prev != nil {
			// do not count double digits
			if unicode.IsDigit(*prev) && unicode.IsDigit(c) {
				prev = &c
				continue
			}

			res[fmt.Sprintf("%c%c", *prev, c)] += 1
		}

		prev = &c
	}

	return res, nil
}

func graph(title string, data map[string]int, out io.Writer) error {
	p := plot.New()
	p.Title.Text = title

	// sort by frequency in descending order
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return data[keys[i]] > data[keys[j]]
	})

	xyData := make(plotter.XYs, len(keys))

	for i, key := range keys {
		xyData[i].X = float64(i)
		xyData[i].Y = float64(data[key])
	}

	line, err := plotter.NewLine(xyData)
	if err != nil {
		return err
	}

	line.LineStyle.Width = vg.Points(2)     // Set line thickness
	line.Color = color.RGBA{0, 0, 255, 255} // Set line color to blue (0, 0, 255)

	p.Add(plotter.NewGrid())
	p.Add(line)
	p.NominalX(keys...)
	p.X.Label.Text = "Symbol"
	p.Y.Label.Text = "Frequency"
	p.Y.Min = 0

	// ticks on the Y-axis every 50 up to the maximum value
	var ticks []plot.Tick
	for i := 0; i <= data[keys[0]]; i += 50 {
		ticks = append(ticks, plot.Tick{Value: float64(i), Label: fmt.Sprintf("%d", i)})
	}
	p.Y.Tick.Marker = plot.ConstantTicks(ticks)

	// display frequency below each symbol on the horizontal axis
	var xTicks []plot.Tick
	for i, key := range keys {
		xTicks = append(xTicks, plot.Tick{Value: float64(i), Label: fmt.Sprintf("%s\n%d", key, data[key])})
	}
	p.X.Tick.Marker = plot.ConstantTicks(xTicks)

	c := vgimg.PngCanvas{
		Canvas: vgimg.New(8*vg.Inch, 4*vg.Inch),
	}

	p.Draw(draw.New(c))

	_, err = c.WriteTo(out)
	if err != nil {
		return err
	}

	return nil
}
