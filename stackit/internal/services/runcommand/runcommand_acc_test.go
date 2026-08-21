package runcommand_test

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	runcommand "github.com/stackitcloud/stackit-sdk-go/services/runcommand/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

//go:embed testdata/action.tf
var actionConfig string

func TestAccRunCommandAction(t *testing.T) {
	randSuffix := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	testConfigVars := config.Variables{
		"project_id":        config.StringVariable(testutil.ProjectId),
		"server_name":       config.StringVariable(fmt.Sprintf("tf-acc-srv-%s", randSuffix)),
		"network_name":      config.StringVariable(fmt.Sprintf("tf-acc-net-%s", randSuffix)),
		"machine_type":      config.StringVariable("g2i.1"),
		"region":            config.StringVariable(testutil.Region),
		"availability_zone": config.StringVariable("eu01-1"),
		"script":            config.StringVariable("echo 'acceptance test' > /root/acc-test.txt"),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServerDestroy,
		Steps: []resource.TestStep{
			{
				Config:          testutil.NewConfigBuilder().EnableBetaResources(true).BuildProviderConfig() + "\n" + actionConfig,
				ConfigVariables: testConfigVars,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_server.server", "server_id"),
					resource.TestCheckResourceAttr("stackit_server.server", "project_id", testutil.ProjectId),
					resource.TestCheckResourceAttr("stackit_server.server", "availability_zone", "eu01-1"),
					resource.TestCheckResourceAttr("stackit_network_interface.nic", "security", "false"),
					testAccCheckRunCommandExecuted,
				),
			},
		},
	})
}

// testAccCheckRunCommandExecuted verifies that at least one command was recorded
// for the server after the action_trigger fired.
func testAccCheckRunCommandExecuted(s *terraform.State) error {
	ctx := context.Background()

	client, err := runcommand.NewAPIClient(
		testutil.NewConfigBuilder().BuildClientOptions(testutil.RunCommandCustomEndpoint, true)...,
	)
	if err != nil {
		return fmt.Errorf("creating run command client: %w", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_server" {
			continue
		}
		// Terraform ID format: "[project_id],[region],[server_id]"
		parts := strings.Split(rs.Primary.ID, core.Separator)
		if len(parts) < 3 {
			return fmt.Errorf("unexpected server ID format: %s", rs.Primary.ID)
		}
		projectId, serverId := parts[0], parts[2]

		cmdsResp, err := client.DefaultAPI.ListCommands(ctx, projectId, serverId).Execute()
		if err != nil {
			return fmt.Errorf("listing commands for server %s: %w", serverId, err)
		}
		if cmdsResp == nil || len(cmdsResp.GetItems()) == 0 {
			return fmt.Errorf("expected at least one command for server %s, got none", serverId)
		}
		return nil
	}

	return fmt.Errorf("no stackit_server resource found in state")
}

func testAccCheckServerDestroy(s *terraform.State) error {
	ctx := context.Background()

	client, err := iaas.NewAPIClient(
		testutil.NewConfigBuilder().BuildClientOptions(testutil.IaaSCustomEndpoint, false)...,
	)
	if err != nil {
		return fmt.Errorf("creating iaas client: %w", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_server" {
			continue
		}
		parts := strings.Split(rs.Primary.ID, core.Separator)
		if len(parts) < 3 {
			continue
		}
		projectId, region, serverId := parts[0], parts[1], parts[2]

		_, err := client.DefaultAPI.GetServer(ctx, projectId, region, serverId).Execute()
		if err == nil {
			return fmt.Errorf("server %s still exists after destroy", serverId)
		}
	}

	return nil
}
