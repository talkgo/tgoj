package main

import (
	"fmt"
	"time"
)

// uncomment to cause CE
func main() {
	time.Sleep(time.Second * 2)

	var n int
	fmt.Scanf("%d", &n)
	for n > 0 {
		n--
		var a, b int
		fmt.Scanf("%d %d", &a, &b)
		fmt.Println(a + b)
	}
}
