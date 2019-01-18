// +build ignore

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "stderr for test")
	os.Exit(2)
}
