package utils

import "math/rand"

func RandInt(max int64, debug bool) int64 {
	if debug {
		r := rand.New(rand.NewSource(1))
		return r.Int63n(max)
	} else {
		return rand.Int63n(max)
	}
}
