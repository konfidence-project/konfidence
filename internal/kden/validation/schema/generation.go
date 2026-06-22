//go:build ignore

package main

import (
	"log"
	"path/filepath"
	"runtime"

	"github.com/konfidence-project/konfidence/internal/kden/validation/schema"
)

func main() {
	_, file, _, _ := runtime.Caller(0)
	dest := filepath.Join(filepath.Dir(file), "../resources/konfidence-artifact-schema.json")
	if err := schema.WriteSchema(dest); err != nil {
		log.Fatal(err)
	}
}
