package main

import (
	"cartapi/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		panic(err)
	}
}
