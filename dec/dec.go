// Package dec implements multi-precision decimal arithmetic.
// It supports the numeric type Dec for signed decimals.
// It is based on and complements the multi-precision integer implementation
// (Int) in the Go library (math/big).
//
// Methods are typically of the form:
//
//	func (z *Dec) Op(x, y *Dec) *Dec
//
// and implement operations z = x Op y with the result as receiver; if it
// is one of the operands it may be overwritten (and its memory reused).
// To enable chaining of operations, the result is also returned. Methods
// returning a result other than *Dec take one of the operands as the receiver.
//
// Quotient (division) operation uses Scalers and Rounders to specify the
// desired behavior. See Quo, Scaler, and Rounder for details. 
//
package dec

// This file implements signed multi-precision decimals.

import (
	"fmt"
	"io"
	"math/big"
	"strings"
)

// A Dec represents a signed multi-precision decimal.
// It is stored as a combination of a multi-precision big.Int unscaled value
// and a fixed-precision scale of type Scale.
//
// The mathematical value of a Dec equals:
//
//  unscaled * 10**(-scale)
//
// Note that different Dec representations may have equal mathematical values.
//
//  unscaled  scale  String()
//  -------------------------
//         0      0    "0"
//         0      2    "0.00"
//         0     -2    "0"
//         1      0    "1"
//       100      2    "1.00"
//        10      0   "10"
//         1     -1   "10"
//
// The zero value for a Dec represents the value 0 with scale 0.
//
type Dec struct {
	unscaled big.Int
	scale    Scale
}

// Scale represents the type used for the scale of a Dec.
type Scale int32

const scaleSize = 4 // bytes in a Scale value

// Scaler represents a function that returns the scale for the result of an
// operation on x and y.
type Scaler func(x *Dec, y *Dec) (scale Scale)

// Rounder represents a method for rounding the (possibly infinite decimal)
// result of a division to a finite Dec. It is used by Quo().
//
type Rounder interface {

	// When UseRemainder() returns true, the Round() method is passed the
	// remainder of the division, expressed as the numerator and denominator of
	// a rational.
	UseRemainder() bool

	// Round sets the rounded value of a quotient to z, and returns z.
	// quo is rounded down (truncated towards zero) to the scale obtained from
	// the Scaler in Quo().
	//
	// When the remainder is not used, remNum and remDen are nil.
	// When used, the remainder is normalized between -1 and 1; that is:
	// 
	//  -|remDen| < remNum < |remDen|
	//
	// remDen has the same sign as y, and remNum is zero or has the same sign
	// as x.
	Round(z, quo *Dec, remNum, remDen *big.Int) *Dec
}

var bigInt = [...]*big.Int{
	big.NewInt(0), big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4),
	big.NewInt(5), big.NewInt(6), big.NewInt(7), big.NewInt(8), big.NewInt(9),
	big.NewInt(10),
}

var exp10cache [64]big.Int = func() [64]big.Int {
	e10, e10i := [64]big.Int{}, bigInt[1]
	for i, _ := range e10 {
		e10[i].Set(e10i)
		e10i = new(big.Int).Mul(e10i, bigInt[10])
	}
	return e10
}()

// NewDec allocates and returns a new Dec set to the given unscaled value and
// scale.
func NewDec(unscaled *big.Int, scale Scale) *Dec {
	return new(Dec).SetUnscaled(unscaled).SetScale(scale)
}

// NewDecInt64 allocates and returns a new Dec set to the given int64 value with
// scale 0.
func NewDecInt64(x int64) *Dec {
	return new(Dec).SetUnscaled(big.NewInt(x))
}

// Scale returns the scale of x.
func (x *Dec) Scale() Scale {
	return x.scale
}

// Unscaled returns the unscaled value of x.
func (x *Dec) Unscaled() *big.Int {
	return &x.unscaled
}

// SetScale sets the scale of x, with the unscaled value unchanged.
// The mathematical value of the Dec changes as if it was multiplied by
// 10**(oldscale-scale). 
func (x *Dec) SetScale(scale Scale) *Dec {
	x.scale = scale
	return x
}

// SetScale sets the unscaled value of x, with the scale unchanged.
func (x *Dec) SetUnscaled(unscaled *big.Int) *Dec {
	x.unscaled.Set(unscaled)
	return x
}

// Set sets z to the value of x and returns z.
// It does nothing if z == x.
func (z *Dec) Set(x *Dec) *Dec {
	if z != x {
		z.SetUnscaled(x.Unscaled())
		z.SetScale(x.Scale())
	}
	return z
}

// Move sets z to the value of x, and sets x to zero, unless z == x.
// It is intended for fast assignment from temporary variables without copying
// the underlying array.
func (z *Dec) move(x *Dec) *Dec {
	if z != x {
		*z = *x
		*x = Dec{}
	}
	return z
}

// Sign returns:
//
//	-1 if x <  0
//	 0 if x == 0
//	+1 if x >  0
//
func (x *Dec) Sign() int {
	return x.Unscaled().Sign()
}

// Neg sets z to -x and returns z.
func (z *Dec) Neg(x *Dec) *Dec {
	z.SetScale(x.Scale())
	z.Unscaled().Neg(x.Unscaled())
	return z
}

// Cmp compares x and y and returns:
//
//   -1 if x <  y
//    0 if x == y
//   +1 if x >  y
//
func (x *Dec) Cmp(y *Dec) int {
	xx, yy := upscale(x, y)
	return xx.Unscaled().Cmp(yy.Unscaled())
}

// Abs sets z to |x| (the absolute value of x) and returns z.
func (z *Dec) Abs(x *Dec) *Dec {
	z.SetScale(x.Scale())
	z.Unscaled().Abs(x.Unscaled())
	return z
}

// Add sets z to the sum x+y and returns z.
// The scale of z is the greater of the scales of x and y.
func (z *Dec) Add(x, y *Dec) *Dec {
	xx, yy := upscale(x, y)
	z.SetScale(xx.Scale())
	z.Unscaled().Add(xx.Unscaled(), yy.Unscaled())
	return z
}

// Sub sets z to the difference x-y and returns z.
// The scale of z is the greater of the scales of x and y.
func (z *Dec) Sub(x, y *Dec) *Dec {
	xx, yy := upscale(x, y)
	z.SetScale(xx.Scale())
	z.Unscaled().Sub(xx.Unscaled(), yy.Unscaled())
	return z
}

// Mul sets z to the product x*y and returns z.
// The scale of z is the sum of the scales of x and y.
func (z *Dec) Mul(x, y *Dec) *Dec {
	z.SetScale(x.Scale() + y.Scale())
	z.Unscaled().Mul(x.Unscaled(), y.Unscaled())
	return z
}

// Quo sets z to the quotient x/y, with the scale obtained from the given
// Scaler, rounded using the given Rounder.
// If the result from the rounder is nil, Quo also returns nil, and the value
// of z is undefined.
//
// There is no corresponding Div method; the equivalent can be achieved through
// the choice of Rounder used.
//
// See Rounder for details on the various ways for rounding.
func (z *Dec) Quo(x, y *Dec, scaler Scaler, rounder Rounder) *Dec {
	s := scaler(x, y)
	var zzz *Dec
	if rounder.UseRemainder() {
		zz, rA, rB := new(Dec).quoRem(x, y, s, true, new(big.Int), new(big.Int))
		zzz = rounder.Round(new(Dec), zz, rA, rB)
	} else {
		zz, _, _ := new(Dec).quoRem(x, y, s, false, nil, nil)
		zzz = rounder.Round(new(Dec), zz, nil, nil)
	}
	if zzz == nil {
		return nil
	}
	return z.move(zzz)
}

// QuoExact(x, y) is a shorthand for Quo(x, y, ScaleQuoExact, RoundExact).
// If x/y can be expressed as a Dec without rounding, QuoExact sets z to the
// quotient x/y and returns z. Otherwise, it returns nil and the value of z is
// undefined.
func (z *Dec) QuoExact(x, y *Dec) *Dec {
	return z.Quo(x, y, ScaleQuoExact, RoundExact)
}

// quoRem sets z to the quotient x/y with the scale s, and if useRem is true,
// it sets remNum and remDen to the numerator and denominator of the remainder.
// It returns z, remNum and remDen.
//
// The remainder is normalized to the range -1 < r < 1 to simplify rounding;
// that is, the results satisfy the following equation:
//
//  x / y = z + (remNum/remDen) * 10**(-z.Scale())
//
// See Rounder for more details about rounding.
//
func (z *Dec) quoRem(x, y *Dec, s Scale, useRem bool,
	remNum, remDen *big.Int) (*Dec, *big.Int, *big.Int) {
	// difference (required adjustment) compared to "canonical" result scale
	shift := s - (x.Scale() - y.Scale())
	// pointers to adjusted unscaled dividend and divisor
	var ix, iy *big.Int
	switch {
	case shift > 0:
		// increased scale: decimal-shift dividend left
		ix = new(big.Int).Mul(x.Unscaled(), exp10(shift))
		iy = y.Unscaled()
	case shift < 0:
		// decreased scale: decimal-shift divisor left
		ix = x.Unscaled()
		iy = new(big.Int).Mul(y.Unscaled(), exp10(-shift))
	default:
		ix = x.Unscaled()
		iy = y.Unscaled()
	}
	// save a copy of iy in case it to be overwritten with the result
	iy2 := iy
	if iy == z.Unscaled() {
		iy2 = new(big.Int).Set(iy)
	}
	// set scale
	z.SetScale(s)
	// set unscaled
	if useRem {
		// Int division
		_, intr := z.Unscaled().QuoRem(ix, iy, new(big.Int))
		// set remainder
		remNum.Set(intr)
		remDen.Set(iy2)
	} else {
		z.Unscaled().Quo(ix, iy)
	}
	return z, remNum, remDen
}

// ScaleFixed returns a Scaler with the given fixed result.
func ScaleFixed(scale Scale) Scaler {
	return func(x, y *Dec) Scale {
		return scale
	}
}

// ScaleFixed0 is a Scaler that always returns 0. It is intended to be used 
// with Quo when the result is to be rounded to an integer. 
var ScaleFixed0 Scaler = ScaleFixed(0)

// ScaleFixed2 is a Scaler that always returns 2. It is intended to be used 
// with Quo when the result is to be rounded to an decimal with scale 2. 
var ScaleFixed2 Scaler = ScaleFixed(2)

// ScaleQuoExact is the Scaler used by QuoExact. It returns a scale that is
// greater than or equal to "x.Scale() - y.Scale()"; it is calculated so that
// the remainder will be zero whenever x/y is a finite decimal.
var ScaleQuoExact Scaler = scaleQuoExact

func scaleQuoExact(x, y *Dec) Scale {
	rem := new(big.Rat).SetFrac(x.Unscaled(), y.Unscaled())
	f2, f5 := factor2(rem.Denom()), factor(rem.Denom(), bigInt[5])
	var f10 Scale
	if f2 > f5 {
		f10 = Scale(f2)
	} else {
		f10 = Scale(f5)
	}
	return x.Scale() - y.Scale() + f10
}

func factor(n *big.Int, p *big.Int) int {
	// could be improved for large factors
	d, f := n, 0
	for {
		dd, dm := new(big.Int).DivMod(d, p, new(big.Int))
		if dm.Sign() == 0 {
			f++
			d = dd
		} else {
			break
		}
	}
	return f
}

func factor2(n *big.Int) int {
	// could be improved for large factors
	f := 0
	for ; n.Bit(f) == 0; f++ {
	}
	return f
}

type rounder struct {
	useRem bool
	round  func(z, quo *Dec, remNum, remDen *big.Int) *Dec
}

func (r rounder) UseRemainder() bool {
	return r.useRem
}

func (r rounder) Round(z, quo *Dec, remNum, remDen *big.Int) *Dec {
	return r.round(z, quo, remNum, remDen)
}

// RoundExact returns quo if rem is zero, or nil otherwise. It is intended to
// be used with ScaleQuoExact when it is guaranteed that the result can be
// obtained without rounding. QuoExact is a shorthand for such a quotient
// operation. 
// 
var RoundExact Rounder = roundExact

// RoundDown rounds towards 0; that is, returns the Dec with the greatest
// absolute value not exceeding that of the result represented by quo and rem.
//
// The following table shows examples of the results for
// Quo(x, y, ScaleFixed(scale), RoundDown).
//
//      x      y    scale   result
//  ------------------------------
//    -1.8    10        1     -0.1
//    -1.5    10        1     -0.1
//    -1.2    10        1     -0.1
//    -1.0    10        1     -0.1
//    -0.8    10        1     -0.0
//    -0.5    10        1     -0.0
//    -0.2    10        1     -0.0
//     0.0    10        1      0.0
//     0.2    10        1      0.0
//     0.5    10        1      0.0
//     0.8    10        1      0.0
//     1.0    10        1      0.1
//     1.2    10        1      0.1
//     1.5    10        1      0.1
//     1.8    10        1      0.1
//
var RoundDown Rounder = roundDown

// RoundUp rounds away from 0; that is, returns the Dec with the smallest
// absolute value not smaller than that of the result represented by quo and
// rem.
//
// The following table shows examples of the results for
// Quo(x, y, ScaleFixed(scale), RoundUp).
//
//      x      y    scale   result
//  ------------------------------
//    -1.8    10        1     -0.2
//    -1.5    10        1     -0.2
//    -1.2    10        1     -0.2
//    -1.0    10        1     -0.1
//    -0.8    10        1     -0.1
//    -0.5    10        1     -0.1
//    -0.2    10        1     -0.1
//     0.0    10        1      0.0
//     0.2    10        1      0.1
//     0.5    10        1      0.1
//     0.8    10        1      0.1
//     1.0    10        1      0.1
//     1.2    10        1      0.2
//     1.5    10        1      0.2
//     1.8    10        1      0.2
//
var RoundUp Rounder = roundUp

// RoundHalfDown rounds to the nearest Dec, and when the remainder is 1/2, it
// rounds to the Dec with the lower absolute value.
//
// The following table shows examples of the results for
// Quo(x, y, ScaleFixed(scale), RoundHalfDown).
//
//      x      y    scale   result
//  ------------------------------
//    -1.8    10        1     -0.2
//    -1.5    10        1     -0.1
//    -1.2    10        1     -0.1
//    -1.0    10        1     -0.1
//    -0.8    10        1     -0.1
//    -0.5    10        1     -0.0
//    -0.2    10        1     -0.0
//     0.0    10        1      0.0
//     0.2    10        1      0.0
//     0.5    10        1      0.0
//     0.8    10        1      0.1
//     1.0    10        1      0.1
//     1.2    10        1      0.1
//     1.5    10        1      0.1
//     1.8    10        1      0.2
//
var RoundHalfDown Rounder = roundHalfDown

// RoundHalfUp rounds to the nearest Dec, and when the remainder is 1/2, it
// rounds to the Dec with the greater absolute value.
//
// The following table shows examples of the results for
// Quo(x, y, ScaleFixed(scale), RoundHalfUp).
//
//      x      y    scale   result
//  ------------------------------
//    -1.8    10        1     -0.2
//    -1.5    10        1     -0.2
//    -1.2    10        1     -0.1
//    -1.0    10        1     -0.1
//    -0.8    10        1     -0.1
//    -0.5    10        1     -0.1
//    -0.2    10        1     -0.0
//     0.0    10        1      0.0
//     0.2    10        1      0.0
//     0.5    10        1      0.1
//     0.8    10        1      0.1
//     1.0    10        1      0.1
//     1.2    10        1      0.1
//     1.5    10        1      0.2
//     1.8    10        1      0.2
//
var RoundHalfUp Rounder = roundHalfUp

// RoundFloor rounds towards negative infinity; that is, returns the greatest
// Dec not exceeding the result represented by quo and rem.
//
// The following table shows examples of the results for
// Quo(x, y, ScaleFixed(scale), RoundFloor).
//
//      x      y    scale   result
//  ------------------------------
//    -1.8    10        1     -0.2
//    -1.5    10        1     -0.2
//    -1.2    10        1     -0.2
//    -1.0    10        1     -0.1
//    -0.8    10        1     -0.1
//    -0.5    10        1     -0.1
//    -0.2    10        1     -0.1
//     0.0    10        1      0.0
//     0.2    10        1      0.0
//     0.5    10        1      0.0
//     0.8    10        1      0.0
//     1.0    10        1      0.1
//     1.2    10        1      0.1
//     1.5    10        1      0.1
//     1.8    10        1      0.1
//
var RoundFloor Rounder = roundFloor

// RoundCeil rounds towards positive infinity; that is, returns the
// smallest Dec not smaller than the result represented by quo and rem.
//
// The following table shows examples of the results for
// Quo(x, y, ScaleFixed(scale), RoundCeil).
//
//      x      y    scale   result
//  ------------------------------
//    -1.8    10        1     -0.1
//    -1.5    10        1     -0.1
//    -1.2    10        1     -0.1
//    -1.0    10        1     -0.1
//    -0.8    10        1     -0.0
//    -0.5    10        1     -0.0
//    -0.2    10        1     -0.0
//     0.0    10        1      0.0
//     0.2    10        1      0.1
//     0.5    10        1      0.1
//     0.8    10        1      0.1
//     1.0    10        1      0.1
//     1.2    10        1      0.2
//     1.5    10        1      0.2
//     1.8    10        1      0.2
//
var RoundCeil Rounder = roundCeil

var intSign = []*big.Int{big.NewInt(-1), big.NewInt(0), big.NewInt(1)}

var roundExact = rounder{true,
	func(z, q *Dec, rA, rB *big.Int) *Dec {
		if rA.Sign() != 0 {
			return nil
		}
		return z.move(q)
	}}

var roundDown = rounder{false,
	func(z, q *Dec, rA, rB *big.Int) *Dec {
		return z.move(q)
	}}

var roundUp = rounder{true,
	func(z, q *Dec, rA, rB *big.Int) *Dec {
		z.move(q)
		if rA.Sign() != 0 {
			z.Unscaled().Add(z.Unscaled(), intSign[rA.Sign()*rB.Sign()+1])
		}
		return z
	}}

var roundHalfDown = rounder{true,
	func(z, q *Dec, rA, rB *big.Int) *Dec {
		z.move(q)
		brA, brB := rA.BitLen(), rB.BitLen()
		if brA < brB-1 {
			// brA < brB-1 => |rA| < |rB/2|
			return z
		}
		adjust := false
		srA, srB := rA.Sign(), rB.Sign()
		s := srA * srB
		if brA == brB-1 {
			rA2 := new(big.Int).Lsh(rA, 1)
			if s < 0 {
				rA2.Neg(rA2)
			}
			if rA2.Cmp(rB)*srB > 0 {
				adjust = true
			}
		} else {
			// brA > brB-1 => |rA| > |rB/2|
			adjust = true
		}
		if adjust {
			z.Unscaled().Add(z.Unscaled(), intSign[s+1])
		}
		return z
	}}

var roundHalfUp = rounder{true,
	func(z, q *Dec, rA, rB *big.Int) *Dec {
		z.move(q)
		brA, brB := rA.BitLen(), rB.BitLen()
		if brA < brB-1 {
			// brA < brB-1 => |rA| < |rB/2|
			return z
		}
		adjust := false
		srA, srB := rA.Sign(), rB.Sign()
		s := srA * srB
		if brA == brB-1 {
			rA2 := new(big.Int).Lsh(rA, 1)
			if s < 0 {
				rA2.Neg(rA2)
			}
			if rA2.Cmp(rB)*srB >= 0 {
				adjust = true
			}
		} else {
			// brA > brB-1 => |rA| > |rB/2|
			adjust = true
		}
		if adjust {
			z.Unscaled().Add(z.Unscaled(), intSign[s+1])
		}
		return z
	}}

var roundFloor = rounder{true,
	func(z, q *Dec, rA, rB *big.Int) *Dec {
		z.move(q)
		if rA.Sign()*rB.Sign() < 0 {
			z.Unscaled().Add(z.Unscaled(), intSign[0])
		}
		return z
	}}

var roundCeil = rounder{true,
	func(z, q *Dec, rA, rB *big.Int) *Dec {
		z.move(q)
		if rA.Sign()*rB.Sign() > 0 {
			z.Unscaled().Add(z.Unscaled(), intSign[2])
		}
		return z
	}}

func upscale(a, b *Dec) (*Dec, *Dec) {
	if a.Scale() == b.Scale() {
		return a, b
	}
	if a.Scale() > b.Scale() {
		bb := b.rescale(a.Scale())
		return a, bb
	}
	aa := a.rescale(b.Scale())
	return aa, b
}

func exp10(x Scale) *big.Int {
	if int(x) < len(exp10cache) {
		return &exp10cache[int(x)]
	}
	return new(big.Int).Exp(bigInt[10], big.NewInt(int64(x)), nil)
}

func (x *Dec) rescale(newScale Scale) *Dec {
	shift := newScale - x.Scale()
	switch {
	case shift < 0:
		e := exp10(-shift)
		return NewDec(new(big.Int).Quo(x.Unscaled(), e), newScale)
	case shift > 0:
		e := exp10(shift)
		return NewDec(new(big.Int).Mul(x.Unscaled(), e), newScale)
	}
	return x
}

var zeros = []byte("00000000000000000000000000000000" +
	"00000000000000000000000000000000")
var lzeros = Scale(len(zeros))

func appendZeros(s []byte, n Scale) []byte {
	for i := Scale(0); i < n; i += lzeros {
		if n > i+lzeros {
			s = append(s, zeros...)
		} else {
			s = append(s, zeros[0:n-i]...)
		}
	}
	return s
}

func (x *Dec) String() string {
	if x == nil {
		return "<nil>"
	}
	scale := x.Scale()
	s := []byte(x.Unscaled().String())
	if scale <= 0 {
		if scale != 0 && x.unscaled.Sign() != 0 {
			s = appendZeros(s, -scale)
		}
		return string(s)
	}
	negbit := Scale(-((x.Sign() - 1) / 2))
	// scale > 0
	lens := Scale(len(s))
	if lens-negbit <= scale {
		ss := make([]byte, 0, scale+2)
		if negbit == 1 {
			ss = append(ss, '-')
		}
		ss = append(ss, '0', '.')
		ss = appendZeros(ss, scale-lens+negbit)
		ss = append(ss, s[negbit:]...)
		return string(ss)
	}
	// lens > scale
	ss := make([]byte, 0, lens+1)
	ss = append(ss, s[:lens-scale]...)
	ss = append(ss, '.')
	ss = append(ss, s[lens-scale:]...)
	return string(ss)
}

// Format is a support routine for fmt.Formatter. It accepts the decimal
// formats 'd' and 'f', and handles both equivalently.
// Width, precision, flags and bases 2, 8, 16 are not supported.
func (x *Dec) Format(s fmt.State, ch rune) {
	if ch != 'd' && ch != 'f' && ch != 'v' && ch != 's' {
		fmt.Fprintf(s, "%%!%c(dec.Dec=%s)", ch, x.String())
		return
	}
	fmt.Fprintf(s, x.String())
}

func (z *Dec) scan(r io.RuneScanner) (*Dec, error) {
	unscaled := make([]byte, 0, 256) // collects chars of unscaled as bytes
	dp, dg := -1, -1                 // indexes of decimal point, first digit
loop:
	for {
		ch, _, err := r.ReadRune()
		if err == io.EOF {
			break loop
		}
		if err != nil {
			return nil, err
		}
		switch {
		case ch == '+' || ch == '-':
			if len(unscaled) > 0 || dp >= 0 { // must be first character
				r.UnreadRune()
				break loop
			}
		case ch == '.':
			if dp >= 0 {
				r.UnreadRune()
				break loop
			}
			dp = len(unscaled)
			continue // don't add to unscaled
		case ch >= '0' && ch <= '9':
			if dg == -1 {
				dg = len(unscaled)
			}
		default:
			r.UnreadRune()
			break loop
		}
		unscaled = append(unscaled, byte(ch))
	}
	if dg == -1 {
		return nil, fmt.Errorf("no digits read")
	}
	if dp >= 0 {
		z.SetScale(Scale(len(unscaled) - dp))
	} else {
		z.SetScale(0)
	}
	_, ok := z.Unscaled().SetString(string(unscaled), 10)
	if !ok {
		return nil, fmt.Errorf("invalid decimal: %s", string(unscaled))
	}
	return z, nil
}

// SetString sets z to the value of s, interpreted as a decimal (base 10),
// and returns z and a boolean indicating success. The scale of z is the
// number of digits after the decimal point (including any trailing 0s), 
// or 0 if there is no decimal point. If SetString fails, the value of z
// is undefined but the returned value is nil.
func (z *Dec) SetString(s string) (*Dec, bool) {
	r := strings.NewReader(s)
	_, err := z.scan(r)
	if err != nil {
		return nil, false
	}
	_, _, err = r.ReadRune()
	if err != io.EOF {
		return nil, false
	}
	// err == io.EOF => scan consumed all of s
	return z, true
}

// Scan is a support routine for fmt.Scanner; it sets z to the value of
// the scanned number. It accepts the decimal formats 'd' and 'f', and 
// handles both equivalently. Bases 2, 8, 16 are not supported.
// The scale of z is the number of digits after the decimal point
// (including any trailing 0s), or 0 if there is no decimal point.
func (z *Dec) Scan(s fmt.ScanState, ch rune) error {
	if ch != 'd' && ch != 'f' && ch != 's' && ch != 'v' {
		return fmt.Errorf("Dec.Scan: invalid verb '%c'", ch)
	}
	s.SkipSpace()
	_, err := z.scan(s)
	return err
}

// Gob encoding version
const decGobVersion byte = 1

func scaleBytes(s Scale) []byte {
	buf := make([]byte, scaleSize)
	i := scaleSize
	for j := 0; j < scaleSize; j++ {
		i--
		buf[i] = byte(s)
		s >>= 8
	}
	return buf
}

func scale(b []byte) (s Scale) {
	for j := 0; j < scaleSize; j++ {
		s <<= 8
		s |= Scale(b[j])
	}
	return
}

// GobEncode implements the gob.GobEncoder interface.
func (x *Dec) GobEncode() ([]byte, error) {
	buf, err := x.Unscaled().GobEncode()
	if err != nil {
		return nil, err
	}
	buf = append(append(buf, scaleBytes(x.Scale())...), decGobVersion)
	return buf, nil
}

// GobDecode implements the gob.GobDecoder interface.
func (z *Dec) GobDecode(buf []byte) error {
	if len(buf) == 0 {
		return fmt.Errorf("Dec.GobDecode: no data")
	}
	b := buf[len(buf)-1]
	if b != decGobVersion {
		return fmt.Errorf("Dec.GobDecode: encoding version %d not supported", b)
	}
	l := len(buf) - scaleSize - 1
	err := z.Unscaled().GobDecode(buf[:l])
	if err != nil {
		return err
	}
	z.SetScale(scale(buf[l : l+scaleSize]))
	return nil
}
