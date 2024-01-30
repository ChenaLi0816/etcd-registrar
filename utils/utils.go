package utils

import (
	"math/rand"
	"strconv"
)

func RandInt(max int64, debug bool) int64 {
	if debug {
		r := rand.New(rand.NewSource(1))
		return r.Int63n(max)
	} else {
		return rand.Int63n(max)
	}
}

func StringAdd(src string, delta int) string {
	i, _ := strconv.Atoi(src)
	return strconv.Itoa(i + delta)
}

func StringModAdd(src string, delta, mod int) string {
	i, _ := strconv.Atoi(src)
	return strconv.Itoa((i + delta) % mod)
}

func StringCmp(a, b string) int {
	v1, _ := strconv.Atoi(a)
	v2, _ := strconv.Atoi(b)
	if v1 == v2 {
		return 0
	}
	if v1 < v2 {
		return -1
	}
	return 1
}
