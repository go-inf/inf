package dec

import (
	"math/big"
	"testing"
)

var decRounderInputs = [...]struct {
	quo    *Dec
	rA, rB *big.Int
}{
	// examples from go language spec
	{NewDec(big.NewInt(1), 0), big.NewInt(2), big.NewInt(3)},   //  5 /  3
	{NewDec(big.NewInt(-1), 0), big.NewInt(-2), big.NewInt(3)}, // -5 /  3
	{NewDec(big.NewInt(-1), 0), big.NewInt(2), big.NewInt(-3)}, //  5 / -3
	{NewDec(big.NewInt(1), 0), big.NewInt(-2), big.NewInt(-3)}, // -5 / -3
	// examples from godoc
	{NewDec(big.NewInt(-1), 1), big.NewInt(-8), big.NewInt(10)},
	{NewDec(big.NewInt(-1), 1), big.NewInt(-5), big.NewInt(10)},
	{NewDec(big.NewInt(-1), 1), big.NewInt(-2), big.NewInt(10)},
	{NewDec(big.NewInt(0), 1), big.NewInt(-8), big.NewInt(10)},
	{NewDec(big.NewInt(0), 1), big.NewInt(-5), big.NewInt(10)},
	{NewDec(big.NewInt(0), 1), big.NewInt(-2), big.NewInt(10)},
	{NewDec(big.NewInt(0), 1), big.NewInt(0), big.NewInt(1)},
	{NewDec(big.NewInt(0), 1), big.NewInt(2), big.NewInt(10)},
	{NewDec(big.NewInt(0), 1), big.NewInt(5), big.NewInt(10)},
	{NewDec(big.NewInt(0), 1), big.NewInt(8), big.NewInt(10)},
	{NewDec(big.NewInt(1), 1), big.NewInt(2), big.NewInt(10)},
	{NewDec(big.NewInt(1), 1), big.NewInt(5), big.NewInt(10)},
	{NewDec(big.NewInt(1), 1), big.NewInt(8), big.NewInt(10)},
}

var decRounderResults = [...]struct {
	rounder Rounder
	results [len(decRounderInputs)]*Dec
}{
	{RoundExact, [...]*Dec{nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil,
		NewDec(big.NewInt(0), 1), nil, nil, nil, nil, nil, nil}},
	{RoundDown, [...]*Dec{
		NewDecInt64(1), NewDecInt64(-1), NewDecInt64(-1), NewDecInt64(1),
		NewDec(big.NewInt(-1), 1), NewDec(big.NewInt(-1), 1), NewDec(big.NewInt(-1), 1),
		NewDec(big.NewInt(0), 1), NewDec(big.NewInt(0), 1), NewDec(big.NewInt(0), 1),
		NewDec(big.NewInt(0), 1),
		NewDec(big.NewInt(0), 1), NewDec(big.NewInt(0), 1), NewDec(big.NewInt(0), 1),
		NewDec(big.NewInt(1), 1), NewDec(big.NewInt(1), 1), NewDec(big.NewInt(1), 1)}},
	{RoundUp, [...]*Dec{
		NewDecInt64(2), NewDecInt64(-2), NewDecInt64(-2), NewDecInt64(2),
		NewDec(big.NewInt(-2), 1), NewDec(big.NewInt(-2), 1), NewDec(big.NewInt(-2), 1),
		NewDec(big.NewInt(-1), 1), NewDec(big.NewInt(-1), 1), NewDec(big.NewInt(-1), 1),
		NewDec(big.NewInt(0), 1),
		NewDec(big.NewInt(1), 1), NewDec(big.NewInt(1), 1), NewDec(big.NewInt(1), 1),
		NewDec(big.NewInt(2), 1), NewDec(big.NewInt(2), 1), NewDec(big.NewInt(2), 1)}},
	{RoundHalfDown, [...]*Dec{
		NewDecInt64(2), NewDecInt64(-2), NewDecInt64(-2), NewDecInt64(2),
		NewDec(big.NewInt(-2), 1), NewDec(big.NewInt(-1), 1), NewDec(big.NewInt(-1), 1),
		NewDec(big.NewInt(-1), 1), NewDec(big.NewInt(0), 1), NewDec(big.NewInt(0), 1),
		NewDec(big.NewInt(0), 1),
		NewDec(big.NewInt(0), 1), NewDec(big.NewInt(0), 1), NewDec(big.NewInt(1), 1),
		NewDec(big.NewInt(1), 1), NewDec(big.NewInt(1), 1), NewDec(big.NewInt(2), 1)}},
	{RoundHalfUp, [...]*Dec{
		NewDecInt64(2), NewDecInt64(-2), NewDecInt64(-2), NewDecInt64(2),
		NewDec(big.NewInt(-2), 1), NewDec(big.NewInt(-2), 1), NewDec(big.NewInt(-1), 1),
		NewDec(big.NewInt(-1), 1), NewDec(big.NewInt(-1), 1), NewDec(big.NewInt(0), 1),
		NewDec(big.NewInt(0), 1),
		NewDec(big.NewInt(0), 1), NewDec(big.NewInt(1), 1), NewDec(big.NewInt(1), 1),
		NewDec(big.NewInt(1), 1), NewDec(big.NewInt(2), 1), NewDec(big.NewInt(2), 1)}},
	{RoundHalfEven, [...]*Dec{
		NewDecInt64(2), NewDecInt64(-2), NewDecInt64(-2), NewDecInt64(2),
		NewDec(big.NewInt(-2), 1), NewDec(big.NewInt(-2), 1), NewDec(big.NewInt(-1), 1),
		NewDec(big.NewInt(-1), 1), NewDec(big.NewInt(0), 1), NewDec(big.NewInt(0), 1),
		NewDec(big.NewInt(0), 1),
		NewDec(big.NewInt(0), 1), NewDec(big.NewInt(0), 1), NewDec(big.NewInt(1), 1),
		NewDec(big.NewInt(1), 1), NewDec(big.NewInt(2), 1), NewDec(big.NewInt(2), 1)}},
	{RoundFloor, [...]*Dec{
		NewDecInt64(1), NewDecInt64(-2), NewDecInt64(-2), NewDecInt64(1),
		NewDec(big.NewInt(-2), 1), NewDec(big.NewInt(-2), 1), NewDec(big.NewInt(-2), 1),
		NewDec(big.NewInt(-1), 1), NewDec(big.NewInt(-1), 1), NewDec(big.NewInt(-1), 1),
		NewDec(big.NewInt(0), 1),
		NewDec(big.NewInt(0), 1), NewDec(big.NewInt(0), 1), NewDec(big.NewInt(0), 1),
		NewDec(big.NewInt(1), 1), NewDec(big.NewInt(1), 1), NewDec(big.NewInt(1), 1)}},
	{RoundCeil, [...]*Dec{
		NewDecInt64(2), NewDecInt64(-1), NewDecInt64(-1), NewDecInt64(2),
		NewDec(big.NewInt(-1), 1), NewDec(big.NewInt(-1), 1), NewDec(big.NewInt(-1), 1),
		NewDec(big.NewInt(0), 1), NewDec(big.NewInt(0), 1), NewDec(big.NewInt(0), 1),
		NewDec(big.NewInt(0), 1),
		NewDec(big.NewInt(1), 1), NewDec(big.NewInt(1), 1), NewDec(big.NewInt(1), 1),
		NewDec(big.NewInt(2), 1), NewDec(big.NewInt(2), 1), NewDec(big.NewInt(2), 1)}},
}

func TestDecRounders(t *testing.T) {
	for i, a := range decRounderResults {
		for j, input := range decRounderInputs {
			q := new(Dec).Set(input.quo)
			rA, rB := new(big.Int).Set(input.rA), new(big.Int).Set(input.rB)
			res := a.rounder.Round(new(Dec), q, rA, rB)
			if a.results[j] == nil && res == nil {
				continue
			}
			if (a.results[j] == nil && res != nil) ||
				(a.results[j] != nil && res == nil) ||
				a.results[j].Cmp(res) != 0 {
				t.Errorf("#%d,%d Rounder got %v; expected %v", i, j, res, a.results[j])
			}
		}
	}
}
