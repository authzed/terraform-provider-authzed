package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"terraform-provider-platform-api/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var (
	version = "dev"
)

func init() {
	// Redirect logs to stderr to avoid breaking the Terraform plugin protocol
	log.SetOutput(os.Stderr)
}

func main() {
	// Set up panic handler to log to stderr
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Panic: %v\n", r)
			fmt.Fprintf(os.Stderr, "Stack: %s\n", debug.Stack())
			os.Exit(1)
		}
	}()

	// Set debug to false by default for normal Terraform operation
	var debug = false

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers")
	flag.Parse()

	// Log important details (to stderr only when in debug mode)
	if debug {
		log.Println("Starting provider with debug =", debug)
	}

	ctx := context.Background()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/authzed/platform-api",
		Debug:   debug,
	}

	err := providerserver.Serve(ctx, provider.New(version), opts)
	if err != nil {
		// Make sure error messages go to stderr as well
		fmt.Fprintf(os.Stderr, "Error serving provider: %v\n", err)
		log.Fatal(err)
	}
}
