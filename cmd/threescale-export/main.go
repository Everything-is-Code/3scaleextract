package main

import (
	"os"

	"github.com/fmeneses/3scaleextract/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
