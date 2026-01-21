package main

import (
	"log"
	"os"

	"github.com/mhenselin/terraform-provider-stackitprivatepreview/tools"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "builder",
		Short:             "...",
		Long:              "...",
		SilenceErrors:     true, // Error is beautified in a custom way before being printed
		SilenceUsage:      true,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return tools.Build()
		},
	}
	cmd.SetOut(os.Stdout)
	return cmd
}

func main() {
	cmd := NewRootCmd()
	err := cmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
