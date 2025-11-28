package access_token_test

import (
	_ "embed"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/ephemeral_resource.tf
var ephemeralResourceConfig string

var testConfigVars = config.Variables{
	"default_region": config.StringVariable(testutil.Region),
}

func TestAccEphemeralAccessToken(t *testing.T) {
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV6ProviderFactories: testutil.TestEphemeralAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:          ephemeralResourceConfig,
				ConfigVariables: testConfigVars,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.example",
						tfjsonpath.New("data").AtMapKey("access_token"),
						knownvalue.NotNull(),
					),
					// JWT access tokens start with "ey" because the first part is base64-encoded JSON that begins with "{".
					statecheck.ExpectKnownValue(
						"echo.example",
						tfjsonpath.New("data").AtMapKey("access_token"),
						knownvalue.StringRegexp(regexp.MustCompile(`^ey`)),
					),
				},
			},
		},
	})
}
