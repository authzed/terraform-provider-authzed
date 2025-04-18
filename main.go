package main

import (
	"context"
	"flag"
	"log"
	"os"

	"terraform-provider-authzed/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "dev"

func init() {
	log.SetOutput(os.Stderr)
}

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run provider with debugger support")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/authzed/authzed",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err)
	}
}
