package main

import "fmt"

func logAction(format string, a ...interface{}) {
	fmt.Println(`►`, fmt.Sprintf(format, a...))
}

func logGenerate(format string, a ...interface{}) {
	fmt.Println(`✚`, fmt.Sprintf(format, a...))
}

func logWaiting(format string, a ...interface{}) {
	fmt.Println(`◎`, fmt.Sprintf(format, a...))
}

func logSuccess(format string, a ...interface{}) {
	fmt.Println(`✔`, fmt.Sprintf(format, a...))
}

func logFailure(format string, a ...interface{}) {
	fmt.Println(`✗`, fmt.Sprintf(format, a...))
}
