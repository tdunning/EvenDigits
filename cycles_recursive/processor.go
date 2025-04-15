package main

import (
	"container/list"
	"fmt"
)

type Digit struct {
	exponent int
	value    uint8
	odd      bool
}

type Cycle struct {
	index  int
	length int
	digits *list.List
}

var cycle *Cycle

// initializes
func init() {
	cycle = &Cycle{
		index:  1,
		length: 4,
		digits: list.New(),
	}
	cycle.digits.PushBack(Digit{exponent: 1, value: 2, odd: false})
	cycle.digits.PushBack(Digit{exponent: 2, value: 4, odd: false})
	cycle.digits.PushBack(Digit{exponent: 3, value: 8, odd: false})
	cycle.digits.PushBack(Digit{exponent: 4, value: 6, odd: false})
}

func main() {
	printCycle(cycle)
	// 11 will run up to ccycle 12 (max on my machine with 64GB Ram)
	for range make([]struct{}, 11) {
		cycle = createNextCycle(cycle)
		printCycle(cycle)
	}
}

// creates a new cycle based on the previous cycle
// instead of using power of 2 we calculate the highest significant digits
// based on the highest signifcant digit of the previous cycle
func createNextCycle(prevCycle *Cycle) *Cycle {
	newCycle := &Cycle{
		index:  prevCycle.index + 1,
		length: prevCycle.length * 5,
		digits: list.New(),
	}

	value := uint8(0)
	// we know that the new cycle is 5 times longer than the previous one
	// and the new cycle starts with +1 exponent offset
	for i := 0; i < 5; i++ {
		for e := prevCycle.digits.Front(); e != nil; e = e.Next() {
			prevDigit := e.Value.(Digit)
			value = (prevDigit.value*2)/10 + (value*2)%10

			newCycle.digits.PushBack(Digit{
				exponent: prevCycle.length*i + prevDigit.exponent + 1,
				value:    value,
				odd:      prevDigit.odd || (value%2 == 1),
			})

		}
	}

	return newCycle
}

func printCycle(cycle *Cycle) {
	fmt.Printf("Cycle: {\n  index: %d,\n  length: %d,\n", cycle.index, cycle.length)
	printValues(cycle.digits)
	fmt.Print("}\n\n")
}

func printValues(digits *list.List) {
	oddCounter := 0
	// fmt.Printf("  values: {\n")
	for e := digits.Front(); e != nil; e = e.Next() {
		digit := e.Value.(Digit)
		// fmt.Printf("    {value: %d, odd: %5t, exponent: %d},\n", digit.value, digit.odd, digit.exponent)
		if digit.odd {
			oddCounter++
		}
	}
	// fmt.Printf("  }\n")
	fmt.Printf("  odd: %.6f %%\n", float64(oddCounter)/float64(digits.Len())*100)
}
