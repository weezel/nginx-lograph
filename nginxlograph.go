package main

import (
	"bufio"
	"fmt"
	"image/color"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

const (
	fileOut string = "traffic.png"
)

var re = regexp.MustCompile(`[\[\]]`)

type Pair struct {
	Key   string
	Value int
}

type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

var stats = struct {
	sync.RWMutex
	clients map[string]int
}{clients: make(map[string]int)}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func sortByHits(m map[string]int) PairList {
	pl := make(PairList, len(m))
	i := 0
	for k, v := range m {
		pl[i] = Pair{k, v}
		i++
	}
	sort.Sort(sort.Reverse(pl))
	return pl
}

func countHitsPerIp(dirPath string, fname string, wg *sync.WaitGroup) {
	defer wg.Done()

	fullFilePath := filepath.Join(dirPath, fname)
	fopen, err := os.Open(fullFilePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer fopen.Close()

	scanner := bufio.NewScanner(fopen)
	for scanner.Scan() {
		splitted := strings.Split(scanner.Text(), " ")
		stats.Lock()
		stats.clients[splitted[0]]++
		stats.Unlock()
	}
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}
}

func visitsPerDay(dirPath string, fname string, wg *sync.WaitGroup) {
	defer wg.Done()

	fullFilePath := filepath.Join(dirPath, fname)
	fopen, err := os.Open(fullFilePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer fopen.Close()

	scanner := bufio.NewScanner(fopen)
	for scanner.Scan() {
		splitted := strings.Split(scanner.Text(), " ")
		rawDateTime := strings.Split(splitted[3], ":")[0] + " " + splitted[4]
		// dateTime is in %d/%b/%Y format
		dateTime := string(re.ReplaceAll([]byte(rawDateTime), []byte(``)))
		if err != nil {
			log.Panicf("Couldn't parse datetime %s: %s", dateTime, err)
		}
		stats.Lock()
		stats.clients[dateTime]++
		stats.Unlock()
	}
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}
}

func doPlotting(sortedKeys *[]string) {
	p, err := plot.New()
	check(err)

	p.Title.Text = "Site loads per day"
	p.Y.Label.Text = "Site loads"
	p.X.Label.Text = "Datetime"
	p.X.Tick.Marker = plot.TimeTicks{Format: "02/Jan/2006"}

	datePlot := make(plotter.XYs, len(*sortedKeys))
	for i, k := range *sortedKeys {
		parsedDateTime, err := time.Parse("02/Jan/2006 -0700", k)
		check(err)
		datePlot[i].X = float64(parsedDateTime.Unix())

		datePlot[i].Y = float64(stats.clients[k])
	}

	line, points, err := plotter.NewLinePoints(datePlot)
	check(err)
	line.Color = color.RGBA{G: 255, A: 255}
	points.Shape = draw.CircleGlyph{}
	points.Color = color.RGBA{R: 200, A: 255}
	p.Add(line, points)

	if err := p.Save(4*vg.Inch, 4*vg.Inch, "testi.png"); err != nil {
		panic(err)
	}
}

func histoPlotting(sortedKeys *[]string) {
	p, err := plot.New()
	check(err)

	p.Title.Text = "Site loads per day"
	p.Y.Label.Text = "Site loads"
	p.X.Label.Text = "Datetime"
	p.X.Tick.Marker = plot.TimeTicks{Format: "02/Jan/2006"}

	datePlot := make(plotter.XYs, len(*sortedKeys))
	for i, k := range *sortedKeys {
		parsedDateTime, err := time.Parse("02/Jan/2006 -0700", k)
		check(err)
		datePlot[i].X = float64(parsedDateTime.Unix())

		datePlot[i].Y = float64(stats.clients[k])
	}
	h, err := plotter.NewHistogram(datePlot, len(datePlot))
	check(err)
	h.FillColor = plotutil.Color(2)
	p.Add(h)

	if err := p.Save(4*vg.Inch, 4*vg.Inch, fileOut); err != nil {
		panic(err)
	}
}

func main() {
	var dirPath string

	if len(os.Args) < 2 {
		fmt.Printf("usage: %s: log-dir-path\n", os.Args[0])
		os.Exit(1)
	}
	dirPath = os.Args[1]

	r, err := ioutil.ReadDir(dirPath)
	check(err)

	var wg sync.WaitGroup
	wg.Add(len(r))
	for _, f := range r {
		go visitsPerDay(dirPath, f.Name(), &wg)
	}
	wg.Wait()

	var sortedKeys []string
	for k := range stats.clients {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)
	for _, k := range sortedKeys {
		fmt.Printf("%s: %d\n", k, stats.clients[k])
	}

	histoPlotting(&sortedKeys)

	fmt.Printf("Output written to %s\n", fileOut)
}
