package main

import (
	"fmt"

	"github.com/gahojin/go-iso8601duration"
)

func main() {
	a, _ := iso8601duration.ParseString("P0.65D")
	fmt.Print(a.String())
}
