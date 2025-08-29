package main

import (
	_ "embed"

	"github.com/zostay/genifest/internal/cmd"
	"github.com/zostay/genifest/internal/config"
)

//go:embed schema/genifest-schema.json
var schemaJSON string

func main() {
	// Initialize schema in config package
	config.InitSchema(schemaJSON)

	cmd.Execute()
}
