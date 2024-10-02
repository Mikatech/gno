package main

import "fmt"

func main() {
	t := struct {
		Name string
		Next interface{}
	}{}
	t.Next = &t
	fmt.Printf("%v", t)
}

// Output:
// { <address>}
