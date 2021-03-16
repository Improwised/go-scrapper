package main

import (
	"fmt"
)

func sum(nums ...int) {
    var s = 0
    for _, n := range nums {
        s = s + n
        fmt.Println(n)
    }
    fmt.Printf("Final = %d\n", s)
}

func main() {

    var arr = []int{1,3,4}

    sum(arr...)
}
