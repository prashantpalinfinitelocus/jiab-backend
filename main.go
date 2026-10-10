package main

// Ad-hoc testing for testing individual functions and Clients
import (
	"fmt"
	"os"

	"github.com/Infinite-Locus-Product/coca_cola_backend/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
