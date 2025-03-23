package main

import (
	"EvenDigits/common"
	"flag"
	"github.com/shopspring/decimal"
	"log"
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

func main() {
	verbose := flag.Bool("verbose", false, "verbose output")
	limitString := flag.String(
		"limit",
		"10M",
		"Maximum value of N to search. Can use M, G, T, P and E as power of ten",
	)
	digits := flag.Int64("digits", 50, "Number of digits to retain in search")
	flag.Parse()
	limit := common.DecodeLimit(limitString, verbose)

	zero := decimal.NewFromInt(0)
	two := decimal.NewFromInt(2)
	ten := decimal.NewFromInt(10)
	mask := ten.Pow(decimal.NewFromInt(*digits))

	steps := []uint64{3, 3, 5, 8}
	bumps := make([]decimal.Decimal, len(steps))
	for i, step := range steps {
		bumps[i] = decimal.NewFromInt(1 << step)
	}

	solutions := []uint64{1, 2}

	t0 := time.Now()
	z := decimal.NewFromInt(1)
	for n := uint64(0); n < limit; {
		for i, dn := range steps {
			n += dn
			z = z.Mul(bumps[i]).Mod(mask)
			allEven := -1
			for i, zdig := i, z; zdig.GreaterThan(zero); i++ {
				var digit decimal.Decimal
				zdig, digit = zdig.QuoRem(ten, 0)
				if digit.IntPart()%2 != 0 {
					allEven = i
					break
				}
			}
			if allEven == -1 {
				solutions = append(solutions, n)
			}
		}
		n++
		z = z.Mul(two)

		if *verbose && n%10_000_000 == 0 {
			t := time.Since(t0).Seconds()
			rate := float64(n) / t
			log.Printf(
				"%6dM candidates searched, %d solutions found, %.3fs remaining\n",
				n/1_000_000,
				len(solutions),
				float64(limit-n)/rate,
			)
		}
	}
	log.Printf("solutions = %v\n", solutions)
}
