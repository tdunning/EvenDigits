package mp

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math"
	"math/rand/v2"
	"testing"
)

func TestInt320_AddSmall(t *testing.T) {
	x := uint64(rand.Uint32())
	a := UInt256{}
	a.AddSmall(x)
	for i := 1; i < 8; i++ {
		assert.Equal(t, uint64(0), a.Content[i])
	}
	assert.Equal(t, x, a.Content[0])

	u0 := uint64(rand.Uint32()) | (1 << 31)
	u1 := uint64(math.MaxUint32)
	u2 := uint64(rand.Uint32()) | (1 << 31)
	a.Content[0] = u0
	a.Content[1] = u1
	a.Content[2] = u2

	x |= 1 << 31

	a.AddSmall(x)
	assert.Equal(t, math.MaxUint32&(u0+x), a.Content[0])
	carry := (u0 + x) >> 32
	assert.Equal(t, math.MaxUint32&(u1+carry), a.Content[1])
	carry = (u1 + carry) >> 32
	assert.Equal(t, math.MaxUint32&(u2+carry), a.Content[2])
	carry = (u2 + carry) >> 32
	assert.Equal(t, math.MaxUint32&carry, a.Content[3])
}

func TestInt320_MulSmall(t *testing.T) {
	x := uint64(rand.Uint32())
	u0 := uint64(rand.Uint32())
	u1 := uint64(rand.Uint32())
	u2 := uint64(rand.Uint32())
	a := UInt256{[8]uint64{u0, u1, u2}}

	a.MulSmall(x)
	t0 := x * u0
	assert.Equal(t, t0&math.MaxUint32, a.Content[0])
	t1 := (t0 >> 32) + x*u1
	assert.Equal(t, t1&math.MaxUint32, a.Content[1])
	t2 := (t1 >> 32) + x*u2
	assert.Equal(t, t2&math.MaxUint32, a.Content[2])
}

func TestInt320_DivRemSmall2(t *testing.T) {
	a := UInt256{}
	a.AddSmall(uint64(5003))
	n := a.DivModSmall(1000)
	assert.Equal(t, uint64(3), n)
	assert.Equal(t, uint64(5), a.Content[0])

	for i := 0; i < 8; i++ {
		a.Content[i] = math.MaxUint32 - 1
	}
	ax := a
	b0 := uint64(math.MaxUint32)
	r0 := a.DivModSmall(b0)
	a.MulSmall(b0)
	a.AddSmall(r0)
	assert.True(t, ax.Cmp(a) == 0)

	for i := 0; i < 1; i++ {
		for i := 0; i < 8; i++ {
			a.Content[i] = uint64(rand.Uint32())
		}
		b := uint64(23)

		ax := a
		r := a.DivModSmall(b)
		assert.True(t, r < b)
		assert.True(t, ax.Cmp(a) != 0)
		a.MulSmall(b)
		a.AddSmall(r)
		assert.True(t, ax.Cmp(a) == 0)
	}

}

func Test_DivRemSmall(t *testing.T) {
	// 2^32 % 5 == 6
	a := UInt256{[8]uint64{0, 1, 0, 0, 0, 0, 0, 0}}
	r := a.DivModSmall(10)
	assert.Equal(t, uint64(6), r)
	assert.Equal(t, uint64(429496729), a.Content[0])
	assert.Equal(t, uint64(0), a.Content[1])

	a = UInt256{[8]uint64{1, 1, 1, 1, 0, 0, 0, 0}}
	r = a.DivModSmall(10000)
	assert.Equal(t, uint64(9249), r)
	assert.Equal(t, uint64(3828104350), a.Content[0])
	assert.Equal(t, uint64(3134037635), a.Content[1])
	assert.Equal(t, uint64(429496), a.Content[2])
	assert.Equal(t, uint64(0), a.Content[3])

	// floor(pi * 10^70) takes up about 234 bits
	piDigits := "31415926535897932384626433832795028841971693993751058209749445923078164"
	pi := UInt256{
		[8]uint64{
			3441197076, 2304935270, 1582441405, 2787492932, 696018738, 153849261, 1208944667, 1165,
		},
	}

	for i := len(piDigits) - 1; i >= 0; i-- {
		r := pi.DivModSmall(uint64(10))
		assert.True(
			t,
			uint64(piDigits[i]) == '0'+r,
			fmt.Sprintf("divide remainder #%d %c vs %c\n", i, uint64(piDigits[i]), '0'+r),
		)
	}
}

var (
	// pi * 10^70
	pi70 = UInt256{[8]uint64{
		3441197076, 2304935270, 1582441405, 2787492932, 696018738, 153849261, 1208944667, 1165,
	}}
	pi100 = UInt512{[16]uint64{
		1454363223, 2483199816, 3756646060, 3135356928, 1069813062, 2158551865, 1748634935,
		27139563, 1418441833, 703265957, 1841414616, 3411550463, 79731, 0, 0, 0,
	}}
	// e * 10^70
	e70 = UInt256{[8]uint64{
		2838434750, 2938999430, 284363889, 1134976221, 2540683272, 1877661008, 1145758728, 1008,
	}}
)

func Test_Mod0(t *testing.T) {
	a := pi70
	b := e70
	a.Mod(b)
	aModB := UInt256{[8]uint64{
		602762326, 3660903136, 1298077515, 1652516711, 2450302762, 2571155548, 63185938, 157,
	}}
	assert.Equal(t, 0, a.Cmp(aModB))
}

func Test_Mod1(t *testing.T) {
	a := pi70
	// pi * 10^55 - 1000
	b := UInt256{
		[8]uint64{2079167289, 1683576302, 4089510878, 392375461, 960694935, 21495649, 0, 0},
	}
	a.Mod(b)
	// a % b = 1000749445923078164
	aModB := UInt256{
		[8]uint64{2708078612, 233005137, 0, 0, 0, 0, 0, 0},
	}
	assert.Equal(t, 0, a.Cmp(aModB))
}

func Test_Mod2(t *testing.T) {
	a := pi70
	// pi70 / 2^64 - 1000
	b := UInt256{[8]uint64{1582440405, 2787492932, 696018738, 153849261, 1208944667, 1165, 0, 0}}
	a.Mod(b)
	aModB := UInt256{[8]uint64{
		3441197076, 2304935270, 1000, 0, 0, 0, 0, 0,
	}}
	assert.Equal(t, 0, a.Cmp(aModB))
}

func Test_Mod3(t *testing.T) {
	a := pi70
	b := UInt256{[8]uint64{
		10000, 0, 0, 0, 0, 0, 0, 0,
	}}
	a.Mod(b)
	aModB := UInt256{[8]uint64{
		8164, 0, 0, 0, 0, 0, 0, 0,
	}}
	assert.Equal(t, 0, a.Cmp(aModB))
}

func Test_Mod4(t *testing.T) {
	a := UInt256{[8]uint64{
		489789245, 0, 0, 0, 0, 0, 0, 0,
	}}
	b := UInt256{[8]uint64{
		10000, 0, 0, 0, 0, 0, 0, 0,
	}}
	a.Mod(b)
	aModB := UInt256{[8]uint64{
		9245, 0, 0, 0, 0, 0, 0, 0,
	}}
	assert.Equal(t, 0, a.Cmp(aModB))
}

func Test_Mod256_0(t *testing.T) {
	a := pi100
	b := e70
	a.Mod256(b)
	aModB := UInt256{[8]uint64{
		1812683721, 2559088218, 2015340409, 921934341, 1001620618, 3020082437, 3700726681, 586,
	}}
	assert.Equal(t, 0, a.Cmp256(aModB))
}

func Test_Mod256_1(t *testing.T) {
	a := pi70
	// pi * 10^55 - 1000
	b := UInt256{
		[8]uint64{2079167289, 1683576302, 4089510878, 392375461, 960694935, 21495649, 0, 0},
	}
	a.Mod(b)
	// a % b = 1000749445923078164
	aModB := UInt256{
		[8]uint64{2708078612, 233005137, 0, 0, 0, 0, 0, 0},
	}
	assert.Equal(t, 0, a.Cmp(aModB))
}

func Test_Mod256_2(t *testing.T) {
	a := pi70
	// pi70 / 2^64 - 1000
	b := UInt256{[8]uint64{1582440405, 2787492932, 696018738, 153849261, 1208944667, 1165, 0, 0}}
	a.Mod(b)
	aModB := UInt256{[8]uint64{
		3441197076, 2304935270, 1000, 0, 0, 0, 0, 0,
	}}
	assert.Equal(t, 0, a.Cmp(aModB))
}

func Test_Mod256_3(t *testing.T) {
	a := UInt512{}
	for i := 0; i < len(pi70.Content); i++ {
		a.content[i] = pi70.Content[i]
	}
	b := UInt256{[8]uint64{
		10000, 0, 0, 0, 0, 0, 0, 0,
	}}
	a.Mod256(b)
	aModB := UInt256{[8]uint64{
		8164, 0, 0, 0, 0, 0, 0, 0,
	}}
	assert.Equal(t, 0, a.Cmp256(aModB))
}

func Test_Mod256_4(t *testing.T) {
	a := UInt512{[16]uint64{
		489789245, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	}}
	b := UInt256{[8]uint64{
		10000, 0, 0, 0, 0, 0, 0, 0,
	}}
	a.Mod256(b)
	aModB := UInt256{[8]uint64{
		9245, 0, 0, 0, 0, 0, 0, 0,
	}}
	assert.Equal(t, 0, a.Cmp256(aModB))
}

func Test_Mul(t *testing.T) {
	a := pi70
	b := e70
	c := a.Mul(b)
	ref := UInt512{[16]uint64{
		273976024, 1875349190, 1318867507, 1103398094, 4019063274, 1781510510, 2878157347, 2301794640,
		1643369626, 2082417994, 819394330, 3670182513, 3030107420, 2537076616, 1174914, 0,
	}}
	assert.Equal(t, 0, c.Cmp512(ref))
}

func Test_PowMod(t *testing.T) {
	mask := UInt256{[8]uint64{1}}
	for i := 0; i < 55; i++ {
		mask.MulSmall(10)
	}
	a := UInt256{[8]uint64{2}}
	a.Pow256(UInt256{[8]uint64{2}}, mask)
	pow := UInt256{[8]uint64{1 << 2}}
	assert.Equal(t, 0, a.Cmp(pow))

	a = UInt256{[8]uint64{2}}
	a.Pow256(UInt256{[8]uint64{10000}}, mask)
	pow = UInt256{[8]uint64{0, 2449473536, 1386834847, 401415762, 3736286779, 2337887, 0, 0}}
	assert.Equal(t, 0, a.Cmp(pow))
}

func Test_PowByTable(t *testing.T) {
	setbit := func(z *UInt256, bit int) {
		i := bit / 32
		bit = bit % 32
		z.Content[i] = z.Content[i] | (1 << bit)
	}

	mask := UInt256{[8]uint64{1}}
	for i := 0; i < 55; i++ {
		mask.MulSmall(10)
	}
	table := PowerTable(UInt256{[8]uint64{2}}, mask)
	for i := 0; i < 250; i++ {
		z1 := table[i]
		n := UInt256{[8]uint64{}}
		setbit(&n, i)
		z2 := PowByTable(table, n, mask)
		assert.Equal(t, z1, z2)
	}
	for i := 3; i < 11; {
		//j := 2
		//k := 5
		j := MinNRand(8, 200)
		k := MinNRand(8, 200)
		if i == j || i == k {
			continue
		}
		z1 := table[i]
		z1.MulMod(table[j], mask)
		z1.MulMod(table[k], mask)

		z2 := table[0]
		n := UInt256{[8]uint64{}}
		setbit(&n, i)
		setbit(&n, j)
		setbit(&n, k)
		z2.Pow256(n, mask)

		z3 := PowByTable(table, n, mask)
		assert.Equal(t, z1, z2)
		assert.Equal(t, z1, z3)
		i++
	}
}

func MinNRand(n int, scale int) int {
	r := scale
	for i := 0; i < n; i++ {
		z := rand.IntN(scale)
		if z < r {
			r = z
		}
	}
	return r
}

func Test_String(t *testing.T) {
	assert.Equal(t, "0", UInt256{[8]uint64{}}.String())
	assert.Equal(t, "1", UInt256{[8]uint64{1}}.String())
	assert.Equal(t, "1000", UInt256{[8]uint64{1000}}.String())
	mask := UInt256{[8]uint64{1}}
	for i := 0; i < 50; i++ {
		mask.MulSmall(10)
	}
	z := UInt256{[8]uint64{2}}
	z.Pow256(UInt256{[8]uint64{2000}}, mask)
	assert.Equal(t, "25175435528800822842770817965453762184851149029376", z.String())
}
