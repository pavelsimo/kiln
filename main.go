package main

import (
	"os"

	"github.com/pavelsimo/kiln/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
