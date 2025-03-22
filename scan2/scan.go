package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"runtime"
	"slices"
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
	zero = big.NewInt(0)
	two  = big.NewInt(2)
	ten  = big.NewInt(10)
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
	Steps []uint64
	Bumps []*big.Int
	Mask  *big.Int
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
	mask := big.NewInt(1)
	for i := 0; i < 55; i++ {
		mask.Mul(mask, ten)
	}

	config, err := readAccelerator("cycle-009.json")

	steps := []uint64{}
	c0 := uint64(0)
	for _, c := range config.Index {
		steps = append(steps, c-c0)
		c0 = c
	}
	steps = append(steps, config.Length-c0)

	bumps := make([]*big.Int, len(steps))
	tmp := big.NewInt(0)
	for i, step := range steps {
		bumps[i] = big.NewInt(0)
		tmp.SetUint64(2)
		pow(bumps[i], tmp, step, mask)
	}

	conf := Configuration{
		Steps: steps,
		Bumps: bumps,
		Mask:  mask,
	}

	solutions := []uint64{}
	z := big.NewInt(2)
	for n := uint64(1); n <= config.Leadin; n++ {
		if even := checkDigits(z); even == -1 {
			solutions = append(solutions, n)
		}
		z.Mul(z, two).Mod(z, mask)
	}
	t0 := time.Now()

	limit := uint64(10_000_000_000)
	totalBatches := (limit + config.Length - 1) / config.Length

	dispatch := make(chan uint64, 2)
	go dispatcher(totalBatches, dispatch)

	fmt.Printf("%d threads\n", runtime.NumCPU()/4)
	results := make(chan Result)
	for i := 0; i < runtime.NumCPU()/4; i++ {
		go worker(i, dispatch, conf, config, results)
	}

	tests := 0
	records := []Record{}
	for i := 0; i < runtime.NumCPU()/4; i++ {
		r, ok := <-results
		if !ok {
			log.Fatalf("Results channel closed ... should be impossible")
		}
		log.Printf("thread %d result\n", r.ID)
		if !r.Success {
			log.Fatalf("Workeder %d failed", r.ID)
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
	defer fr.Close()

	slices.Sort(solutions)
	slices.SortFunc(records, func(a, b Record) int {
		return b.Digits - a.Digits
	})
	txt, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fr.Write(txt)
	fr.Close()
	dt := time.Since(t0).Seconds()
	fmt.Printf("%.1f test/s, total time %.1f s\n", float64(limit)/dt, dt)
	fmt.Printf("Limit: %d\nTests: %d\n", limit, tests)
	fmt.Printf("Gain over brute: %f.1\n", float64(limit)/float64(tests))
	fmt.Printf("solutions = %v\n", solutions)
}

// worker is where the actual testing happens
func worker(thread int, dispatch chan uint64, conf Configuration, config LoopAccelerator, results chan Result) {
	solutions := []uint64{}
	records := []Record{}
	r := Result{
		ID:        thread,
		Success:   false,
		Solutions: solutions,
		Records:   records,
		MaxEven:   0,
		Tests:     0,
	}
	defer func() {
		results <- r
	}()

	cycleSize := len(conf.Steps) - 1

	jobs := 0
	n := uint64(0)
	z := big.NewInt(1)
	tmp1 := big.NewInt(0)
	tmp2 := big.NewInt(0)
	for {
		job, ok := <-dispatch
		jobs++
		if !ok {
			log.Printf("breaking %d\n", thread)
			break
		}
		next := job * config.Length
		tmp1.SetUint64(1)
		tmp2.SetUint64(2)
		pow(tmp1, tmp2, next-n, conf.Mask)
		z.Mul(z, tmp1)
		n = next

		for i, dn := range conf.Steps[:cycleSize] {
			n += dn
			z.Mul(z, conf.Bumps[i]).Mod(z, conf.Mask)
			r.Tests++
			if even := checkDigits(z); even == -1 {
				solutions = append(solutions, n)
			} else {
				if even > r.MaxEven {
					r.MaxEven = even
					records = append(records, Record{
						Z:      n,
						Digits: even,
					})
				}
			}
		}
		n += conf.Steps[cycleSize]
		z.Mul(z, conf.Bumps[cycleSize]).Mod(z, conf.Mask)
	}
	r.Success = true
	log.Printf("exiting %d\n", thread)
}

// dispatcher sends small batches of work to the workers via a channel
// each work is iteration through the repetition cycle we got from the
// cycle detector program.
func dispatcher(totalBatches uint64, dispatch chan uint64) {
	step := (totalBatches + 99) / 100
	t0 := time.Now()
	for i := uint64(0); i < totalBatches; i++ {
		if i%step == 0 {
			t1 := time.Now()
			total := t1.Sub(t0).Seconds()
			dt := (total + 0.5) / float64(i+1)

			log.Printf(
				"sender: %6d (%.0f%%, %.1f %.1f) %.1f seconds remaining",
				i,
				float64(i*100)/float64(totalBatches),
				dt*1000, total*1000,
				float64(totalBatches-i)*dt,
			)
		}
		dispatch <- i
	}
	log.Printf("sender: completed")
	close(dispatch)
}

func readAccelerator(name string) (LoopAccelerator, error) {
	config := LoopAccelerator{}

	f, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
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
func checkDigits(z *big.Int) int {
	digit := big.NewInt(0)
	zdig := big.NewInt(0)
	zdig.Add(z, zero)
	for j := 0; zdig.Cmp(zero) > 0; j++ {
		zdig, digit = zdig.QuoRem(zdig, ten, digit)
		if digit.Uint64()%2 != 0 {
			return j
		}
	}
	return -1
}

// pow sets `z = m^n mod mask`. The value of m is destroyed in the process.
func pow(z *big.Int, m *big.Int, n uint64, mask *big.Int) {
	z.SetInt64(1)
	if n == 0 {
		return
	}
	for n > 0 {
		if n%2 == 1 {
			z.Mul(z, m)
			z.Mod(z, mask)
		}
		n = n / 2
		m.Mul(m, m)
	}
}
