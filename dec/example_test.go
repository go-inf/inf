package dec_test

import (
	"fmt"
	"log"
)

import "code.google.com/p/godec/dec"

func ExampleDec_SetString() {
	d := new(dec.Dec)
	d.SetString("012345.67890") // decimal; leading 0 ignored; trailing 0 kept
	fmt.Println(d)
	// Output: 12345.67890
}

func ExampleDec_Scan() {
	// The Scan function is rarely used directly;
	// the fmt package recognizes it as an implementation of fmt.Scanner.
	d := new(dec.Dec)
	_, err := fmt.Sscan("184467440.73709551617", d)
	if err != nil {
		log.Println("error scanning value:", err)
	} else {
		fmt.Println(d)
	}
	// Output: 184467440.73709551617
}

func ExampleDec_Quo_scale2RoundDown() {
	// 10 / 3 is an infinite decimal; it has no exact Dec representation
	x, y := dec.NewDecInt64(10), dec.NewDecInt64(3)
	// use 2 digits beyond the decimal point, round towards 0
	z := new(dec.Dec).Quo(x, y, dec.Scale(2), dec.RoundDown)
	fmt.Println(z)
	// Output: 3.33
}

func ExampleDec_Quo_scale2RoundCeil() {
	// -42 / 400 is an finite decimal with 3 digits beyond the decimal point
	x, y := dec.NewDecInt64(-42), dec.NewDecInt64(400)
	// use 2 digits beyond decimal point, round towards positive infinity
	z := new(dec.Dec).Quo(x, y, dec.Scale(2), dec.RoundCeil)
	fmt.Println(z)
	// Output: -0.10
}

func ExampleDec_QuoExact_ok() {
	// 1 / 25 is a finite decimal; it has exact Dec representation
	x, y := dec.NewDecInt64(1), dec.NewDecInt64(25)
	z := new(dec.Dec).QuoExact(x, y)
	fmt.Println(z)
	// Output: 0.04
}

func ExampleDec_QuoExact_fail() {
	// 1 / 3 is an infinite decimal; it has no exact Dec representation
	x, y := dec.NewDecInt64(1), dec.NewDecInt64(3)
	z := new(dec.Dec).QuoExact(x, y)
	fmt.Println(z)
	// Output: <nil>
}
