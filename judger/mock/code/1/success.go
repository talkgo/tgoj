package main

import (
	"fmt"
)

// uncomment to cause CE
func main() {
	var n int
	fmt.Scanf("%d", &n)
	for n > 0 {
		n--
		var a, b int
		fmt.Scanf("%d %d", &a, &b)
		fmt.Println(a + b)
	}
}
