package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

type MeasurementResult struct {
	name  string
	count uint
	sum   float64
	min   float64
	max   float64
}

type Measurement struct {
	name  string
	value float64
}

var routineMap = make(map[rune]*Routine)

func main() {
	for i := 'a'; i < 'z'; i++ {
		r := Routine{
			inCh: make(chan Measurement, math.MaxInt16),
			out:  make(chan *[](*MeasurementResult)),
		}

		routineMap[i] = &r
		go calcMeasurements(r.inCh, r.out)
	}

	// this calculates all values for any non alphabetic character
	r := Routine{
		inCh: make(chan Measurement, 100),
		out:  make(chan *[]*MeasurementResult),
	}
	routineMap[-1] = &r
	go calcMeasurements(r.inCh, r.out)

	calcChan := make(chan Measurement, math.MaxInt16)
	var wg sync.WaitGroup

    for range 7 {
        wg.Add(1)
        go splitMeasurements(calcChan, &wg)
    }

	file, err := os.Open("../measurements.txt")
	if err != nil {
		panic(err.Error())
	}
	defer file.Close()

	filescanner := bufio.NewScanner(file)
	filescanner.Split(bufio.ScanLines)

	for filescanner.Scan() {
		text := filescanner.Text()
		input := strings.Split(text, ";")

		val, err := strconv.ParseFloat(input[1], 64)
		if err != nil {
			panic(err.Error())
		}

		m := Measurement{
			name:  input[0],
			value: val,
		}

		calcChan <- m
	}

	close(calcChan)
	wg.Wait()

	var pringWg sync.WaitGroup
	pringWg.Add(1)
    var writeChan = make(chan *[](*MeasurementResult), 26)
	go PrintResults(writeChan, &pringWg)
	for _, r := range routineMap {
		close(r.inCh)
		output := <-r.out
		writeChan <- output
		close(r.out)
	}

    close(writeChan)
    pringWg.Wait()
}

type Routine struct {
	inCh chan Measurement
	out chan *[](*MeasurementResult)

}

func splitMeasurements(inputCh chan Measurement, wg *sync.WaitGroup) {
	defer wg.Done()

	for m := range inputCh {
		firstLetter := unicode.ToLower(rune(m.name[0]))
		r := routineMap[firstLetter]
		if r == nil {
			rest := routineMap[-1]
			rest.inCh <- m
		} else {
			r.inCh <- m
		}
	}
}

func PrintResults(inp chan *[](*MeasurementResult), wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Printf("{")
	for mr := range inp {
		for _, res := range *mr {
			fmt.Printf("%s, ", res.ToResultString())
		}
	}
	fmt.Printf("}\n")
}

func calcMeasurements(inputCh chan Measurement, outCh chan *[](*MeasurementResult)) {
	results := make(map[string]*MeasurementResult)

	for m := range inputCh {
		if results[m.name] == nil {
			results[m.name] = &MeasurementResult{
				name:  m.name,
				sum:   m.value,
				min:   m.value,
				max:   m.value,
				count: 1,
			}
		} else {
			var res *MeasurementResult = results[m.name]
			res.count++
			res.sum += m.value

			if m.value < res.min {
				res.min = m.value
			}

			if m.value > res.max {
				res.max = m.value
			}
		}
	}

	keys := make([]string, 0, len(results))
	for k := range results {
		keys = append(keys, k)
	}
	sort.Sort(sort.StringSlice(keys))

	resultList := make([](*MeasurementResult), 0, len(results))
	for _, k := range keys {
		resultList = append(resultList, results[k])
	}

	outCh <- &resultList
}

func (m *MeasurementResult) ToResultString() string {
	return fmt.Sprintf("%s=%.1f/%.1f/%.1f", m.name, m.min, m.sum/float64(m.count), m.max)
}
