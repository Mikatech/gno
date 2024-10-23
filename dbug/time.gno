package main

import (
	"time"
)

func main() {
	y, m, d := time.Now().Date()
	println("year", y, "month", m, "day", d)
	abs := time.Now().AbsDebug()
	println("AbsDebug", abs)
}

// Output:
// 2009-02-13 23:31:30 +0000 UTC m=+1234567890.000000001
