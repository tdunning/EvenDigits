package mp

import (
	"math"
	"slices"
)

// UInt256 is a 256-bit integer. These structures are entirely static and thus are
// subject to copy semantics. Importantly, they can be allocated on the stack to
// avoid GC pressure.
type UInt256 struct {
	Content [8]uint64
}

// UInt512 is a 512-bit integer. Only limited operations are supported since these
// are only used as temporary values in the implementation of ModMul for UInt256.
type UInt512 struct {
	content [16]uint64
}

// Cmp returns -1, 0 or 1 if a < b, a == b or a > b, respectively.
func (a UInt256) Cmp(b UInt256) int {
	for i := len(a.Content) - 1; i >= 0; i-- {
		if a.Content[i] > b.Content[i] {
			return 1
		}
		if a.Content[i] < b.Content[i] {
			return -1
		}
	}
	return 0
}

// Cmp256 returns -1, 0 or 1 if a < b, a == b or a > b, respectively but `a`
// is UInt512 and `b` is UInt256.
func (a UInt512) Cmp256(b UInt256) int {
	for i := len(a.content) - 1; i >= len(b.Content); i-- {
		if a.content[i] > 0 {
			return 1
		}
	}

	for i := len(b.Content) - 1; i >= 0; i-- {
		if a.content[i] > b.Content[i] {
			return 1
		}
		if a.content[i] < b.Content[i] {
			return -1
		}
	}
	return 0
}

// MulSmall destructively multiplies a large value by a small one. The
// destination value must be normalized and will be normalized again upon return.
// The multiplier should be limited to `[0...math.MaxUint32]`
func (a *UInt256) MulSmall(b uint64) {
	if b > math.MaxUint32 {
		panic("b > math.MaxUint16")
	}
	carry := uint64(0)
	for i := 0; i < len(a.Content); i++ {
		tmp := a.Content[i]*b + carry
		a.Content[i] = tmp & math.MaxUint32
		carry = tmp >> 32
	}
}

// AddSmall adds a 64bit quantity to a larger value which is destructively
// modified. The destination does not have to be normalized before calling
// this, but will be normalized afterwards.
func (a *UInt256) AddSmall(b uint64) {
	if b > math.MaxUint32 {
		panic("b > math.MaxUint16")
	}
	for i := 0; b != 0 && i < len(a.Content); i++ {
		u := a.Content[i] + b
		a.Content[i] = u & math.MaxUint32
		b = u >> 32
	}
}

// DivModSmall divides a large value by a small one and returns
// the remainder. The divisor should be less than `math.MaxUint32`
// and the destination should be normalized on entry.
func (a *UInt256) DivModSmall(b uint64) uint64 {
	rem := uint64(0)
	for i := len(a.Content) - 1; i >= 0; i-- {
		rem = rem << 32
		// if b==1, rem=0 and so this will work
		// if b>2, then this will succeed because the sum will fit
		// the result may not be normalized, however
		u := (rem + a.Content[i]) / b
		// have to do the % in two phases to avoid 65-bit overflow
		// since b <= math.MaxUint32, the remainders can be added
		// safely
		rem = (rem + a.Content[i]) % b
		a.Content[i] = u
	}

	// renormalize what we have
	carry := uint64(0)
	// loop invariant carry <= 2^32
	for i := 0; i < len(a.Content); i++ {
		z := a.Content[i]
		// (z&math.MaxUint32) < 2^32, carry <= 2^32, thus u < 2^33
		u := (z & math.MaxUint32) + carry
		a.Content[i] = u & math.MaxUint32
		// (z>>32) < 2^32, (u>>32) <= 1, thus carry <= 2^32
		carry = (z >> 32) + (u >> 32)
		if carry == 0 {
			break
		}
	}

	return rem
}

func (a *UInt256) Mod(b UInt256) {
	// j is last non-zero element of b
	j := len(b.Content) - 1
	for ; j >= 0; j-- {
		if b.Content[j] != 0 {
			break
		}
	}

	// the value `i` tracks the last non-zero element of a
	for i := len(a.Content) - 1; i >= j; {
		if a.Content[i] == 0 {
			i--
			continue
		}
		if i == j && a.Cmp(b) <= 0 {
			// our work here is done
			break
		}

		var (
			m      uint64
			ax, bx uint64
			offset int
		)

		if j > 0 {
			if a.Content[i] < b.Content[j] {
				ax = (a.Content[i] << 32) + a.Content[i-1]
				bx = b.Content[j]
				offset = i - j - 1
			} else {
				ax = (a.Content[i] << 32) + a.Content[i-1]
				bx = (b.Content[j] << 32) + b.Content[j-1]
				offset = i - j
			}
		} else {
			if i > 0 {
				ax = (a.Content[i] << 32) + a.Content[i-1]
				offset = i - j - 1
			} else {
				ax = a.Content[i]
				offset = i - j
			}
			bx = b.Content[j]
		}

		m = ax / (bx + 1)
		if m == 0 {
			// this happens if the difference between a and b is only in lower bits
			// so that ax == bx
			m = 1
		}

		// tmp = m * M^offset * b where M = 2^32
		tmp := UInt256{}
		carry := uint64(0)
		for i := offset; i < len(tmp.Content); i++ {
			u := b.Content[i-offset]*m + carry
			tmp.Content[i] = u & math.MaxUint32
			carry = u >> 32
		}
		if carry != 0 {
			panic("overflow on m * b")
		}

		for k := 0; k <= i; k++ {
			u := a.Content[k] - tmp.Content[k] + carry
			a.Content[k] = u & math.MaxUint32
			carry = uint64(int64(u) >> 32)
		}
		// assert carry == 0
	}
}

func (a *UInt512) Mod256(b UInt256) {
	// j is last non-zero element of b
	j := len(b.Content) - 1
	for ; j >= 0; j-- {
		if b.Content[j] != 0 {
			break
		}
	}

	// `i` tracks the last non-zero element of a
	for i := len(a.content) - 1; i >= j; {
		if a.content[i] == 0 {
			i--
			continue
		}
		if i == j && a.Cmp256(b) <= 0 {
			// our work here is done
			break
		}

		var (
			m      uint64
			ax, bx uint64
			offset int
		)

		if j > 0 {
			if a.content[i] < b.Content[j] {
				ax = (a.content[i] << 32) + a.content[i-1]
				bx = b.Content[j]
				offset = i - j - 1
			} else {
				ax = (a.content[i] << 32) + a.content[i-1]
				bx = (b.Content[j] << 32) + b.Content[j-1]
				offset = i - j
			}
		} else {
			if i > 0 {
				ax = (a.content[i] << 32) + a.content[i-1]
				offset = i - j - 1
			} else {
				ax = a.content[i]
				offset = i - j
			}
			bx = b.Content[j]
		}

		m = ax / (bx + 1)
		if m == 0 {
			// this happens if the difference between a and b is only in lower bits
			// so that ax == bx
			m = 1
		}

		// tmp = m * M^offset * b where M = 2^32
		tmp := UInt512{}
		carry := uint64(0)
		for i := offset; i < len(tmp.content); i++ {
			u := carry
			if i-offset < len(b.Content) {
				u += b.Content[i-offset] * m
			}

			tmp.content[i] = u & math.MaxUint32
			carry = u >> 32
			if i-offset >= len(b.Content) && carry == 0 {
				break
			}
		}
		if carry != 0 {
			panic("overflow on m * b")
		}

		for k := 0; k <= i; k++ {
			u := a.content[k] - tmp.content[k] + carry
			a.content[k] = u & math.MaxUint32
			carry = uint64(int64(u) >> 32)
		}
		// assert carry == 0
	}
}

func (a UInt512) Cmp512(b UInt512) int {
	for i := len(a.content) - 1; i >= 0; i-- {
		if a.content[i] > b.content[i] {
			return 1
		}
		if a.content[i] < b.content[i] {
			return -1
		}
	}
	return 0
}

func (a UInt256) Mul(b UInt256) UInt512 {
	r := UInt512{}
	for i, ax := range a.Content {
		for j, bx := range b.Content {
			// this will fit (just) because
			// math.MaxUint32 * math.MaxUint32 + math.MaxUint32 = math.MaxUint64
			r.content[i+j] += ax * bx
		}
		// loop invariant: c0 <= math.MaxUint32 + 2
		c0 := uint64(0)
		for k, rx := range r.content {
			u0 := rx >> 32
			// max value here is 2 * math.MaxUint32 + 2 = 2^34
			u1 := (rx & math.MaxUint32) + c0
			r.content[k] = u1 & math.MaxUint32
			c0 = u0 + (u1 >> 32) // max value is math.MaxUint32 + 2^34 / 2^32
		}
	}
	return r
}

func (a *UInt256) MulMod(b, mask UInt256) {
	z := a.Mul(b)
	z.Mod256(mask)
	for i := 0; i < len(a.Content); i++ {
		a.Content[i] = z.content[i]
	}
}

func (a *UInt256) Pow256(n, mask UInt256) {
	m := *a
	r := UInt256{[8]uint64{1}}
	i := 0
	bit := 0
	for i < len(n.Content) {
		if n.Content[i] == 0 {
			i++
			bit = 0
			continue
		}
		if n.Content[i]&(1<<bit) != 0 {
			r.MulMod(m, mask)
		}
		m.MulMod(m, mask)
		bit++
		if bit == 32 {
			i++
			bit = 0
		}
	}
	*a = r
}

// PowerTable creates a table of $a^{2^n}$ that helps accelerate the
// computation of powers of $a$ with `PowByTable`
func PowerTable(a, mask UInt256) []UInt256 {
	r := make([]UInt256, 256)
	z := a
	for i := 0; i < 256; i++ {
		r[i] = z
		z.MulMod(z, mask)
	}
	return r
}

// PowByTable computes the $n$-th power of `table[0]`.
// The `table` should the output of a call to `PowerTable`
func PowByTable(table []UInt256, n, mask UInt256) UInt256 {
	r := UInt256{[8]uint64{1}}
	i := 0
	bit := 0
	for i < len(n.Content) {
		if n.Content[i] == 0 {
			i++
			bit = 0
			continue
		}
		if n.Content[i]&(1<<bit) != 0 {
			r.MulMod(table[32*i+bit], mask)
		}
		bit++
		if bit == 32 {
			i++
			bit = 0
		}
	}
	return r
}

func (a UInt256) String() string {
	r := make([]byte, 0)
	zero := UInt256{}
	for a.Cmp(zero) > 0 {
		d := a.DivModSmall(10)
		r = append(r, byte(d)+'0')
	}
	if len(r) == 0 {
		return "0"
	}
	slices.Reverse(r)
	return string(r)
}
