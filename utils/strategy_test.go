package utils

import (
	"fmt"
	"testing"
)

func TestRandInt(t *testing.T) {

	fmt.Println("With debug false:")
	fmt.Println("No1:", RandInt(40, false))
	fmt.Println("No2:", RandInt(40, false))
	fmt.Println("No3:", RandInt(40, false))
	fmt.Println("No4:", RandInt(40, false))

	fmt.Println("With debug true:")
	fmt.Println("No1:", RandInt(40, true))
	fmt.Println("No2:", RandInt(40, true))
	fmt.Println("No3:", RandInt(40, true))
	fmt.Println("No4:", RandInt(40, true))
}
