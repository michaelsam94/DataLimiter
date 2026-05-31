package main

import (
	"os"

	"datalimiter/internal/datalimiter"
)

func main() {
	app := datalimiter.NewApp(datalimiter.OSDeps{})
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr))
}
