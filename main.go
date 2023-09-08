package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/stackitcloud/terraform-provider-stackit/stackit"
)

var (
	// goreleaser configuration will override this value
	version string = "dev"
)

func main() {
	err := providerserver.Serve(context.Background(), stackit.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/stackitcloud/stackit",
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
