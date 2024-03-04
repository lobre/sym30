package main

import (
	"bufio"
	"errors"
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

type stats struct {
	title string

	unigrams map[string]int
	bigrams  map[string]int
}

func newStats(title string) stats {
	var s stats
	s.title = title
	s.unigrams = make(map[string]int)
	s.bigrams = make(map[string]int)
	return s
}

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

	stats := newStats(title)

	fmt.Println("Calculating statistics...")

	if err := stats.calculate(input); err != nil {
		return err
	}

	fmt.Println("Generating graph for unigrams...")

	path, err := stats.graphUnigrams()
	if err != nil {
		return err
	}

	fmt.Printf("Graph for unigrams has been generated at: %s\n", path)

	fmt.Println("Generating graph for bigrams...")

	path, err = stats.graphBigrams()
	if err != nil {
		return err
	}

	fmt.Printf("Graph for bigrams has been generated at: %s\n", path)

	return nil
}

func stripExtension(filename string) string {
	return filepath.Base(filename[:len(filename)-len(filepath.Ext(filename))])
}

func isSymbol(c rune) bool {
	return strings.ContainsRune("^<>$%{()}=~[]_#@&*'`\\+-/\"|!;:?", c)
}

func (s stats) calculate(input io.Reader) error {
	r := bufio.NewReader(input)

	var prev *rune

	for {
		c, _, err := r.ReadRune()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if unicode.IsSpace(c) && c != '\n' && c != '\r' {
			continue
		}

		if !isSymbol(c) && !unicode.IsDigit(c) {
			prev = nil // reset
			continue
		}

		if !unicode.IsDigit(c) {
			s.unigrams[string(c)] += 1
		}

		if prev != nil {
			// do not count double digits
			if unicode.IsDigit(*prev) && unicode.IsDigit(c) {
				prev = &c
				continue
			}

			s.bigrams[fmt.Sprintf("%c%c", *prev, c)] += 1
		}

		prev = &c
	}

	return nil
}

func (s stats) graphUnigrams() (string, error) {
	path := s.title + "_unigrams.png"

	file, err := os.Create(path)
	if err != nil {
		return path, err
	}
	defer file.Close()

	err = graph(s.title+" unigrams", s.unigrams, file)
	if err != nil {
		return path, err
	}

	return path, nil
}

func (s stats) graphBigrams() (string, error) {
	path := s.title + "_bigrams.png"

	file, err := os.Create(path)
	if err != nil {
		return path, err
	}
	defer file.Close()

	err = graph(s.title+" bigrams", s.bigrams, file)
	if err != nil {
		return path, err
	}

	return path, nil
}

func graph(title string, data map[string]int, out io.Writer) error {
	if len(data) == 0 {
		return errors.New("cannot graph as data is empty")
	}

	p := plot.New()
	p.Title.Text = title

	keys := sortedKeys(data)
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

func sortedKeys(input map[string]int) []string {
	keys := make([]string, 0, len(input))

	for key := range input {
		keys = append(keys, key)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return input[keys[i]] > input[keys[j]]
	})

	return keys
}
