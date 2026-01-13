package main

import (
	"fmt"
	"os"
)

func debugLog(format string, args ...interface{}) {
	f, err := os.OpenFile("debug_trace.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	fmt.Fprintf(f, format+"\n", args...)
}
