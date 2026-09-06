package main

import (
	"fmt"

	"github.com/GunithaR/ShipHold/internal/cli"
)

func main() {
	fmt.Println("ShipHold")
	rootCmd := cli.NewRootCommand()

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
