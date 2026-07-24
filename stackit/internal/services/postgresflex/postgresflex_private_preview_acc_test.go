package postgresflex

import (
	_ "embed"
	"fmt"
	"maps"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	kms "github.com/stackitcloud/stackit-sdk-go/services/kms/v1api"
	postgresflex "github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v3api"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	// Instance

	//go:embed testdata/resource-instance-private-preview-max.tf
	resourceInstancePrivatePreviewMaxConfig string
)

// Instance - MAX
var testConfigInstanceVarsPrivatePreviewMax = config.Variables{
	"project_id":            config.StringVariable(testutil.ProjectId),
	"kek_key_version":       config.StringVariable("1"),
	"service_account_email": config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"keyring_display_name":  config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"display_name":          config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"algorithm":             config.StringVariable(string(kms.ALGORITHM_AES_256_GCM)),
	"protection":            config.StringVariable("software"),
	"purpose":               config.StringVariable(string(kms.PURPOSE_SYMMETRIC_ENCRYPT_DECRYPT)),

	"name":             config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"acl":              config.StringVariable("192.168.0.0/24"),
	"access_scope":     config.StringVariable(string(postgresflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
	"backup_schedule":  config.StringVariable("0 16 * * *"),
	"flavor_id":        config.StringVariable("4.8-replica"),
	"flavor_cpu":       config.IntegerVariable(4),
	"flavor_ram":       config.IntegerVariable(8),
	"replicas":         config.IntegerVariable(3),
	"storage_class":    config.StringVariable("premium-perf2-stackit"),
	"storage_size":     config.IntegerVariable(5),
	"instance_version": config.StringVariable("16"),
	"retention_days":   config.IntegerVariable(40),
	"region":           config.StringVariable(testutil.Region),
}

var testConfigInstanceVarsPrivatePreviewMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigInstanceVarsPrivatePreviewMax)
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf(
		"%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"]),
	))
	updatedConfig["acl"] = config.StringVariable("192.160.2.0/24")
	updatedConfig["backup_schedule"] = config.StringVariable("1 0 * * *")
	updatedConfig["flavor_id"] = config.StringVariable("4.8")
	updatedConfig["flavor_cpu"] = config.IntegerVariable(8)
	updatedConfig["flavor_ram"] = config.IntegerVariable(16)
	updatedConfig["replicas"] = config.IntegerVariable(1)
	updatedConfig["storage_size"] = config.IntegerVariable(11)
	updatedConfig["instance_version"] = config.StringVariable("17")
	updatedConfig["retention_days"] = config.IntegerVariable(32)
	return updatedConfig
}()

func TestAccPostgresFlexInstancePrivatePreviewMax(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigInstanceVarsPrivatePreviewMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Region(testutil.Region).BuildProviderConfig(), resourceInstancePrivatePreviewMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.instance", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance with flavor id
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["name"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["acl"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "network.acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "network.acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["acl"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "network.access_scope"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["backup_schedule"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "flavor_id", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["flavor_id"])),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "flavor.id"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "flavor.description"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "flavor.cpu"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "flavor.ram"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "connection_info.write.host"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "connection_info.write.port"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "replicas"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "retention_days", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["retention_days"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "storage.class", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["storage_class"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "storage.size", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["storage_size"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "version", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["instance_version"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "region", testutil.Region),

					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "encryption.kek_key_id"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "encryption.kek_keyring_id"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "encryption.kek_key_version", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["kek_key_version"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "encryption.service_account", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["service_account_email"])),
				),
			},
			// data source
			{
				ConfigVariables: testConfigInstanceVarsPrivatePreviewMax,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.instance", plancheck.ResourceActionNoop),
					},
				},
				Config: fmt.Sprintf(`
					%s

					%s
					data "stackit_postgresflex_instance" "instance" {
						project_id     = stackit_postgresflex_instance.instance.project_id
						instance_id    = stackit_postgresflex_instance.instance.instance_id
					}
					`,
					testutil.NewConfigBuilder().Region(testutil.Region).BuildProviderConfig(), resourceInstancePrivatePreviewMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance with flavor id
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["acl"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "network.acl.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "network.acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["acl"])),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "network.access_scope"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["backup_schedule"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "flavor_id", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["flavor_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "flavor.id"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "flavor.cpu"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "flavor.ram"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "connection_info.write.host"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "connection_info.write.port"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "replicas"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "retention_days", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["retention_days"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "storage.class", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["storage_class"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "storage.size", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["storage_size"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "version", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["instance_version"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "region", testutil.Region),

					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "encryption.kek_key_id"),
					resource.TestCheckResourceAttrSet("data.stackit_postgresflex_instance.instance", "encryption.kek_keyring_id"),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "encryption.kek_key_version", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["kek_key_version"])),
					resource.TestCheckResourceAttr("data.stackit_postgresflex_instance.instance", "encryption.service_account", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMax["service_account_email"])),
				),
			},
			// Import with flavor id
			{
				ConfigVariables: testConfigInstanceVarsPrivatePreviewMax,
				ResourceName:    "stackit_postgresflex_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_postgresflex_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_postgresflex_instance.instance")
					}

					projectId, ok := r.Primary.Attributes["project_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute project_id")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					return fmt.Sprintf("%s,%s,%s", projectId, region, instanceId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"flavor", "replicas"},
			},
			// Update
			{
				ConfigVariables: testConfigInstanceVarsPrivatePreviewMaxUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.NewConfigBuilder().Region(testutil.Region).BuildProviderConfig(), resourceInstancePrivatePreviewMaxConfig),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_postgresflex_instance.instance", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance with flavor id
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMaxUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMaxUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMaxUpdated["acl"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "network.acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "network.acl.0", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMaxUpdated["acl"])),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "network.access_scope"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMaxUpdated["backup_schedule"])),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "flavor.id"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "flavor.description"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "flavor.cpu"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "flavor.ram"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "connection_info.write.host"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "connection_info.write.port"),
					resource.TestCheckNoResourceAttr("stackit_postgresflex_instance.instance", "replicas"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "retention_days", retentionDaysDefault),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "storage.class", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMaxUpdated["storage_class"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "storage.size", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMaxUpdated["storage_size"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "version", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMaxUpdated["instance_version"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "region", testutil.Region),

					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "encryption.kek_key_id"),
					resource.TestCheckResourceAttrSet("stackit_postgresflex_instance.instance", "encryption.kek_keyring_id"),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "encryption.kek_key_version", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMaxUpdated["kek_key_version"])),
					resource.TestCheckResourceAttr("stackit_postgresflex_instance.instance", "encryption.service_account", testutil.ConvertConfigVariable(testConfigInstanceVarsPrivatePreviewMaxUpdated["service_account_email"]))),
			},
			// Deletion is done by the framework implicitly
		},
	})
}
