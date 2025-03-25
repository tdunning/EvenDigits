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
		assert.Equal(t, uint64(0), a.content[i])
	}
	assert.Equal(t, x, a.content[0])

	u0 := uint64(rand.Uint32()) | (1 << 31)
	u1 := uint64(math.MaxUint32)
	u2 := uint64(rand.Uint32()) | (1 << 31)
	a.content[0] = u0
	a.content[1] = u1
	a.content[2] = u2

	x |= 1 << 31

	a.AddSmall(x)
	assert.Equal(t, math.MaxUint32&(u0+x), a.content[0])
	carry := (u0 + x) >> 32
	assert.Equal(t, math.MaxUint32&(u1+carry), a.content[1])
	carry = (u1 + carry) >> 32
	assert.Equal(t, math.MaxUint32&(u2+carry), a.content[2])
	carry = (u2 + carry) >> 32
	assert.Equal(t, math.MaxUint32&carry, a.content[3])
}

func TestInt320_MulSmall(t *testing.T) {
	x := uint64(rand.Uint32())
	u0 := uint64(rand.Uint32())
	u1 := uint64(rand.Uint32())
	u2 := uint64(rand.Uint32())
	a := UInt256{[8]uint64{u0, u1, u2}}

	a.MulSmall(x)
	t0 := x * u0
	assert.Equal(t, t0&math.MaxUint32, a.content[0])
	t1 := (t0 >> 32) + x*u1
	assert.Equal(t, t1&math.MaxUint32, a.content[1])
	t2 := (t1 >> 32) + x*u2
	assert.Equal(t, t2&math.MaxUint32, a.content[2])
}

func TestInt320_DivRemSmall2(t *testing.T) {
	a := UInt256{}
	a.AddSmall(uint64(5003))
	n := a.DivModSmall(1000)
	assert.Equal(t, uint64(3), n)
	assert.Equal(t, uint64(5), a.content[0])

	for i := 0; i < 8; i++ {
		a.content[i] = math.MaxUint32 - 1
	}
	ax := a
	b0 := uint64(math.MaxUint32)
	r0 := a.DivModSmall(b0)
	a.MulSmall(b0)
	a.AddSmall(r0)
	assert.True(t, ax.Cmp(a) == 0)

	for i := 0; i < 1; i++ {
		for i := 0; i < 8; i++ {
			a.content[i] = uint64(rand.Uint32())
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
	assert.Equal(t, uint64(429496729), a.content[0])
	assert.Equal(t, uint64(0), a.content[1])

	a = UInt256{[8]uint64{1, 1, 1, 1, 0, 0, 0, 0}}
	r = a.DivModSmall(10000)
	assert.Equal(t, uint64(9249), r)
	assert.Equal(t, uint64(3828104350), a.content[0])
	assert.Equal(t, uint64(3134037635), a.content[1])
	assert.Equal(t, uint64(429496), a.content[2])
	assert.Equal(t, uint64(0), a.content[3])

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
