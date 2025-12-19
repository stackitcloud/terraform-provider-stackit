package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit"
)

var (
	// goreleaser configuration will override this value
	version string = "dev"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "allows debugging the provider")
	flag.Parse()
	err := providerserver.Serve(context.Background(), stackit.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/mhenselin/stackitprivatepreview",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
