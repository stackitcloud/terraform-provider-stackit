package testdestroy

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-testing/terraform"
	secretsmanager "github.com/stackitcloud/stackit-sdk-go/services/secretsmanager/v1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func testAccCheckSecretsManagerDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := secretsmanager.NewAPIClient(testutil.NewConfigBuilder().BuildClientOptions(testutil.SecretsManagerCustomEndpoint, true)...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	instancesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_secretsmanager_instance" {
			continue
		}
		// instance terraform ID: "[project_id],[instance_id]"
		instanceId := strings.Split(rs.Primary.ID, core.Separator)[1]
		instancesToDestroy = append(instancesToDestroy, instanceId)
	}

	instancesResp, err := client.DefaultAPI.ListInstances(ctx, testutil.ProjectId).Execute()
	if err != nil {
		return fmt.Errorf("getting instancesResp: %w", err)
	}

	for _, instance := range instancesResp.Instances {
		if slices.Contains(instancesToDestroy, instance.Id) {
			err := client.DefaultAPI.DeleteInstance(ctx, testutil.ProjectId, instance.Id).Execute()
			if err != nil {
				return fmt.Errorf("destroying instance %s during CheckDestroy: %w", instance.Id, err)
			}
		}
	}
	return nil
}
