package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/text/message"
	"log"
	"os"
	"slices"
)

/*
Scans for cycles in the low digits of powers of two. Patterns of this
sort can decimate the search for values of 2^n where all digits are even.
*/
func main() {
	p := message.NewPrinter(message.MatchLanguage("en"))

	exports := map[uint64]bool{
		10:                    true,
		100:                   true,
		1000:                  true,
		1_000_000:             true,
		1_000_000_000:         true,
		1_000_000_000_000:     true,
		10_000_000_000_000:    true,
		100_000_000_000_000:   true,
		1_000_000_000_000_000: true,
	}

	start := uint64(1)
	fmt.Printf("%8s %5s %15s %8s %8s %6s %7s\n", "digits", "tail", "cycle", "exclude", "maximal", "last", "even")
	for mask, digits := uint64(10), 1; mask <= 1_000_000_000_000_000; mask, digits = 10*mask, digits+1 {
		fast := start
		slow := start
		i := 0
		for {
			fast = (fast * 4) % mask
			slow = (slow * 2) % mask
			i++
			if fast == slow {
				break
			}
		}

		slow = start
		mu := 0
		for {
			fast = (fast * 2) % mask
			slow = (slow * 2) % mask
			mu++
			if fast == slow {
				break
			}
		}
		// fmt.Printf("found cycle entry %d %d,%d\n", mu, fast, slow)

		n := 0
		for {
			fast = (fast * 2) % mask
			n++
			if fast == slow {
				break
			}
		}

		// verify that all of the tail elements never appear in the cycle
		// also that the tail is as long as possible
		tail := make([]uint64, mu+1)
		for i := 0; i < len(tail); i++ {
			tail[i] = (1 << i) % mask
		}

		exclusion := true  // are all the tail elements excluded from the cycle?
		inclusion := false // is the first element after the tail included?
		allEven := 0

		for i := 0; i < n; i++ {
			tmp := fast * 2
			fast = tmp % mask
			if evenDigits(fast) && tmp == fast {
				// all even digit and no carry
				allEven++
			}
			//fmt.Printf("%5d %t vs ", fast, evenDigits(fast))
			for j := 0; j < len(tail)-1; j++ {
				//fmt.Printf("%5d %t ", tail[j], fast == tail[j])
				if fast == tail[j] {
					exclusion = false
					break
				}
			}
			//fmt.Printf("\n")
			if fast == tail[mu] {
				inclusion = true
			}
		}

		_, export := exports[mask]
		if export {
			indexes := []int{}
			cycle := []uint64{}
			for i := 0; i < n; i++ {
				tmp := fast * 2
				fast = tmp % mask
				if evenDigits(fast) && tmp == fast {
					indexes = append(indexes, i+mu+1)
					cycle = append(cycle, fast)
				}
			}
			slices.Sort(cycle)
			output := struct {
				Mask      uint64
				Order     int
				Length    int
				Leadin    int
				EvenItems int
				Gain      float64
				Cycle     []uint64
				Index     []int
			}{
				Mask:      mask,
				Order:     digits,
				Length:    n,
				Leadin:    mu,
				EvenItems: len(cycle),
				Gain:      float64(n) / float64(len(cycle)),
				Cycle:     cycle,
				Index:     indexes,
			}

			f, err := os.OpenFile(fmt.Sprintf("cycle-%03d.json", digits), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
			if err != nil {
				log.Fatal(err)
			}
			defer func(f *os.File) {
				_ = f.Close()
			}(f)
			txt, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				log.Fatal(err)
			}
			_, err = f.Write(txt)
			if err != nil {
				log.Fatal(err)
			}
			_ = f.Close()
		}
		//fmt.Printf("entered cycle of length %d after %d steps\n", n, mu)
		_, _ = p.Printf("%8d %5d %15d %8t %8t %6d %7d %10.2f\n", digits, mu, n, inclusion, exclusion, tail[mu-1], allEven, float64(n)/float64(allEven))
	}
}

func evenDigits(x uint64) bool {
	even := true
	for z := x; z > 0; {
		d := z % 10
		z = z / 10
		if d%2 == 1 {
			even = false
			break
		}
	}
	return even
}
