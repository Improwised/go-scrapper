package main

import (
	"fmt"
)

type P struct {
	Age int
}

func (me *P) SetAge(n int) {
	me.Age = n
}

type C struct{
	P
}

func main() {
	child := C{}
	child.SetAge(10)
	fmt.Println(child)
}