package main

import (
	"EvenDigits/common"
	"EvenDigits/mp"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"runtime"
	"runtime/pprof"
	"slices"
	"sync"
	"time"
)

/*
Tests the hypothesis that there are only four values of n where 2^n has all even digits
by direct examination.

This is suitable for testing several billions of values, but this is known to hold for
values up to 2^(10^10) which is much further than can be tested with this program.

This program uses the fact that there are typically less than 25 even digits for
any value of n in the range of this program. That means we can compute 2^n mod
mask where mask is 10^35 or so. This is good since 2^(10^9) has 300 million
digits so the computation would become very expensive. Furthermore, it is known
that n mod 20 must be 3, 6, 11, or 19 for n > 2. This decreases the number of
cases we need to examine by a further factor of 5.
*/
var (
	zero = mp.UInt256{}
	two  = mp.UInt256{[8]uint64{2}}
)

type LoopAccelerator struct {
	Mask      uint64
	Order     int
	Length    uint64
	Leadin    uint64
	EvenItems int
	Gain      float64
	Index     []uint64
}

type Configuration struct {
	mu      sync.Mutex
	Steps   []uint64
	Bumps   []mp.UInt256
	Mask    mp.UInt256
	Verbose bool
}

type Result struct {
	ID        int
	Success   bool
	Solutions []uint64
	Records   []Record
	MaxEven   int
	Tests     int
}

type Record struct {
	Z      uint64
	Digits int
}

func main() {
	verbose := flag.Bool("verbose", false, "verbose output")
	digits := flag.Int("digits", 50, "Number of digits to use in search")
	threads := flag.Int("threads", runtime.NumCPU()/2, "Number of threads to use in search")
	sieve := flag.String("sieve", "cycle-012.json", "JSON file containing a sieve definition")
	limitString := flag.String("limit", "10G", "Maximum value of N to search. Can use M, G, T, P and E as power of ten")
	cpuProfile := flag.String("cpuprofile", "", "write cpu profile to file")
	memProfile := flag.String("memprofile", "", "write memory profile to file")
	flag.Parse()

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			log.Fatal(err)
			return
		}
		defer pprof.StopCPUProfile()
	}
	defer func() {
		if *memProfile != "" {
			f, err := os.Create(*memProfile)
			if err != nil {
				log.Fatal(err)
			}
			runtime.GC()
			err = pprof.WriteHeapProfile(f)
			if err != nil {
				log.Fatal(err)
			}
			_ = f.Close()
		}
	}()

	limit := common.DecodeLimit(limitString, verbose)

	mask := mp.UInt256{[8]uint64{1}}
	for i := 0; i < *digits; i++ {
		mask.MulSmall(10)
	}

	config, err := readAccelerator(*sieve)
	if err != nil {
		log.Fatal(err)
	}

	steps := []uint64{}
	c0 := uint64(0)
	for _, c := range config.Index {
		steps = append(steps, c-c0)
		c0 = c
	}
	steps = append(steps, config.Length-c0)

	bumps := make([]mp.UInt256, len(steps))
	for i, step := range steps {
		bumps[i] = two
		bumps[i].Pow256(mp.UInt256{[8]uint64{step}}, mask)
	}

	conf := Configuration{
		Verbose: *verbose,
		Steps:   steps,
		Bumps:   bumps,
		Mask:    mask,
	}

	solutions := []uint64{}
	z := two
	for n := uint64(1); n <= config.Leadin; n++ {
		if even := checkDigits(z); even == -1 {
			solutions = append(solutions, n)
		}
		z.MulMod(two, mask)
	}
	t0 := time.Now()

	totalBatches := (limit + config.Length - 1) / config.Length

	dispatch := make(chan uint64, *threads)
	go dispatcher(totalBatches, dispatch, *verbose)

	fmt.Printf("%d threads\n", *threads)
	results := make(chan Result, *threads)
	for i := 0; i < *threads; i++ {
		go worker(i, dispatch, &conf, config, results)
	}

	tests := 0
	records := []Record{}
	for i := 0; i < *threads; i++ {
		r, ok := <-results
		if !ok {
			log.Fatalf("Results channel closed ... should be impossible")
		}
		log.Printf("thread %d result (max = %d)\n", r.ID, r.MaxEven)
		if !r.Success {
			log.Fatalf("Worker %d failed", r.ID)
		}

		for _, solution := range r.Solutions {
			solutions = append(solutions, solution)
		}
		for _, record := range r.Records {
			records = append(records, record)
		}
		tests += r.Tests
	}
	fr, err := os.OpenFile("records.json", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer func(fr *os.File) {
		_ = fr.Close()
	}(fr)

	slices.Sort(solutions)
	slices.SortFunc(records, func(a, b Record) int {
		return b.Digits - a.Digits
	})
	txt, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	_, _ = fr.Write(txt)
	_ = fr.Close()
	dt := time.Since(t0).Seconds()
	fmt.Printf("%.1f test/s, total time %.1f s\n", float64(limit)/dt, dt)
	fmt.Printf("Limit: %d\nTests: %d\n", limit, tests)
	fmt.Printf("Gain over brute: %f.1\n", float64(limit)/float64(tests))
	fmt.Printf("solutions = %v\n", solutions)
}

// worker is where the actual testing happens
func worker(thread int, dispatch chan uint64, conf *Configuration, config LoopAccelerator, results chan Result) {
	solutions := []uint64{}
	r := Result{
		ID:        thread,
		Success:   false,
		Solutions: solutions,
		Records:   []Record{},
		MaxEven:   0,
		Tests:     0,
	}
	defer func() {
		results <- r
	}()

	conf.mu.Lock()
	mask := conf.Mask

	bumps := make([]mp.UInt256, len(conf.Bumps))
	for i, bump := range conf.Bumps {
		bumps[i] = bump
	}

	steps := make([]uint64, len(conf.Steps))
	for i, step := range conf.Steps {
		steps[i] = step
	}
	conf.mu.Unlock()

	cycleSize := len(steps) - 1

	jobs := 0
	n := uint64(0)
	z := mp.UInt256{[8]uint64{1}}
	for {
		var (
			job uint64
			ok  bool
		)

		job, ok = <-dispatch
		jobs++
		if !ok {
			if conf.Verbose {
				log.Printf("breaking %d\n", thread)
			}
			break
		}
		next := job * config.Length
		tmp := two
		tmp.Pow256(mp.UInt256{[8]uint64{next - n}}, mask)
		z.MulMod(tmp, mask)
		n = next

		for i, dn := range steps[:cycleSize] {
			n += dn
			z.MulMod(bumps[i], mask)
			r.Tests++
			if even := checkDigits(z); even == -1 {
				r.Solutions = append(r.Solutions, n)
			} else {
				if even > r.MaxEven {
					r.MaxEven = even
					r.Records = append(r.Records, Record{
						Z:      n,
						Digits: even,
					})
				}
			}
		}
		n += steps[cycleSize]
		z.MulMod(bumps[cycleSize], mask)
	}
	r.Success = true
	if conf.Verbose {
		log.Printf("exiting %d\n", thread)
	}
}

// dispatcher sends small batches of work to the workers via a channel
// each work is iteration through the repetition cycle we got from the
// cycle detector program.
func dispatcher(totalBatches uint64, dispatch chan uint64, verbose bool) {
	step := (totalBatches + 19) / 20
	t0 := time.Now()
	tick := time.NewTicker(time.Second)
	lastReport := time.Now()
	startTime := time.Now()
	normalReporting := false
	for i := uint64(0); i < totalBatches; {
		report := func() {
			t1 := time.Now()
			total := t1.Sub(t0).Seconds()
			dt := (total + 0.5) / float64(i+1)

			log.Printf(
				"sender: %6d (%10.0f%%, %.1f %.1f) %.1f seconds remaining",
				i,
				float64(i*100)/float64(totalBatches),
				dt*1000, total*1000,
				float64(totalBatches-i)*dt,
			)

		}

		select {
		case <-tick.C:
			if normalReporting {
				continue
			}
			total := time.Since(startTime).Seconds()
			recent := time.Since(lastReport).Seconds()
			interval := math.Min(30.0, math.Max(5, total/2.5))
			//fmt.Printf("total: %.1f, recent: %.1f, interval: %.1f\n", total, recent, interval)
			if verbose && recent >= interval {
				report()
				lastReport = time.Now()
			}
		case dispatch <- i:
			i++
			if verbose && i%step == 0 {
				if normalReporting || time.Since(lastReport).Seconds() > 5 {
					report()
				}
				normalReporting = true
			}
		}
	}
	if verbose {
		log.Printf("sender: completed")
	}
	close(dispatch)
}

func readAccelerator(name string) (LoopAccelerator, error) {
	config := LoopAccelerator{}

	f, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	txt, err := io.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	_ = f.Close()
	err = json.Unmarshal(txt, &config)
	if err != nil {
		log.Fatal(err)
	}
	return config, err
}

// checkDigits returns -1 if all of the digits in z are even. If not, the
// position counting from the right is returned.
func checkDigits(z mp.UInt256) int {
	for j := 0; z.Cmp(zero) > 0; j++ {
		digit := z.DivModSmall(10)
		if digit%2 == 1 {
			return j
		}
	}
	return -1
}
