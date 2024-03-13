package sdl

import "math/bits"

/**
 *  Get the index of the most significant bit. Result is undefined when called
 *  with 0. This operation can also be stated as "count leading zeroes" and
 *  "log base 2".
 *
 *  \return the index of the most significant bit, or -1 if the value is 0.
 */
func SDL_MostSignificantBitIndex32(x uint32) int {
	if x == 0 {
		return -1
	}
	return bits.Len32(x)
}

func SDL_HasExactlyOneBitSet32(x uint32) bool {
	return (x != 0) && ((x & (x - 1)) == 0)
}
