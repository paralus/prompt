// +build ignore

package main

import (
	"log"
	"net/http"

	"github.com/shurcooL/vfsgen"
)

var devFS http.FileSystem = http.Dir("dev/data")

func main() {

	err := vfsgen.Generate(devFS, vfsgen.Options{
		Filename:     "dev/fixtures.go",
		PackageName:  "dev",
		VariableName: "DevFS",
	})
	if err != nil {
		log.Fatalln(err)
	}

}
