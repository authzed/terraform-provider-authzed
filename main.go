package main

import (
	"context"
	"flag"
	"log"
	"os"

	"terraform-provider-platform-api/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var (
	version = "dev"
)

func main() {
	// Enable debugging by default for development
	var debug = true

	flag.BoolVar(&debug, "debug", true, "set to true to run the provider with support for debuggers")
	flag.Parse()

	// Log important details
	log.Println("Starting provider with debug =", debug)
	log.Println("Environment:", os.Environ())

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/authzed/platform-api",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
