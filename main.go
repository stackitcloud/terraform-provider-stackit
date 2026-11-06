package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/stackitcloud/terraform-provider-stackit/stackit"
)

var (
	// goreleaser configuration will override this value
	version string = "dev"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "allows debugging the provider")
	flag.Parse()

	// When wanting to debug the Terraform provider you'll set the '-debug' flag and the provider will print out
	// something like this:
	// > Provider started, to attach Terraform set the TF_REATTACH_PROVIDERS env var:
	// > TF_REATTACH_PROVIDERS='{"registry.terraform.io/stackitcloud/stackit":{"Protocol":"grpc",...'
	// Then you'll have to set that env variable so the Terraform CLI will pick up your running debug provider instance.
	// https://developer.hashicorp.com/terraform/plugin/debugging#running-terraform-with-a-provider-in-debug-mode
	//
	// Since this process is not very intuitive we have some logic here which writes the value of the TF_REATTACH_PROVIDERS
	// variable the provider printed to stdout to a file in the user's home directory so one is able to automatically
	// pick up the env variable using some wrapper-script for the Terraform CLI.
	if debug {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("failed to get home directory: %v", err)
		}

		filePath := filepath.Join(homeDir, ".stackit", "tf-debug-reattach.json")

		// create parent directory in case it doesn't exist
		if err := os.MkdirAll(filepath.Dir(filePath), 0o700); err != nil {
			log.Fatalf("failed to create directory: %v", err)
		}

		// Pipe stdout to intercept TF_REATTACH_PROVIDERS automatically
		r, w, _ := os.Pipe()
		origStdout := os.Stdout
		os.Stdout = w

		go func() {
			re := regexp.MustCompile(`TF_REATTACH_PROVIDERS='([^']+)'`)
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				line := scanner.Text()
				_, err := fmt.Fprintln(origStdout, line) // Keep stdout still visible in IDE/console
				if err != nil {
					log.Fatal(err.Error())
				}

				if matches := re.FindStringSubmatch(line); len(matches) > 1 {
					_ = os.WriteFile(filePath, []byte(matches[1]), 0o600)
				}
			}

			if err := scanner.Err(); err != nil {
				log.Fatal(err.Error())
			}
		}()
	}

	err := providerserver.Serve(context.Background(), stackit.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/stackitcloud/stackit",
		Debug:   debug,
	})

	if err != nil {
		log.Fatal(err.Error())
	}
}
