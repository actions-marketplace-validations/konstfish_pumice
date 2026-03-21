package main

import (
	"fmt"
	"os"

	"github.com/konstfish/pumice/internal/app"
)

func main() {
	application := app.NewApplication()

	if err := application.LoadConfig(os.Args[1:]); err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := application.Build(); err != nil {
		fmt.Printf("Error building site: %v\n", err)
		os.Exit(1)
	}

	if application.ShouldServe() {
		application.ServeWithLogging(application.GetPort())
	}
}
