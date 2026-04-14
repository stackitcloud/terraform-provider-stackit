package secretsmanager_test

import (
	_ "embed"
	"maps"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	rolebindings_testing "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iam/rolebindings/v1/rolebindings-testing"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/instance.tf
	instanceConfig string
)

func TestAccSecretsManagerInstanceRoleBindings(t *testing.T) {
	variables := config.Variables{
		"project_id":    config.StringVariable(testutil.ProjectId),
		"instance_name": config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
		"role":          config.StringVariable("owner"),
		"subject":       config.StringVariable(testutil.TestProjectServiceAccountEmail),
	}

	variablesUpdated := func() config.Variables {
		tempConfig := make(config.Variables, len(variables))
		maps.Copy(tempConfig, variables)
		tempConfig["role"] = config.StringVariable("editor")
		return tempConfig
	}

	providerConfig := testutil.NewConfigBuilder().Experiments(testutil.ExperimentIAM).BuildProviderConfig()

	tc := rolebindings_testing.NewRoleBindingAccTestBuilder(providerConfig, "secretsmanager", "instance", "role_binding").
		CreateStep(instanceConfig, variables, "stackit_secretsmanager_instance.instance", "instance_id").
		ImportStep(variables).
		UpdateStep(instanceConfig, variablesUpdated(), "stackit_secretsmanager_instance.instance", "instance_id").
		Build()

	resource.Test(t, tc)
}
