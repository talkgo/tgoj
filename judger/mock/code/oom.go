package main

import (
	"fmt"
)

// uncomment to cause CE
func main() {
	a := make([]int64, 1<<40)
	for i := 0; i < 10; i++ {
		a[i] = 2
	}
	for i := 1<<24 - 1; i >= 0; i-- {
		a[i] = int64(i)
	}

	var n int
	fmt.Scanf("%d", &n)
	for n > 0 {
		n--
		var a, b int
		fmt.Scanf("%d %d", &a, &b)
		fmt.Println(a + b)
	}
}
