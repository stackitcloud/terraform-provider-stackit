package sqlserverflex_test

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
	sqlserverflex "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v3api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/resource-max-private-preview.tf
	resourcePrivatePreviewConfig string
)

var testConfigVarsMaxPrivatePreview = config.Variables{
	"project_id":            config.StringVariable(testutil.ProjectId),
	"name":                  config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlphaNum))),
	"acl1":                  config.StringVariable("192.168.0.0/16"),
	"storage_class":         config.StringVariable("premium-perf2-stackit"),
	"storage_size":          config.IntegerVariable(40),
	"server_version":        config.StringVariable("2022"),
	"replicas":              config.IntegerVariable(1),
	"access_scope":          config.StringVariable(string(sqlserverflex.INSTANCENETWORKACCESSSCOPE_PUBLIC)),
	"retention_days":        config.IntegerVariable(32),
	"flavor_id":             config.StringVariable("4.16-Single"),
	"backup_schedule":       config.StringVariable("0 6 * * *"),
	"username":              config.StringVariable(fmt.Sprintf("tf-acc-user-%s", acctest.RandStringFromCharSet(7, acctest.CharSetAlpha))),
	"role":                  config.StringVariable("##STACKIT_LoginManager##"),
	"region":                config.StringVariable(testutil.Region),
	"kek_key_version":       config.StringVariable("1"),
	"service_account_email": config.StringVariable(testutil.TestProjectServiceAccountEmail),

	"keyring_display_name": config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"display_name":         config.StringVariable("tf-acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)),
	"algorithm":            config.StringVariable(string(kms.ALGORITHM_AES_256_GCM)),
	"protection":           config.StringVariable("software"),
	"purpose":              config.StringVariable(string(kms.PURPOSE_SYMMETRIC_ENCRYPT_DECRYPT)),
}

func configVarsMaxPrivatePreviewUpdated() config.Variables {
	temp := maps.Clone(testConfigVarsMaxPrivatePreview)
	temp["backup_schedule"] = config.StringVariable("0 12 * * *")
	temp["acl1"] = config.StringVariable("192.168.2.0/16")
	temp["retention_days"] = config.IntegerVariable(40)
	return temp
}

func TestAccSQLServerFlexMaxPrivatePreviewResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccChecksqlserverflexDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourcePrivatePreviewConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_sqlserverflex_instance.instance", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("stackit_sqlserverflex_user.user", plancheck.ResourceActionCreate),
					},
				},
				ConfigVariables: testConfigVarsMaxPrivatePreview,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Key and Keyring
					resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "project_id", testutil.ProjectId),
						resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "region", testutil.Region),
						resource.TestCheckResourceAttr("stackit_kms_keyring.keyring", "display_name", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["display_name"])),
						resource.TestCheckResourceAttrSet("stackit_kms_keyring.keyring", "keyring_id"),
						resource.TestCheckNoResourceAttr("stackit_kms_keyring.keyring", "description"),
					),

					// Instance
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["name"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "network.acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "network.acl.0", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["acl1"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "network.access_scope", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["access_scope"])),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "flavor_id"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.id"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "replicas", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["replicas"])),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.cpu"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.ram"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "storage.class", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["storage_class"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "storage.size", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["storage_size"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "version", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["server_version"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "retention_days", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["retention_days"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["backup_schedule"])),

					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "encryption.kek_key_id"),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "encryption.kek_keyring_id"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "encryption.kek_key_version", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["kek_key_version"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "encryption.service_account", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["service_account_email"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "region", testutil.Region),
					// User
					resource.TestCheckResourceAttrPair(
						"stackit_sqlserverflex_user.user", "project_id",
						"stackit_sqlserverflex_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_sqlserverflex_user.user", "instance_id",
						"stackit_sqlserverflex_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_user.user", "user_id"),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_user.user", "password"),
				),
			},
			// data source
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourcePrivatePreviewConfig,
				ConfigVariables: testConfigVarsMaxPrivatePreview,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "project_id", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "name", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["name"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sqlserverflex_instance.instance", "project_id",
						"stackit_sqlserverflex_instance.instance", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sqlserverflex_instance.instance", "instance_id",
						"stackit_sqlserverflex_instance.instance", "instance_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_sqlserverflex_user.user", "instance_id",
						"stackit_sqlserverflex_user.user", "instance_id",
					),

					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "acl.0", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["acl1"])),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_instance.instance", "flavor_id"),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_instance.instance", "flavor.id"),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_instance.instance", "flavor.description"),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_instance.instance", "flavor.cpu"),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_instance.instance", "flavor.ram"),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "replicas", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["replicas"])),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "retention_days", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["retention_days"])),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["backup_schedule"])),

					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_instance.instance", "encryption.kek_key_id"),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_instance.instance", "encryption.kek_keyring_id"),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "encryption.kek_key_version", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["kek_key_version"])),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_instance.instance", "encryption.service_account", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["service_account_email"])),

					// User data
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_user.user", "project_id", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_user.user", "user_id"),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_user.user", "username", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["username"])),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_user.user", "roles.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_sqlserverflex_user.user", "roles.0", testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["role"])),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_user.user", "host"),
					resource.TestCheckResourceAttrSet("data.stackit_sqlserverflex_user.user", "port"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVarsMaxPrivatePreview,
				ResourceName:    "stackit_sqlserverflex_instance.instance",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_sqlserverflex_instance.instance"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_sqlserverflex_instance.instance")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}

					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"backup_schedule",
					// Will be added during the import, because it's unknown which attribute defined the flavor
					"flavor.cpu",
					"flavor.description",
					"flavor.id",
					"flavor.ram",
				},
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					if len(s) != 1 {
						return fmt.Errorf("expected 1 state, got %d", len(s))
					}
					if s[0].Attributes["backup_schedule"] != testutil.ConvertConfigVariable(testConfigVarsMaxPrivatePreview["backup_schedule"]) {
						return fmt.Errorf("expected backup_schedule %s, got %s", testConfigVarsMaxPrivatePreview["backup_schedule"], s[0].Attributes["backup_schedule"])
					}
					return nil
				},
			},
			{
				ResourceName:    "stackit_sqlserverflex_user.user",
				ConfigVariables: testConfigVarsMaxPrivatePreview,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_sqlserverflex_user.user"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_sqlserverflex_user.user")
					}
					instanceId, ok := r.Primary.Attributes["instance_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute instance_id")
					}
					userId, ok := r.Primary.Attributes["user_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute user_id")
					}

					return fmt.Sprintf("%s,%s,%s,%s", testutil.ProjectId, testutil.Region, instanceId, userId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			// Update
			{
				Config:          testutil.NewConfigBuilder().BuildProviderConfig() + "\n" + resourcePrivatePreviewConfig,
				ConfigVariables: configVarsMaxPrivatePreviewUpdated(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("stackit_sqlserverflex_instance.instance", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("stackit_sqlserverflex_user.user", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance data
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "project_id", testutil.ConvertConfigVariable(configVarsMaxPrivatePreviewUpdated()["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "instance_id"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "name", testutil.ConvertConfigVariable(configVarsMaxPrivatePreviewUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "acl.#", "1"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "acl.0", testutil.ConvertConfigVariable(configVarsMaxPrivatePreviewUpdated()["acl1"])),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "flavor_id"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.id"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.description"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.cpu"),
					resource.TestCheckNoResourceAttr("stackit_sqlserverflex_instance.instance", "flavor.ram"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "replicas", testutil.ConvertConfigVariable(configVarsMaxPrivatePreviewUpdated()["replicas"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "storage.class", testutil.ConvertConfigVariable(configVarsMaxPrivatePreviewUpdated()["storage_class"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "storage.size", testutil.ConvertConfigVariable(configVarsMaxPrivatePreviewUpdated()["storage_size"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "version", testutil.ConvertConfigVariable(configVarsMaxPrivatePreviewUpdated()["server_version"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "retention_days", testutil.ConvertConfigVariable(configVarsMaxPrivatePreviewUpdated()["retention_days"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "backup_schedule", testutil.ConvertConfigVariable(configVarsMaxPrivatePreviewUpdated()["backup_schedule"])),

					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "encryption.kek_key_id"),
					resource.TestCheckResourceAttrSet("stackit_sqlserverflex_instance.instance", "encryption.kek_keyring_id"),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "encryption.kek_key_version", testutil.ConvertConfigVariable(configVarsMaxPrivatePreviewUpdated()["kek_key_version"])),
					resource.TestCheckResourceAttr("stackit_sqlserverflex_instance.instance", "encryption.service_account", testutil.ConvertConfigVariable(configVarsMaxPrivatePreviewUpdated()["service_account_email"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}
