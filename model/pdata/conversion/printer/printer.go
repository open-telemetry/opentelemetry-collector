package main

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"
)

func main() {
	scale, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("usage: %s scale (an integer)\n", os.Args[1])
		os.Exit(1)
	}

	var (
		// size is 2^scale
		size       = int64(1) << scale
		thresholds = make([]uint64, size)

		// constants
		onef = big.NewFloat(1)
		onei = big.NewInt(1)
	)

	newf := func() *big.Float { return &big.Float{} }
	newi := func() *big.Int { return &big.Int{} }
	pow2 := func(x int) *big.Float {
		return newf().SetMantExp(onef, x)
	}
	toInt64 := func(x *big.Float) *big.Int {
		i, _ := x.SetMode(big.ToZero).Int64()
		return big.NewInt(i)
	}
	ipow := func(b *big.Int, p int64) *big.Int {
		r := onei
		for i := int64(0); i < p; i++ {
			r = newi().Mul(r, b)
		}
		return r
	}
	for position := range thresholds {
		// whereas (position/size) in the range [0, 1),
		//   x = 2^(position/size)
		// falls in the range [1, 2).  Equivalently,
		// calculate 2^position, then square-root scale times.
		x := pow2(position)
		for i := 0; i < scale; i++ {
			x = newf().Sqrt(x)
		}

		// Compute the integer value in the range [2^52, 2^53)
		// which is the 52-bit significand of the IEEE float64
		// as an uint64 value plus 2^52.
		scaled := newf().Mul(x, pow2(52))
		ieeeNormalized := toInt64(scaled) // in the range [2^52, 2^53)

		compareTo, _ := pow2(52*int(size) + position).Int(nil)

		if ipow(ieeeNormalized, size).Cmp(compareTo) < 0 {
			ieeeNormalized = newi().Add(ieeeNormalized, onei)
		}

		thresholds[position] = ieeeNormalized.Uint64() & ((uint64(1) << 52) - 1)

		// Validate that this is the correct result by
		// subtracting one, ensure that the value is less than
		// compareTo.
		sigLessOne := newi().Sub(ieeeNormalized, onei)

		// If (ieeeNormalized-1)^size is greater than or equal to the
		// inclusive lower bound
		if ipow(sigLessOne, size).Cmp(compareTo) >= 0 {
			panic("incorrect result")
		}
	}

	fmt.Printf(`package conversion

var exponentialConstants = [%d]uint64{
`, size)

	for pos, value := range thresholds {
		fmt.Printf("\t0x%012x, // significand(2^(%d/%d) == %.016g)\n",
			value,
			pos,
			size,
			math.Float64frombits((uint64(1023)<<52)+value),
		)
	}
	fmt.Printf(`}
`)
}
