package main

import (
	"github.com/breml/errchkjson"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(errchkjson.NewAnalyzer())
}
