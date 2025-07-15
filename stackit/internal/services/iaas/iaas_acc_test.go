package iaas_test

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	stackitSdkConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/wait"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
	waitAlpha "github.com/stackitcloud/stackit-sdk-go/services/iaasalpha/wait"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

var (
	//go:embed testdata/resource-security-group-min.tf
	resourceSecurityGroupMinConfig string

	//go:embed testdata/resource-security-group-max.tf
	resourceSecurityGroupMaxConfig string

	//go:embed testdata/resource-image-min.tf
	resourceImageMinConfig string

	//go:embed testdata/resource-image-max.tf
	resourceImageMaxConfig string

	//go:embed testdata/resource-key-pair-min.tf
	resourceKeyPairMinConfig string

	//go:embed testdata/resource-key-pair-max.tf
	resourceKeyPairMaxConfig string

	//go:embed testdata/resource-network-area-min.tf
	resourceNetworkAreaMinConfig string

	//go:embed testdata/resource-network-area-max.tf
	resourceNetworkAreaMaxConfig string

	//go:embed testdata/resource-network-v1-min.tf
	resourceNetworkV1MinConfig string

	//go:embed testdata/resource-network-v1-max.tf
	resourceNetworkV1MaxConfig string

	//go:embed testdata/resource-network-v2-min.tf
	resourceNetworkV2MinConfig string

	//go:embed testdata/resource-network-v2-max.tf
	resourceNetworkV2MaxConfig string

	//go:embed testdata/resource-network-interface-min.tf
	resourceNetworkInterfaceMinConfig string

	//go:embed testdata/resource-network-interface-max.tf
	resourceNetworkInterfaceMaxConfig string

	//go:embed testdata/resource-volume-min.tf
	resourceVolumeMinConfig string

	//go:embed testdata/resource-volume-max.tf
	resourceVolumeMaxConfig string

	//go:embed testdata/resource-affinity-group-min.tf
	resourceAffinityGroupMinConfig string

	//go:embed testdata/resource-server-min.tf
	resourceServerMinConfig string

	//go:embed testdata/resource-server-max.tf
	resourceServerMaxConfig string

	//go:embed testdata/resource-server-max-server-attachments.tf
	resourceServerMaxAttachmentConfig string
)

const (
	keypairPublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIDsPd27M449akqCtdFg2+AmRVJz6eWio0oMP9dVg7XZ"
	// TODO: create network area using terraform resource instead once it's out of experimental stage and GA
	testNetworkAreaId = "25bbf23a-8134-4439-9f5e-1641caf8354e"
)

var testConfigServerVarsMin = config.Variables{
	"project_id":   config.StringVariable(testutil.ProjectId),
	"name":         config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"machine_type": config.StringVariable("t1.1"),
	"image_id":     config.StringVariable("a2c127b2-b1b5-4aee-986f-41cd11b41279"),
}

var testConfigServerVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigServerVarsMin {
		updatedConfig[k] = v
	}
	updatedConfig["name"] = config.StringVariable(testutil.ProjectId)
	updatedConfig["machine_type"] = config.StringVariable("t1.2")
	return updatedConfig
}()

var testConfigServerVarsMax = config.Variables{
	"project_id":           config.StringVariable(testutil.ProjectId),
	"name":                 config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"name_not_updated":     config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"machine_type":         config.StringVariable("t1.1"),
	"image_id":             config.StringVariable("a2c127b2-b1b5-4aee-986f-41cd11b41279"),
	"availability_zone":    config.StringVariable("eu01-1"),
	"label":                config.StringVariable("label"),
	"user_data":            config.StringVariable("#!/bin/bash"),
	"policy":               config.StringVariable("soft-affinity"),
	"size":                 config.IntegerVariable(16),
	"service_account_mail": config.StringVariable(testutil.TestProjectServiceAccountEmail),
	"public_key":           config.StringVariable(keypairPublicKey),
	"desired_status":       config.StringVariable("active"),
}

var testConfigServerVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigServerVarsMax {
		updatedConfig[k] = v
	}
	updatedConfig["name"] = config.StringVariable(testutil.ProjectId)
	updatedConfig["machine_type"] = config.StringVariable("t1.2")
	updatedConfig["label"] = config.StringVariable("updated")
	updatedConfig["desired_status"] = config.StringVariable("inactive")
	return updatedConfig
}()

var testConfigServerVarsMaxUpdatedDesiredStatus = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigServerVarsMaxUpdated {
		updatedConfig[k] = v
	}
	updatedConfig["name"] = config.StringVariable(testutil.ProjectId)
	updatedConfig["machine_type"] = config.StringVariable("t1.2")
	updatedConfig["label"] = config.StringVariable("updated")
	updatedConfig["desired_status"] = config.StringVariable("deallocated")
	return updatedConfig
}()

var testConfigAffinityGroupVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"policy":     config.StringVariable("hard-affinity"),
}

var testConfigNetworkInterfaceVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
}

var testConfigNetworkInterfaceVarsMax = config.Variables{
	"project_id":      config.StringVariable(testutil.ProjectId),
	"name":            config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"allowed_address": config.StringVariable("10.2.10.0/24"),
	"ipv4":            config.StringVariable("10.2.10.20"),
	"ipv4_prefix":     config.StringVariable("10.2.10.0/24"),
	"security":        config.BoolVariable(true),
	"label":           config.StringVariable("label"),
}

var testConfigNetworkInterfaceVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigNetworkInterfaceVarsMax {
		updatedConfig[k] = v
	}
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["name"])))
	updatedConfig["ipv4"] = config.StringVariable("10.2.10.21")
	updatedConfig["security"] = config.BoolVariable(false)
	updatedConfig["label"] = config.StringVariable("updated")
	return updatedConfig
}()

var testConfigVolumeVarsMin = config.Variables{
	"project_id":        config.StringVariable(testutil.ProjectId),
	"availability_zone": config.StringVariable("eu01-1"),
	"size":              config.IntegerVariable(16),
}

var testConfigVolumeVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigVolumeVarsMin {
		updatedConfig[k] = v
	}
	updatedConfig["size"] = config.IntegerVariable(20)
	return updatedConfig
}()

var testConfigVolumeVarsMax = config.Variables{
	"project_id":        config.StringVariable(testutil.ProjectId),
	"availability_zone": config.StringVariable("eu01-1"),
	"name":              config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"size":              config.IntegerVariable(16),
	"description":       config.StringVariable("description"),
	"performance_class": config.StringVariable("storage_premium_perf0"),
	"label":             config.StringVariable("label"),
}

var testConfigVolumeVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigVolumeVarsMax {
		updatedConfig[k] = v
	}
	updatedConfig["size"] = config.IntegerVariable(20)
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["name"])))
	updatedConfig["description"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["description"])))
	updatedConfig["label"] = config.StringVariable("updated")
	return updatedConfig
}()

var testConfigNetworkV1VarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
}

var testConfigNetworkV1VarsMax = config.Variables{
	"project_id":         config.StringVariable(testutil.ProjectId),
	"name":               config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"ipv4_gateway":       config.StringVariable("10.2.2.1"),
	"ipv4_nameserver_0":  config.StringVariable("10.2.2.2"),
	"ipv4_nameserver_1":  config.StringVariable("10.2.2.3"),
	"ipv4_prefix":        config.StringVariable("10.2.2.0/24"),
	"ipv4_prefix_length": config.IntegerVariable(24),
	"routed":             config.BoolVariable(false),
	"label":              config.StringVariable("label"),
}

var testConfigNetworkV1VarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigNetworkV1VarsMax {
		updatedConfig[k] = v
	}
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"])))
	updatedConfig["ipv4_gateway"] = config.StringVariable("")
	updatedConfig["ipv4_nameserver_0"] = config.StringVariable("10.2.2.10")
	updatedConfig["label"] = config.StringVariable("updated")
	return updatedConfig
}()

var testConfigNetworkV2VarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
}

var testConfigNetworkV2VarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigNetworkV2VarsMin)
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"])))
	return updatedConfig
}()

var testConfigNetworkV2VarsMax = config.Variables{
	"project_id":         config.StringVariable(testutil.ProjectId),
	"name":               config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))),
	"ipv4_gateway":       config.StringVariable("10.2.2.1"),
	"ipv4_nameserver_0":  config.StringVariable("10.2.2.2"),
	"ipv4_nameserver_1":  config.StringVariable("10.2.2.3"),
	"ipv4_prefix":        config.StringVariable("10.2.2.0/24"),
	"ipv4_prefix_length": config.IntegerVariable(24),
	"routed":             config.BoolVariable(true),
	"label":              config.StringVariable("label"),
	"organization_id":    config.StringVariable(testutil.OrganizationId),
	"network_area_id":    config.StringVariable(testNetworkAreaId),
}

var testConfigNetworkV2VarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	maps.Copy(updatedConfig, testConfigNetworkV2VarsMax)
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"])))
	updatedConfig["ipv4_gateway"] = config.StringVariable("")
	updatedConfig["ipv4_nameserver_0"] = config.StringVariable("10.2.2.10")
	updatedConfig["label"] = config.StringVariable("updated")
	return updatedConfig
}()

var testConfigNetworkAreaVarsMin = config.Variables{
	"organization_id":       config.StringVariable(testutil.OrganizationId),
	"name":                  config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
	"transfer_network":      config.StringVariable("10.1.2.0/24"),
	"network_ranges_prefix": config.StringVariable("10.0.0.0/16"),
	"route_prefix":          config.StringVariable("1.1.1.0/24"),
	"route_next_hop":        config.StringVariable("1.1.1.1"),
}

var testConfigNetworkAreaVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigNetworkAreaVarsMin {
		updatedConfig[k] = v
	}
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"])))
	updatedConfig["network_ranges_prefix"] = config.StringVariable("10.0.0.0/18")
	return updatedConfig
}()

var testConfigNetworkAreaVarsMax = config.Variables{
	"organization_id":       config.StringVariable(testutil.OrganizationId),
	"name":                  config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
	"transfer_network":      config.StringVariable("10.1.2.0/24"),
	"network_ranges_prefix": config.StringVariable("10.0.0.0/16"),
	"default_nameservers":   config.StringVariable("1.1.1.1"),
	"default_prefix_length": config.IntegerVariable(24),
	"max_prefix_length":     config.IntegerVariable(24),
	"min_prefix_length":     config.IntegerVariable(16),
	"route_prefix":          config.StringVariable("1.1.1.0/24"),
	"route_next_hop":        config.StringVariable("1.1.1.1"),
	"label":                 config.StringVariable("label"),
}

var testConfigNetworkAreaVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigNetworkAreaVarsMax {
		updatedConfig[k] = v
	}
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"])))
	updatedConfig["network_ranges_prefix"] = config.StringVariable("10.0.0.0/18")
	updatedConfig["default_nameservers"] = config.StringVariable("1.1.1.2")
	updatedConfig["default_prefix_length"] = config.IntegerVariable(25)
	updatedConfig["max_prefix_length"] = config.IntegerVariable(25)
	updatedConfig["min_prefix_length"] = config.IntegerVariable(20)
	updatedConfig["label"] = config.StringVariable("updated")
	return updatedConfig
}()

var testConfigSecurityGroupsVarsMin = config.Variables{
	"project_id": config.StringVariable(testutil.ProjectId),
	"name":       config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
	"direction":  config.StringVariable("ingress"),
}

func testConfigSecurityGroupsVarsMinUpdated() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigSecurityGroupsVarsMin {
		updatedConfig[k] = v
	}
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"])))
	return updatedConfig
}

var testConfigSecurityGroupsVarsMax = config.Variables{
	"project_id":       config.StringVariable(testutil.ProjectId),
	"name":             config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
	"description":      config.StringVariable("description"),
	"description_rule": config.StringVariable("description"),
	"label":            config.StringVariable("label"),
	"stateful":         config.BoolVariable(false),
	"direction":        config.StringVariable("ingress"),
	"ether_type":       config.StringVariable("IPv4"),
	"ip_range":         config.StringVariable("192.168.2.0/24"),
	"port":             config.StringVariable("443"),
	"protocol":         config.StringVariable("tcp"),
	"icmp_code":        config.IntegerVariable(0),
	"icmp_type":        config.IntegerVariable(8),
	"name_remote":      config.StringVariable(fmt.Sprintf("tf-acc-remote-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
}

func testConfigSecurityGroupsVarsMaxUpdated() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigSecurityGroupsVarsMax {
		updatedConfig[k] = v
	}
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"])))
	updatedConfig["name_remote"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["name_remote"])))
	updatedConfig["description"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["description"])))
	updatedConfig["label"] = config.StringVariable("updated")

	return updatedConfig
}

var testConfigImageVarsMin = func() config.Variables {
	localFilePath := testutil.TestImageLocalFilePath
	if localFilePath == "default" {
		localFileForIaasImage = testutil.CreateDefaultLocalFile()
		filePath, err := filepath.Abs(localFileForIaasImage.Name())
		if err != nil {
			fmt.Println("Absolute path for localFileForIaasImage could not be retrieved.")
		}
		localFilePath = filePath
	}
	return config.Variables{
		"project_id":      config.StringVariable(testutil.ProjectId),
		"name":            config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
		"disk_format":     config.StringVariable("qcow2"),
		"local_file_path": config.StringVariable(localFilePath),
	}
}()

var testConfigImageVarsMinUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigImageVarsMin {
		updatedConfig[k] = v
	}
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"])))
	return updatedConfig
}()

var testConfigImageVarsMax = func() config.Variables {
	localFilePath := testutil.TestImageLocalFilePath
	if localFilePath == "default" {
		localFileForIaasImage = testutil.CreateDefaultLocalFile()
		filePath, err := filepath.Abs(localFileForIaasImage.Name())
		if err != nil {
			fmt.Println("Absolute path for localFileForIaasImage could not be retrieved.")
		}
		localFilePath = filePath
	}
	return config.Variables{
		"project_id":               config.StringVariable(testutil.ProjectId),
		"name":                     config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
		"disk_format":              config.StringVariable("qcow2"),
		"local_file_path":          config.StringVariable(localFilePath),
		"min_disk_size":            config.IntegerVariable(20),
		"min_ram":                  config.IntegerVariable(2048),
		"label":                    config.StringVariable("label"),
		"boot_menu":                config.BoolVariable(false),
		"cdrom_bus":                config.StringVariable("scsi"),
		"disk_bus":                 config.StringVariable("scsi"),
		"nic_model":                config.StringVariable("e1000"),
		"operating_system":         config.StringVariable("linux"),
		"operating_system_distro":  config.StringVariable("ubuntu"),
		"operating_system_version": config.StringVariable("16.04"),
		"rescue_bus":               config.StringVariable("sata"),
		"rescue_device":            config.StringVariable("cdrom"),
		"secure_boot":              config.BoolVariable(true),
		"uefi":                     config.BoolVariable(true),
		"video_model":              config.StringVariable("vga"),
		"virtio_scsi":              config.BoolVariable(true),
	}
}()

var testConfigImageVarsMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigImageVarsMax {
		updatedConfig[k] = v
	}
	updatedConfig["name"] = config.StringVariable(fmt.Sprintf("%s-updated", testutil.ConvertConfigVariable(updatedConfig["name"])))
	updatedConfig["min_disk_size"] = config.IntegerVariable(25)
	updatedConfig["min_ram"] = config.IntegerVariable(4096)
	updatedConfig["label"] = config.StringVariable("updated")
	updatedConfig["boot_menu"] = config.BoolVariable(false)
	updatedConfig["cdrom_bus"] = config.StringVariable("usb")
	updatedConfig["disk_bus"] = config.StringVariable("usb")
	updatedConfig["nic_model"] = config.StringVariable("virtio")
	updatedConfig["operating_system"] = config.StringVariable("windows")
	updatedConfig["operating_system_distro"] = config.StringVariable("debian")
	updatedConfig["operating_system_version"] = config.StringVariable("18.04")
	updatedConfig["rescue_bus"] = config.StringVariable("usb")
	updatedConfig["rescue_device"] = config.StringVariable("disk")
	updatedConfig["secure_boot"] = config.BoolVariable(false)
	updatedConfig["uefi"] = config.BoolVariable(false)
	updatedConfig["video_model"] = config.StringVariable("virtio")
	updatedConfig["virtio_scsi"] = config.BoolVariable(false)
	return updatedConfig
}()

var testConfigKeyPairMin = config.Variables{
	"name":       config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
	"public_key": config.StringVariable(keypairPublicKey),
}

var testConfigKeyPairMax = config.Variables{
	"name":       config.StringVariable(fmt.Sprintf("tf-acc-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))),
	"public_key": config.StringVariable(keypairPublicKey),
	"label":      config.StringVariable("label"),
}

var testConfigKeyPairMaxUpdated = func() config.Variables {
	updatedConfig := config.Variables{}
	for k, v := range testConfigKeyPairMax {
		updatedConfig[k] = v
	}
	updatedConfig["label"] = config.StringVariable("updated")
	return updatedConfig
}()

// if no local file is provided the test should create a default file and work with this instead of failing
var localFileForIaasImage os.File

func TestAccNetworkV1Min(t *testing.T) {
	t.Logf("TestAccNetworkV1Min name: %s", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMin["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigNetworkV1VarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceNetworkV1MinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_network.network", "name", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMin["name"])),
					resource.TestCheckResourceAttrSet("stackit_network.network", "ipv4_prefixes.#"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "public_ip"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigNetworkV1VarsMin,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_network" "network" {
						project_id  = stackit_network.network.project_id
						network_id  = stackit_network.network.network_id
					}
					`,
					testutil.IaaSProviderConfig(), resourceNetworkV1MinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("data.stackit_network.network", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMin["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_network.network", "name", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMin["name"])),
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "ipv4_prefixes.#"),
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "public_ip"),
				),
			},

			// Import
			{
				ConfigVariables: testConfigNetworkV1VarsMin,
				ResourceName:    "stackit_network.network",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network.network"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network.network")
					}
					networkId, ok := r.Primary.Attributes["network_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, networkId), nil
				},
				ImportState: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_network.network", "name", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMin["name"])),
					resource.TestCheckResourceAttrSet("stackit_network.network", "ipv4_prefixes.#"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "public_ip"),
				),
			},
			// In this minimal setup, no update can be performed
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccNetworkV1Max(t *testing.T) {
	t.Logf("TestAccNetworkV1Max name: %s", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigNetworkV1VarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceNetworkV1MaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "name", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_gateway", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_gateway"])),
					resource.TestCheckNoResourceAttr("stackit_network.network_prefix", "no_ipv4_gateway"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_nameservers.#", "2"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_nameservers.0", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_nameserver_0"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_nameservers.1", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_nameserver_1"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_prefix_length"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefix", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_prefix"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefixes.#", "1"),
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "routed", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["routed"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["label"])),
					resource.TestCheckNoResourceAttr("stackit_network.network_prefix", "public_ip"),

					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "name", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["name"])),
					resource.TestCheckNoResourceAttr("stackit_network.network_prefix_length", "ipv4_gateway"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "no_ipv4_gateway", "true"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_nameservers.#", "2"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_nameservers.0", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_nameserver_0"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_nameservers.1", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_nameserver_1"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_prefix_length"])),
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "ipv4_prefix"),
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "routed", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["routed"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["label"])),
					resource.TestCheckNoResourceAttr("stackit_network.network_prefix_length", "public_ip"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigNetworkV1VarsMax,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_network" "network_prefix" {
						project_id  = stackit_network.network_prefix.project_id
						network_id  = stackit_network.network_prefix.network_id
					}

					data "stackit_network" "network_prefix_length" {
						project_id  = stackit_network.network_prefix_length.project_id
						network_id  = stackit_network.network_prefix_length.network_id
					}
					`,
					testutil.IaaSProviderConfig(), resourceNetworkV1MaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.stackit_network.network_prefix", "network_id"),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "name", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "ipv4_gateway", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_gateway"])),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "ipv4_nameservers.#", "2"),
					resource.TestCheckTypeSetElemAttr("data.stackit_network.network_prefix", "ipv4_nameservers.*", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_nameserver_0"])),
					resource.TestCheckTypeSetElemAttr("data.stackit_network.network_prefix", "ipv4_nameservers.*", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_nameserver_1"])),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "ipv4_prefix", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_prefix"])),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "ipv4_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_prefix_length"])),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "ipv4_prefixes.#", "1"),
					resource.TestCheckResourceAttrSet("data.stackit_network.network_prefix", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "routed", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["routed"])),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["label"])),

					resource.TestCheckResourceAttrSet("data.stackit_network.network_prefix_length", "network_id"),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix_length", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix_length", "name", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["name"])),
					resource.TestCheckNoResourceAttr("data.stackit_network.network_prefix_length", "ipv4_gateway"),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix_length", "ipv4_nameservers.#", "2"),
					resource.TestCheckTypeSetElemAttr("data.stackit_network.network_prefix_length", "ipv4_nameservers.*", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_nameserver_0"])),
					resource.TestCheckTypeSetElemAttr("data.stackit_network.network_prefix_length", "ipv4_nameservers.*", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_nameserver_1"])),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix_length", "ipv4_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_prefix_length"])),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix_length", "ipv4_prefixes.#", "1"),
					resource.TestCheckResourceAttrSet("data.stackit_network.network_prefix_length", "ipv4_prefix"),
					resource.TestCheckResourceAttrSet("data.stackit_network.network_prefix_length", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix_length", "routed", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["routed"])),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix_length", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["label"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigNetworkV1VarsMax,
				ResourceName:    "stackit_network.network_prefix",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network.network_prefix"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network.network_prefix")
					}
					networkId, ok := r.Primary.Attributes["network_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, networkId), nil
				},
				ImportState: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "name", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_gateway", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_gateway"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_nameservers.#", "2"),
					// nameservers may be returned in a randomized order, so we have to check them with a helper function
					resource.TestCheckTypeSetElemAttr("stackit_network.network_prefix", "nameservers.*", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_nameserver_0"])),
					resource.TestCheckTypeSetElemAttr("stackit_network.network_prefix", "nameservers.*", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_nameserver_1"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefix", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_prefix"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_prefix_length"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefixes.#", "1"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefixes.0", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_prefix"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "routed", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["routed"])),
				),
			},
			{
				ConfigVariables: testConfigNetworkV1VarsMax,
				ResourceName:    "stackit_network.network_prefix_length",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network.network_prefix_length"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network.network_prefix_length")
					}
					networkId, ok := r.Primary.Attributes["network_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, networkId), nil
				},
				ImportState: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "name", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["project_id"])),
					resource.TestCheckNoResourceAttr("stackit_network.network_prefix_length", "ipv4_gateway"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_nameservers.#", "2"),
					// nameservers may be returned in a randomized order, so we have to check them with a helper function
					resource.TestCheckTypeSetElemAttr("stackit_network.network_prefix_length", "nameservers.*", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_nameserver_0"])),
					resource.TestCheckTypeSetElemAttr("stackit_network.network_prefix_length", "nameservers.*", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_nameserver_1"])),
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "ipv4_prefix"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["ipv4_prefix_length"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_prefixes.#", "1"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "routed", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMax["routed"])),
				),
			},
			// Update
			{
				ConfigVariables: testConfigNetworkV1VarsMaxUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceNetworkV1MaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "name", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMaxUpdated["name"])),
					resource.TestCheckNoResourceAttr("stackit_network.network_prefix", "ipv4_gateway"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "no_ipv4_gateway", "true"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_nameservers.#", "2"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_nameservers.0", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMaxUpdated["ipv4_nameserver_0"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_nameservers.1", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMaxUpdated["ipv4_nameserver_1"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefix", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMaxUpdated["ipv4_prefix"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMaxUpdated["ipv4_prefix_length"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefixes.#", "1"),
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "routed", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMaxUpdated["routed"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMaxUpdated["label"])),
					resource.TestCheckNoResourceAttr("stackit_network.network_prefix", "public_ip"),

					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "name", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMaxUpdated["name"])),
					resource.TestCheckNoResourceAttr("stackit_network.network_prefix_length", "ipv4_gateway"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "no_ipv4_gateway", "true"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_nameservers.#", "2"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_nameservers.0", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMaxUpdated["ipv4_nameserver_0"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_nameservers.1", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMaxUpdated["ipv4_nameserver_1"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMaxUpdated["ipv4_prefix_length"])),
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "ipv4_prefix"),
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "routed", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMaxUpdated["routed"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkV1VarsMaxUpdated["label"])),
					resource.TestCheckNoResourceAttr("stackit_network.network_prefix_length", "public_ip"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccNetworkV2Min(t *testing.T) {
	t.Logf("TestAccNetworkV2Min name: %s", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMin["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNetworkV2Destroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigNetworkV2VarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfigWithExperiments(), resourceNetworkV2MinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_network.network", "name", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMin["name"])),
					resource.TestCheckResourceAttrSet("stackit_network.network", "ipv4_prefixes.#"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "public_ip"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "region"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "routing_table_id"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigNetworkV2VarsMin,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_network" "network" {
						project_id  = stackit_network.network.project_id
						network_id  = stackit_network.network.network_id
					}
					`,
					testutil.IaaSProviderConfigWithExperiments(), resourceNetworkV2MinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("data.stackit_network.network", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMin["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_network.network", "name", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMin["name"])),
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "ipv4_prefixes.#"),
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "public_ip"),
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "region"),
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "routing_table_id"),
				),
			},

			// Import
			{
				ConfigVariables: testConfigNetworkV2VarsMin,
				ResourceName:    "stackit_network.network",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network.network"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network.network")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					networkId, ok := r.Primary.Attributes["network_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, region, networkId), nil
				},
				ImportState: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_network.network", "name", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMin["name"])),
					resource.TestCheckResourceAttrSet("stackit_network.network", "ipv4_prefixes.#"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "public_ip"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "region"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "routing_table_id"),
				),
			},
			// Update
			{
				ConfigVariables: testConfigNetworkV2VarsMinUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfigWithExperiments(), resourceNetworkV2MinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMinUpdated["project_id"])),
					resource.TestCheckResourceAttr("stackit_network.network", "name", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMinUpdated["name"])),
					resource.TestCheckResourceAttrSet("stackit_network.network", "ipv4_prefixes.#"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "public_ip"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "region"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "routing_table_id"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccNetworkV2Max(t *testing.T) {
	t.Logf("TestAccNetworkV2Max name: %s", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNetworkV2Destroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigNetworkV2VarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfigWithExperiments(), resourceNetworkV2MaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// TODO: enable test cases for prefix option, when the API works again
					// Network with prefix
					// resource.TestCheckResourceAttrSet("stackit_network.network_prefix", "network_id"),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["project_id"])),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "name", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["name"])),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_gateway", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_gateway"])),
					// resource.TestCheckNoResourceAttr("stackit_network.network_prefix", "no_ipv4_gateway"),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_nameservers.#", "2"),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_nameservers.0", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_nameserver_0"])),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_nameservers.1", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_nameserver_1"])),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_prefix_length"])),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefix", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_prefix"])),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefixes.#", "1"),
					// resource.TestCheckResourceAttrSet("stackit_network.network_prefix", "ipv6_prefixes.#"),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "routed", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["routed"])),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["label"])),
					// resource.TestCheckNoResourceAttr("stackit_network.network_prefix", "public_ip"),

					// Network with prefix_length
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "name", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["name"])),
					// resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "ipv4_gateway"),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "no_ipv4_gateway", "true"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_nameservers.#", "2"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_nameservers.0", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_nameserver_0"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_nameservers.1", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_nameserver_1"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_prefix_length"])),
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "ipv4_prefix"),
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "routed", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["routed"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["label"])),
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "public_ip"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "region", testutil.Region),

					resource.TestCheckResourceAttrPair(
						"stackit_network.network_prefix_length", "routing_table_id",
						"stackit_routing_table.routing_table", "routing_table_id",
					),

					// Routing table
					resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "organization_id", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["organization_id"])),
					resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "network_area_id", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["network_area_id"])),
					resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "routing_table_id"),
					resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "name", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "labels.%", "0"),
					resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "region", testutil.Region),
					resource.TestCheckNoResourceAttr("stackit_routing_table.routing_table", "description"),
					resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "system_routes", "true"),
					resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "updated_at"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigNetworkV2VarsMax,
				Config: fmt.Sprintf(`
					%s
					%s

					//data "stackit_network" "network_prefix" {
					//	project_id  = stackit_network.network_prefix.project_id
					//	network_id  = stackit_network.network_prefix.network_id
					//}

					data "stackit_network" "network_prefix_length" {
						project_id  = stackit_network.network_prefix_length.project_id
						network_id  = stackit_network.network_prefix_length.network_id
					}

					data "stackit_routing_table" "routing_table" {
						organization_id  = stackit_routing_table.routing_table.organization_id
						network_area_id  = stackit_routing_table.routing_table.network_area_id
						routing_table_id  = stackit_routing_table.routing_table.routing_table_id
					}
					`,
					testutil.IaaSProviderConfigWithExperiments(), resourceNetworkV2MaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// TODO: enable test cases for prefix option, when the API works again
					// Network with prefix
					// resource.TestCheckResourceAttrSet("data.stackit_network.network_prefix", "network_id"),
					// resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["project_id"])),
					// resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "name", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["name"])),
					// resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "ipv4_gateway", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_gateway"])),
					// resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "ipv4_nameservers.#", "2"),
					// resource.TestCheckTypeSetElemAttr("data.stackit_network.network_prefix", "ipv4_nameservers.*", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_nameserver_0"])),
					// resource.TestCheckTypeSetElemAttr("data.stackit_network.network_prefix", "ipv4_nameservers.*", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_nameserver_1"])),
					// resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "ipv4_prefix", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_prefix"])),
					// resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "ipv4_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_prefix_length"])),
					// resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "ipv4_prefixes.#", "1"),
					// resource.TestCheckResourceAttrSet("data.stackit_network.network_prefix", "ipv6_prefixes.#"),
					// resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "routed", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["routed"])),
					// resource.TestCheckResourceAttr("data.stackit_network.network_prefix", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["label"])),

					// Network with prefix_length
					resource.TestCheckResourceAttrSet("data.stackit_network.network_prefix_length", "network_id"),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix_length", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix_length", "name", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["name"])),
					// resource.TestCheckNoResourceAttr("data.stackit_network.network_prefix_length", "ipv4_gateway"),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix_length", "ipv4_nameservers.#", "2"),
					resource.TestCheckTypeSetElemAttr("data.stackit_network.network_prefix_length", "ipv4_nameservers.*", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_nameserver_0"])),
					resource.TestCheckTypeSetElemAttr("data.stackit_network.network_prefix_length", "ipv4_nameservers.*", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_nameserver_1"])),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix_length", "ipv4_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_prefix_length"])),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix_length", "ipv4_prefixes.#", "1"),
					resource.TestCheckResourceAttrSet("data.stackit_network.network_prefix_length", "ipv4_prefix"),
					resource.TestCheckResourceAttrSet("data.stackit_network.network_prefix_length", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix_length", "routed", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["routed"])),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix_length", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["label"])),
					resource.TestCheckResourceAttr("data.stackit_network.network_prefix_length", "region", testutil.Region),

					resource.TestCheckResourceAttrPair(
						"data.stackit_network.network_prefix_length", "routing_table_id",
						"data.stackit_routing_table.routing_table", "routing_table_id",
					),

					// Routing table
					resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "organization_id", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["organization_id"])),
					resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "network_area_id", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["network_area_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_routing_table.routing_table", "routing_table_id"),
					resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "name", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "labels.%", "0"),
					resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "region", testutil.Region),
					resource.TestCheckNoResourceAttr("data.stackit_routing_table.routing_table", "description"),
					resource.TestCheckResourceAttr("data.stackit_routing_table.routing_table", "system_routes", "true"),
					resource.TestCheckResourceAttrSet("data.stackit_routing_table.routing_table", "created_at"),
					resource.TestCheckResourceAttrSet("data.stackit_routing_table.routing_table", "updated_at"),
				),
			},
			// Import
			// TODO: enable test cases for prefix option, when the API works again
			//{
			//	ConfigVariables: testConfigNetworkV2VarsMax,
			//	ResourceName:    "stackit_network.network_prefix",
			//	ImportStateIdFunc: func(s *terraform.State) (string, error) {
			//		r, ok := s.RootModule().Resources["stackit_network.network_prefix"]
			//		if !ok {
			//			return "", fmt.Errorf("couldn't find resource stackit_network.network_prefix")
			//		}
			//		networkId, ok := r.Primary.Attributes["network_id"]
			//		if !ok {
			//			return "", fmt.Errorf("couldn't find attribute network_id")
			//		}
			//		return fmt.Sprintf("%s,%s", testutil.ProjectId, networkId), nil
			//	},
			//	ImportState: true,
			//	Check: resource.ComposeAggregateTestCheckFunc(
			//		resource.TestCheckResourceAttrSet("stackit_network.network_prefix", "network_id"),
			//		resource.TestCheckResourceAttr("stackit_network.network_prefix", "name", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["project_id"])),
			//		resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_gateway", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_gateway"])),
			//		resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_nameservers.#", "2"),
			//		// nameservers may be returned in a randomized order, so we have to check them with a helper function
			//		resource.TestCheckTypeSetElemAttr("stackit_network.network_prefix", "nameservers.*", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_nameserver_0"])),
			//		resource.TestCheckTypeSetElemAttr("stackit_network.network_prefix", "nameservers.*", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_nameserver_1"])),
			//		resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefix", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_prefix"])),
			//		resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_prefix_length"])),
			//		resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefixes.#", "1"),
			//		resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefixes.0", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_prefix"])),
			//		resource.TestCheckResourceAttr("stackit_network.network_prefix", "routed", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["routed"])),
			//	),
			// },
			{
				ConfigVariables: testConfigNetworkV2VarsMax,
				ResourceName:    "stackit_network.network_prefix_length",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network.network_prefix_length"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network.network_prefix_length")
					}
					region, ok := r.Primary.Attributes["region"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute region")
					}
					networkId, ok := r.Primary.Attributes["network_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, region, networkId), nil
				},
				ImportState: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "name", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["project_id"])),
					// resource.TestCheckNoResourceAttr("stackit_network.network_prefix_length", "ipv4_gateway"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_nameservers.#", "2"),
					// nameservers may be returned in a randomized order, so we have to check them with a helper function
					resource.TestCheckTypeSetElemAttr("stackit_network.network_prefix_length", "nameservers.*", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_nameserver_0"])),
					resource.TestCheckTypeSetElemAttr("stackit_network.network_prefix_length", "nameservers.*", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_nameserver_1"])),
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "ipv4_prefix"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["ipv4_prefix_length"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_prefixes.#", "1"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "routed", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMax["routed"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "region", testutil.Region),
				),
			},
			// Update
			{
				ConfigVariables: testConfigNetworkV2VarsMaxUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfigWithExperiments(), resourceNetworkV2MaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// TODO: enable test cases for prefix option, when the API works again
					// resource.TestCheckResourceAttrSet("stackit_network.network_prefix", "network_id"),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["project_id"])),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "name", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["name"])),
					// resource.TestCheckNoResourceAttr("stackit_network.network_prefix", "ipv4_gateway"),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "no_ipv4_gateway", "true"),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_nameservers.#", "2"),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_nameservers.0", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["ipv4_nameserver_0"])),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_nameservers.1", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["ipv4_nameserver_1"])),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefix", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["ipv4_prefix"])),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["ipv4_prefix_length"])),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "ipv4_prefixes.#", "1"),
					// resource.TestCheckResourceAttrSet("stackit_network.network_prefix", "ipv6_prefixes.#"),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "routed", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["routed"])),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["label"])),
					// resource.TestCheckNoResourceAttr("stackit_network.network_prefix", "public_ip"),

					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "project_id", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "name", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["name"])),
					// resource.TestCheckNoResourceAttr("stackit_network.network_prefix_length", "ipv4_gateway"),
					// resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "no_ipv4_gateway", "true"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_nameservers.#", "2"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_nameservers.0", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["ipv4_nameserver_0"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_nameservers.1", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["ipv4_nameserver_1"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "ipv4_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["ipv4_prefix_length"])),
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "ipv4_prefix"),
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "routed", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["routed"])),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["label"])),
					resource.TestCheckResourceAttrSet("stackit_network.network_prefix_length", "public_ip"),
					resource.TestCheckResourceAttr("stackit_network.network_prefix_length", "region", testutil.Region),

					resource.TestCheckResourceAttrPair(
						"stackit_network.network_prefix_length", "routing_table_id",
						"stackit_routing_table.routing_table", "routing_table_id",
					),

					// Routing table
					resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "organization_id", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["organization_id"])),
					resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "network_area_id", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["network_area_id"])),
					resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "routing_table_id"),
					resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "name", testutil.ConvertConfigVariable(testConfigNetworkV2VarsMaxUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "labels.%", "0"),
					resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "region", testutil.Region),
					resource.TestCheckNoResourceAttr("stackit_routing_table.routing_table", "description"),
					resource.TestCheckResourceAttr("stackit_routing_table.routing_table", "system_routes", "true"),
					resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_routing_table.routing_table", "updated_at"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccNetworkAreaMin(t *testing.T) {
	t.Logf("TestAccNetworkAreaMin name: %s", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMin["name"]))
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigNetworkAreaVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceNetworkAreaMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Network Area
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "organization_id", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMin["organization_id"])),
					resource.TestCheckResourceAttrSet("stackit_network_area.network_area", "network_area_id"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "name", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "network_ranges.#", "1"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "network_ranges.0.prefix", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMin["network_ranges_prefix"])),
					resource.TestCheckResourceAttrSet("stackit_network_area.network_area", "network_ranges.0.network_range_id"),

					// Network Area Route
					resource.TestCheckResourceAttrPair(
						"stackit_network_area_route.network_area_route", "organization_id",
						"stackit_network_area.network_area", "organization_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_area_route.network_area_route", "network_area_id",
						"stackit_network_area.network_area", "network_area_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_area_route.network_area_route", "network_area_route_id"),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "prefix", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMin["route_prefix"])),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "next_hop", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMin["route_next_hop"])),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigNetworkAreaVarsMin,
				Config: fmt.Sprintf(`
					%s
					%s
						
					data "stackit_network_area" "network_area" {
						organization_id  = stackit_network_area.network_area.organization_id
						network_area_id  = stackit_network_area.network_area.network_area_id
					}
					
					data "stackit_network_area_route" "network_area_route" {
						organization_id  	  = stackit_network_area.network_area.organization_id
						network_area_id  	  = stackit_network_area.network_area.network_area_id
						network_area_route_id = stackit_network_area_route.network_area_route.network_area_route_id
					}
					`,
					testutil.IaaSProviderConfig(), resourceNetworkAreaMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Network Area
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "organization_id", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMin["organization_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_network_area.network_area", "network_area_id"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_network_area.network_area", "network_area_id",
						"stackit_network_area.network_area", "network_area_id",
					),
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "name", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMin["name"])),
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "network_ranges.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "network_ranges.0.prefix", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMin["network_ranges_prefix"])),
					resource.TestCheckResourceAttrSet("data.stackit_network_area.network_area", "network_ranges.0.network_range_id"),

					// Network Area Route
					resource.TestCheckResourceAttrPair(
						"data.stackit_network_area_route.network_area_route", "organization_id",
						"data.stackit_network_area.network_area", "organization_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_network_area_route.network_area_route", "network_area_id",
						"data.stackit_network_area.network_area", "network_area_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_network_area_route.network_area_route", "network_area_route_id",
						"stackit_network_area_route.network_area_route", "network_area_route_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_network_area_route.network_area_route", "network_area_route_id"),
					resource.TestCheckResourceAttr("data.stackit_network_area_route.network_area_route", "prefix", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMin["route_prefix"])),
					resource.TestCheckResourceAttr("data.stackit_network_area_route.network_area_route", "next_hop", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMin["route_next_hop"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigNetworkAreaVarsMinUpdated,
				ResourceName:    "stackit_network_area.network_area",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network_area.network_area"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network_area.network_area")
					}
					networkAreaId, ok := r.Primary.Attributes["network_area_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_area_id")
					}
					return fmt.Sprintf("%s,%s", testutil.OrganizationId, networkAreaId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigNetworkAreaVarsMinUpdated,
				ResourceName:    "stackit_network_area_route.network_area_route",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network_area_route.network_area_route"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network_area_route.network_area_route")
					}
					networkAreaId, ok := r.Primary.Attributes["network_area_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_area_id")
					}
					networkAreaRouteId, ok := r.Primary.Attributes["network_area_route_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_area_route_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.OrganizationId, networkAreaId, networkAreaRouteId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigNetworkAreaVarsMinUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceNetworkAreaMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Network Area
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "organization_id", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMinUpdated["organization_id"])),
					resource.TestCheckResourceAttrSet("stackit_network_area.network_area", "network_area_id"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "name", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMinUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "network_ranges.#", "1"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "network_ranges.0.prefix", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMinUpdated["network_ranges_prefix"])),
					resource.TestCheckResourceAttrSet("stackit_network_area.network_area", "network_ranges.0.network_range_id"),

					// Network Area Route
					resource.TestCheckResourceAttrPair(
						"stackit_network_area_route.network_area_route", "organization_id",
						"stackit_network_area.network_area", "organization_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_area_route.network_area_route", "network_area_id",
						"stackit_network_area.network_area", "network_area_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_area_route.network_area_route", "network_area_route_id"),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "prefix", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMinUpdated["route_prefix"])),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "next_hop", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMinUpdated["route_next_hop"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccNetworkAreaMax(t *testing.T) {
	t.Logf("TestAccNetworkAreaMax name: %s", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["name"]))
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigNetworkAreaVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceNetworkAreaMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Network Area
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "organization_id", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["organization_id"])),
					resource.TestCheckResourceAttrSet("stackit_network_area.network_area", "network_area_id"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "name", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "network_ranges.#", "1"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "network_ranges.0.prefix", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["network_ranges_prefix"])),
					resource.TestCheckResourceAttrSet("stackit_network_area.network_area", "network_ranges.0.network_range_id"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["label"])),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "default_nameservers.#", "1"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "default_nameservers.0", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["default_nameservers"])),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "default_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["default_prefix_length"])),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "max_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["max_prefix_length"])),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "min_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["min_prefix_length"])),

					// Network Area Route
					resource.TestCheckResourceAttrPair(
						"stackit_network_area_route.network_area_route", "organization_id",
						"stackit_network_area.network_area", "organization_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_area_route.network_area_route", "network_area_id",
						"stackit_network_area.network_area", "network_area_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_area_route.network_area_route", "network_area_route_id"),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "prefix", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["route_prefix"])),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "next_hop", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["route_next_hop"])),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["label"])),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigNetworkAreaVarsMax,
				Config: fmt.Sprintf(`
					%s
					%s
						
					data "stackit_network_area" "network_area" {
						organization_id  = stackit_network_area.network_area.organization_id
						network_area_id  = stackit_network_area.network_area.network_area_id
					}
					
					data "stackit_network_area_route" "network_area_route" {
						organization_id  	  = stackit_network_area.network_area.organization_id
						network_area_id  	  = stackit_network_area.network_area.network_area_id
						network_area_route_id = stackit_network_area_route.network_area_route.network_area_route_id
					}
					`,
					testutil.IaaSProviderConfig(), resourceNetworkAreaMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Network Area
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "organization_id", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["organization_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_network_area.network_area", "network_area_id"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_network_area.network_area", "network_area_id",
						"stackit_network_area.network_area", "network_area_id",
					),
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "name", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "network_ranges.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "network_ranges.0.prefix", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["network_ranges_prefix"])),
					resource.TestCheckResourceAttrSet("data.stackit_network_area.network_area", "network_ranges.0.network_range_id"),
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["label"])),
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "default_nameservers.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "default_nameservers.0", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["default_nameservers"])),
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "default_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["default_prefix_length"])),
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "max_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["max_prefix_length"])),
					resource.TestCheckResourceAttr("data.stackit_network_area.network_area", "min_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["min_prefix_length"])),

					// Network Area Route
					resource.TestCheckResourceAttrPair(
						"data.stackit_network_area_route.network_area_route", "organization_id",
						"data.stackit_network_area.network_area", "organization_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_network_area_route.network_area_route", "network_area_id",
						"data.stackit_network_area.network_area", "network_area_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_network_area_route.network_area_route", "network_area_route_id",
						"stackit_network_area_route.network_area_route", "network_area_route_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_network_area_route.network_area_route", "network_area_route_id"),
					resource.TestCheckResourceAttr("data.stackit_network_area_route.network_area_route", "prefix", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["route_prefix"])),
					resource.TestCheckResourceAttr("data.stackit_network_area_route.network_area_route", "next_hop", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMax["route_next_hop"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigNetworkAreaVarsMaxUpdated,
				ResourceName:    "stackit_network_area.network_area",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network_area.network_area"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network_area.network_area")
					}
					networkAreaId, ok := r.Primary.Attributes["network_area_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_area_id")
					}
					return fmt.Sprintf("%s,%s", testutil.OrganizationId, networkAreaId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigNetworkAreaVarsMaxUpdated,
				ResourceName:    "stackit_network_area_route.network_area_route",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network_area_route.network_area_route"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network_area_route.network_area_route")
					}
					networkAreaId, ok := r.Primary.Attributes["network_area_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_area_id")
					}
					networkAreaRouteId, ok := r.Primary.Attributes["network_area_route_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_area_route_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.OrganizationId, networkAreaId, networkAreaRouteId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigNetworkAreaVarsMaxUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceNetworkAreaMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Network Area
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "organization_id", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMaxUpdated["organization_id"])),
					resource.TestCheckResourceAttrSet("stackit_network_area.network_area", "network_area_id"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "name", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMaxUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "network_ranges.#", "1"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "network_ranges.0.prefix", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMaxUpdated["network_ranges_prefix"])),
					resource.TestCheckResourceAttrSet("stackit_network_area.network_area", "network_ranges.0.network_range_id"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMaxUpdated["label"])),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "default_nameservers.#", "1"),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "default_nameservers.0", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMaxUpdated["default_nameservers"])),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "default_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMaxUpdated["default_prefix_length"])),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "max_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMaxUpdated["max_prefix_length"])),
					resource.TestCheckResourceAttr("stackit_network_area.network_area", "min_prefix_length", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMaxUpdated["min_prefix_length"])),

					// Network Area Route
					resource.TestCheckResourceAttrPair(
						"stackit_network_area_route.network_area_route", "organization_id",
						"stackit_network_area.network_area", "organization_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_area_route.network_area_route", "network_area_id",
						"stackit_network_area.network_area", "network_area_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_area_route.network_area_route", "network_area_route_id"),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "prefix", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMaxUpdated["route_prefix"])),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "next_hop", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMaxUpdated["route_next_hop"])),
					resource.TestCheckResourceAttr("stackit_network_area_route.network_area_route", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkAreaVarsMaxUpdated["label"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccVolumeMin(t *testing.T) {
	t.Logf("TestAccVolumeMin name: null")
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVolumeVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceVolumeMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Volume size
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "project_id", testutil.ConvertConfigVariable(testConfigVolumeVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_volume.volume_size", "volume_id"),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "availability_zone", testutil.ConvertConfigVariable(testConfigVolumeVarsMin["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "size", testutil.ConvertConfigVariable(testConfigVolumeVarsMin["size"])),
					resource.TestCheckResourceAttrSet("stackit_volume.volume_size", "performance_class"),
					resource.TestCheckNoResourceAttr("stackit_volume.volume_size", "server_id"),

					// Volume source
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "project_id", testutil.ConvertConfigVariable(testConfigVolumeVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_volume.volume_source", "volume_id"),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "availability_zone", testutil.ConvertConfigVariable(testConfigVolumeVarsMin["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "size", testutil.ConvertConfigVariable(testConfigVolumeVarsMin["size"])),
					resource.TestCheckResourceAttrSet("stackit_volume.volume_source", "performance_class"),
					resource.TestCheckResourceAttrPair(
						"stackit_volume.volume_source", "source.id",
						"stackit_volume.volume_size", "volume_id",
					),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "source.type", "volume"),
					resource.TestCheckNoResourceAttr("stackit_volume.volume_source", "server_id"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVolumeVarsMin,
				Config: fmt.Sprintf(`
					%s
					%s
			
					data "stackit_volume" "volume_size" {
						project_id  = stackit_volume.volume_size.project_id
						volume_id = stackit_volume.volume_size.volume_id
					}

					data "stackit_volume" "volume_source" {
						project_id  = stackit_volume.volume_source.project_id
						volume_id = stackit_volume.volume_source.volume_id
					}
					`,
					testutil.IaaSProviderConfig(), resourceVolumeMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Volume size
					resource.TestCheckResourceAttr("data.stackit_volume.volume_size", "project_id", testutil.ConvertConfigVariable(testConfigVolumeVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_volume.volume_size", "volume_id",
						"data.stackit_volume.volume_size", "volume_id",
					),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_size", "availability_zone", testutil.ConvertConfigVariable(testConfigVolumeVarsMin["availability_zone"])),
					resource.TestCheckResourceAttrSet("data.stackit_volume.volume_size", "performance_class"),
					resource.TestCheckNoResourceAttr("data.stackit_volume.volume_size", "server_id"),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_size", "size", testutil.ConvertConfigVariable(testConfigVolumeVarsMin["size"])),

					// Volume source
					resource.TestCheckResourceAttr("data.stackit_volume.volume_source", "project_id", testutil.ConvertConfigVariable(testConfigVolumeVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_volume.volume_source", "volume_id",
						"data.stackit_volume.volume_source", "volume_id",
					),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_source", "availability_zone", testutil.ConvertConfigVariable(testConfigVolumeVarsMin["availability_zone"])),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_source", "size", testutil.ConvertConfigVariable(testConfigVolumeVarsMin["size"])),
					resource.TestCheckResourceAttrSet("data.stackit_volume.volume_source", "performance_class"),
					resource.TestCheckNoResourceAttr("data.stackit_volume.volume_source", "server_id"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_volume.volume_source", "source.id",
						"data.stackit_volume.volume_size", "volume_id",
					),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_source", "source.type", "volume"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVolumeVarsMin,
				ResourceName:    "stackit_volume.volume_size",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_volume.volume_size"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_volume.volume_size")
					}
					volumeId, ok := r.Primary.Attributes["volume_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute volume_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, volumeId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigVolumeVarsMin,
				ResourceName:    "stackit_volume.volume_source",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_volume.volume_source"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_volume.volume")
					}
					volumeId, ok := r.Primary.Attributes["volume_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute volume_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, volumeId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVolumeVarsMinUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceVolumeMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Volume size
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "project_id", testutil.ConvertConfigVariable(testConfigVolumeVarsMinUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_volume.volume_size", "volume_id"),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "availability_zone", testutil.ConvertConfigVariable(testConfigVolumeVarsMinUpdated["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "size", testutil.ConvertConfigVariable(testConfigVolumeVarsMinUpdated["size"])),
					resource.TestCheckResourceAttrSet("stackit_volume.volume_size", "performance_class"),
					resource.TestCheckNoResourceAttr("stackit_volume.volume_size", "server_id"),

					// Volume source
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "project_id", testutil.ConvertConfigVariable(testConfigVolumeVarsMinUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_volume.volume_source", "volume_id"),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "availability_zone", testutil.ConvertConfigVariable(testConfigVolumeVarsMinUpdated["availability_zone"])),
					// Volume from source doesn't change size. So here the initial size will be used
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "size", testutil.ConvertConfigVariable(testConfigVolumeVarsMin["size"])),
					resource.TestCheckResourceAttrSet("stackit_volume.volume_source", "performance_class"),
					resource.TestCheckResourceAttrPair(
						"stackit_volume.volume_source", "source.id",
						"stackit_volume.volume_size", "volume_id",
					),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "source.type", "volume"),
					resource.TestCheckNoResourceAttr("stackit_volume.volume_source", "server_id"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccVolumeMax(t *testing.T) {
	t.Logf("TestAccVolumeMax name: %s", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigVolumeVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceVolumeMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Volume size
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "project_id", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_volume.volume_size", "volume_id"),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "availability_zone", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "size", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["size"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "description", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["description"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "performance_class", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["performance_class"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "name", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "labels.acc-test", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["label"])),
					resource.TestCheckNoResourceAttr("stackit_volume.volume_size", "server_id"),

					// Volume source
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "project_id", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_volume.volume_source", "volume_id"),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "availability_zone", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "size", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["size"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "description", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["description"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "performance_class", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["performance_class"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "name", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "labels.acc-test", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["label"])),
					resource.TestCheckResourceAttrPair(
						"stackit_volume.volume_source", "source.id",
						"stackit_volume.volume_size", "volume_id",
					),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "source.type", "volume"),
					resource.TestCheckNoResourceAttr("stackit_volume.volume_source", "server_id"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigVolumeVarsMax,
				Config: fmt.Sprintf(`
					%s
					%s
			
					data "stackit_volume" "volume_size" {
						project_id  = stackit_volume.volume_size.project_id
						volume_id = stackit_volume.volume_size.volume_id
					}

					data "stackit_volume" "volume_source" {
						project_id  = stackit_volume.volume_source.project_id
						volume_id = stackit_volume.volume_source.volume_id
					}
					`,
					testutil.IaaSProviderConfig(), resourceVolumeMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Volume size
					resource.TestCheckResourceAttr("data.stackit_volume.volume_size", "project_id", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_volume.volume_size", "volume_id",
						"data.stackit_volume.volume_size", "volume_id",
					),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_size", "availability_zone", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["availability_zone"])),
					resource.TestCheckNoResourceAttr("data.stackit_volume.volume_size", "server_id"),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_size", "size", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["size"])),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_size", "description", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["description"])),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_size", "performance_class", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["performance_class"])),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_size", "name", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_size", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_size", "labels.acc-test", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["label"])),

					// Volume source
					resource.TestCheckResourceAttr("data.stackit_volume.volume_source", "project_id", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_volume.volume_source", "volume_id"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_volume.volume_source", "volume_id",
						"stackit_volume.volume_source", "volume_id",
					),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_source", "availability_zone", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["availability_zone"])),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_source", "size", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["size"])),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_source", "description", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["description"])),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_source", "performance_class", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["performance_class"])),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_source", "name", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_source", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_source", "labels.acc-test", testutil.ConvertConfigVariable(testConfigVolumeVarsMax["label"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_volume.volume_source", "source.id",
						"data.stackit_volume.volume_size", "volume_id",
					),
					resource.TestCheckResourceAttr("data.stackit_volume.volume_source", "source.type", "volume"),
					resource.TestCheckNoResourceAttr("data.stackit_volume.volume_source", "server_id"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigVolumeVarsMax,
				ResourceName:    "stackit_volume.volume_size",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_volume.volume_size"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_volume.volume_size")
					}
					volumeId, ok := r.Primary.Attributes["volume_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute volume_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, volumeId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigVolumeVarsMax,
				ResourceName:    "stackit_volume.volume_source",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_volume.volume_source"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_volume.volume_source")
					}
					volumeId, ok := r.Primary.Attributes["volume_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute volume_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, volumeId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigVolumeVarsMaxUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceVolumeMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Volume size
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "project_id", testutil.ConvertConfigVariable(testConfigVolumeVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_volume.volume_size", "volume_id"),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "availability_zone", testutil.ConvertConfigVariable(testConfigVolumeVarsMaxUpdated["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "size", testutil.ConvertConfigVariable(testConfigVolumeVarsMaxUpdated["size"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "description", testutil.ConvertConfigVariable(testConfigVolumeVarsMaxUpdated["description"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "performance_class", testutil.ConvertConfigVariable(testConfigVolumeVarsMaxUpdated["performance_class"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "name", testutil.ConvertConfigVariable(testConfigVolumeVarsMaxUpdated["name"])),
					resource.TestCheckNoResourceAttr("stackit_volume.volume_size", "server_id"),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_volume.volume_size", "labels.acc-test", testutil.ConvertConfigVariable(testConfigVolumeVarsMaxUpdated["label"])),

					// Volume source
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "project_id", testutil.ConvertConfigVariable(testConfigVolumeVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_volume.volume_source", "volume_id"),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "availability_zone", testutil.ConvertConfigVariable(testConfigVolumeVarsMaxUpdated["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "size", testutil.ConvertConfigVariable(testConfigVolumeVarsMaxUpdated["size"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "description", testutil.ConvertConfigVariable(testConfigVolumeVarsMaxUpdated["description"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "performance_class", testutil.ConvertConfigVariable(testConfigVolumeVarsMaxUpdated["performance_class"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "name", testutil.ConvertConfigVariable(testConfigVolumeVarsMaxUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "labels.acc-test", testutil.ConvertConfigVariable(testConfigVolumeVarsMaxUpdated["label"])),
					resource.TestCheckResourceAttrPair(
						"stackit_volume.volume_source", "source.id",
						"stackit_volume.volume_size", "volume_id",
					),
					resource.TestCheckResourceAttr("stackit_volume.volume_source", "source.type", "volume"),
					resource.TestCheckNoResourceAttr("stackit_volume.volume_source", "server_id"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccServerMin(t *testing.T) {
	t.Logf("TestAccServerMin name: %s", testutil.ConvertConfigVariable(testConfigServerVarsMin["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigServerVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceServerMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Server
					resource.TestCheckResourceAttr("stackit_server.server", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_server.server", "name", testutil.ConvertConfigVariable(testConfigServerVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_server.server", "machine_type", testutil.ConvertConfigVariable(testConfigServerVarsMin["machine_type"])),
					resource.TestCheckResourceAttrSet("stackit_server.server", "boot_volume.%"),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.source_type", "image"),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.source_id", testutil.ConvertConfigVariable(testConfigServerVarsMin["image_id"])),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.delete_on_termination", "true"),
					resource.TestCheckNoResourceAttr("stackit_server.server", "boot_volume.performance_class"),
					resource.TestCheckResourceAttrSet("stackit_server.server", "boot_volume.size"),
					resource.TestCheckResourceAttrSet("stackit_server.server", "boot_volume.id"),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.source_type", "image"),
					resource.TestCheckNoResourceAttr("stackit_server.server", "image_id"),
					resource.TestCheckResourceAttr("stackit_server.server", "labels.%", "0"),
					resource.TestCheckResourceAttrSet("stackit_server.server", "server_id"),
					resource.TestCheckResourceAttrSet("stackit_server.server", "availability_zone"),
					resource.TestCheckNoResourceAttr("stackit_server.server", "desired_status"),
					resource.TestCheckNoResourceAttr("stackit_server.server", "user_data"),
					resource.TestCheckNoResourceAttr("stackit_server.server", "keypair_name"),
					resource.TestCheckNoResourceAttr("stackit_server.server", "network_interfaces"),
					resource.TestCheckResourceAttrSet("stackit_server.server", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_server.server", "launched_at"),
					resource.TestCheckResourceAttrSet("stackit_server.server", "updated_at"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigServerVarsMin,
				Config: fmt.Sprintf(`
						%s
						%s

						data "stackit_server" "server" {
							project_id  = stackit_server.server.project_id
							server_id = stackit_server.server.server_id
						}
						`,
					testutil.IaaSProviderConfig(), resourceServerMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Server
					resource.TestCheckResourceAttr("data.stackit_server.server", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMin["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_server.server", "name", testutil.ConvertConfigVariable(testConfigServerVarsMin["name"])),
					resource.TestCheckResourceAttr("data.stackit_server.server", "machine_type", testutil.ConvertConfigVariable(testConfigServerVarsMin["machine_type"])),
					resource.TestCheckResourceAttrSet("data.stackit_server.server", "boot_volume.%"),
					// boot_volume.attributes are unknown in the datasource. only boot_volume.id and boot_volume.delete_on_termination are returned from the api
					resource.TestCheckNoResourceAttr("data.stackit_server.server", "boot_volume.source_type"),
					resource.TestCheckNoResourceAttr("data.stackit_server.server", "boot_volume.source_id"),
					resource.TestCheckNoResourceAttr("data.stackit_server.server", "boot_volume.size"),
					resource.TestCheckNoResourceAttr("data.stackit_server.server", "boot_volume.performance_class"),
					resource.TestCheckNoResourceAttr("data.stackit_server.server", "boot_volume.source_type"),
					resource.TestCheckResourceAttr("data.stackit_server.server", "boot_volume.delete_on_termination", "true"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_server.server", "boot_volume.id",
						"stackit_server.server", "boot_volume.id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_server.server", "server_id",
						"stackit_server.server", "server_id",
					),
					resource.TestCheckNoResourceAttr("data.stackit_server.server", "image_id"),
					resource.TestCheckResourceAttr("data.stackit_server.server", "labels.%", "0"),
					resource.TestCheckResourceAttrSet("data.stackit_server.server", "server_id"),
					resource.TestCheckResourceAttrSet("data.stackit_server.server", "availability_zone"),
					resource.TestCheckNoResourceAttr("data.stackit_server.server", "desired_status"),
					resource.TestCheckNoResourceAttr("data.stackit_server.server", "user_data"),
					resource.TestCheckNoResourceAttr("data.stackit_server.server", "keypair_name"),
					resource.TestCheckNoResourceAttr("data.stackit_server.server", "network_interfaces"),
					resource.TestCheckResourceAttrSet("data.stackit_server.server", "created_at"),
					resource.TestCheckResourceAttrSet("data.stackit_server.server", "launched_at"),
					resource.TestCheckResourceAttrSet("data.stackit_server.server", "updated_at"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigServerVarsMin,
				ResourceName:    "stackit_server.server",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_server.server"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_server.server")
					}
					serverId, ok := r.Primary.Attributes["server_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute server_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, serverId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"boot_volume", "network_interfaces"}, // Field is not mapped as it is only relevant on creation
			},
			// Update
			{
				ConfigVariables: testConfigServerVarsMinUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceServerMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Server
					resource.TestCheckResourceAttr("stackit_server.server", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMinUpdated["project_id"])),
					resource.TestCheckResourceAttr("stackit_server.server", "name", testutil.ConvertConfigVariable(testConfigServerVarsMinUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_server.server", "machine_type", testutil.ConvertConfigVariable(testConfigServerVarsMinUpdated["machine_type"])),
					resource.TestCheckResourceAttrSet("stackit_server.server", "boot_volume.%"),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.source_type", "image"),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.source_id", testutil.ConvertConfigVariable(testConfigServerVarsMinUpdated["image_id"])),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.delete_on_termination", "true"),
					resource.TestCheckNoResourceAttr("stackit_server.server", "boot_volume.performance_class"),
					resource.TestCheckResourceAttrSet("stackit_server.server", "boot_volume.size"),
					resource.TestCheckResourceAttrSet("stackit_server.server", "boot_volume.id"),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.source_type", "image"),
					resource.TestCheckNoResourceAttr("stackit_server.server", "image_id"),
					resource.TestCheckResourceAttr("stackit_server.server", "labels.%", "0"),
					resource.TestCheckResourceAttrSet("stackit_server.server", "server_id"),
					resource.TestCheckResourceAttrSet("stackit_server.server", "availability_zone"),
					resource.TestCheckNoResourceAttr("stackit_server.server", "desired_status"),
					resource.TestCheckNoResourceAttr("stackit_server.server", "user_data"),
					resource.TestCheckNoResourceAttr("stackit_server.server", "keypair_name"),
					resource.TestCheckNoResourceAttr("stackit_server.server", "network_interfaces"),
					resource.TestCheckResourceAttrSet("stackit_server.server", "created_at"),
					resource.TestCheckResourceAttrSet("stackit_server.server", "launched_at"),
					resource.TestCheckResourceAttrSet("stackit_server.server", "updated_at"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccServerMax(t *testing.T) {
	t.Logf("TestAccServerMax name: %s", testutil.ConvertConfigVariable(testConfigServerVarsMax["name"]))
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigServerVarsMax,
				Config:          fmt.Sprintf("%s\n%s\n%s", testutil.IaaSProviderConfig(), resourceServerMaxConfig, resourceServerMaxAttachmentConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Affinity group
					resource.TestCheckResourceAttr("stackit_affinity_group.affinity_group", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_affinity_group.affinity_group", "name", testutil.ConvertConfigVariable(testConfigServerVarsMax["name_not_updated"])),
					resource.TestCheckResourceAttr("stackit_affinity_group.affinity_group", "policy", testutil.ConvertConfigVariable(testConfigServerVarsMax["policy"])),
					resource.TestCheckResourceAttrSet("stackit_affinity_group.affinity_group", "affinity_group_id"),

					// Volume base
					resource.TestCheckResourceAttr("stackit_volume.base_volume", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_volume.base_volume", "availability_zone", testutil.ConvertConfigVariable(testConfigServerVarsMax["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_volume.base_volume", "size", testutil.ConvertConfigVariable(testConfigServerVarsMax["size"])),
					resource.TestCheckResourceAttr("stackit_volume.base_volume", "source.id", testutil.ConvertConfigVariable(testConfigServerVarsMax["image_id"])),
					resource.TestCheckResourceAttr("stackit_volume.base_volume", "source.type", "image"),
					resource.TestCheckResourceAttrSet("stackit_volume.base_volume", "volume_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_volume.base_volume", "volume_id",
						"stackit_server.server", "boot_volume.source_id",
					),

					// Volume data
					resource.TestCheckResourceAttr("stackit_volume.data_volume", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_volume.data_volume", "availability_zone", testutil.ConvertConfigVariable(testConfigServerVarsMax["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_volume.data_volume", "size", testutil.ConvertConfigVariable(testConfigServerVarsMax["size"])),
					resource.TestCheckResourceAttrSet("stackit_volume.data_volume", "volume_id"),

					// Volume data attach
					resource.TestCheckResourceAttr("stackit_server_volume_attach.data_volume_attachment", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_server_volume_attach.data_volume_attachment", "server_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_server_volume_attach.data_volume_attachment", "server_id",
						"stackit_server.server", "server_id",
					),
					resource.TestCheckResourceAttrSet("stackit_server_volume_attach.data_volume_attachment", "volume_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_volume.data_volume", "volume_id",
						"stackit_server_volume_attach.data_volume_attachment", "volume_id",
					),

					// Network
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "name", testutil.ConvertConfigVariable(testConfigServerVarsMax["name"])),

					// Network interface init
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface_init", "project_id",
						"stackit_network.network", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface_init", "network_id",
						"stackit_network.network", "network_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface_init", "network_interface_id"),

					// Network interface second
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface_second", "project_id",
						"stackit_network.network", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface_second", "network_id",
						"stackit_network.network", "network_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface_second", "network_interface_id"),

					// Network interface attachment
					resource.TestCheckResourceAttr("stackit_server_network_interface_attach.network_interface_second_attachment", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_server_network_interface_attach.network_interface_second_attachment", "network_interface_id",
						"stackit_network_interface.network_interface_second", "network_interface_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_server_network_interface_attach.network_interface_second_attachment", "server_id",
						"stackit_server.server", "server_id",
					),

					// Keypair
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "name", testutil.ConvertConfigVariable(testConfigServerVarsMax["name_not_updated"])),
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "public_key", testutil.ConvertConfigVariable(testConfigServerVarsMax["public_key"])),

					// Service account attachment
					resource.TestCheckResourceAttrPair(
						"stackit_server_service_account_attach.attached_service_account", "project_id",
						"stackit_server.server", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_server_service_account_attach.attached_service_account", "server_id",
						"stackit_server.server", "server_id",
					),
					resource.TestCheckResourceAttr(
						"stackit_server_service_account_attach.attached_service_account", "service_account_email",
						testutil.ConvertConfigVariable(testConfigServerVarsMax["service_account_mail"]),
					),

					// Server
					resource.TestCheckResourceAttr("stackit_server.server", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_server.server", "server_id"),
					resource.TestCheckResourceAttr("stackit_server.server", "name", testutil.ConvertConfigVariable(testConfigServerVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_server.server", "machine_type", testutil.ConvertConfigVariable(testConfigServerVarsMax["machine_type"])),
					resource.TestCheckResourceAttr("stackit_server.server", "desired_status", testutil.ConvertConfigVariable(testConfigServerVarsMax["desired_status"])),
					resource.TestCheckResourceAttrPair(
						"stackit_server.server", "affinity_group",
						"stackit_affinity_group.affinity_group", "affinity_group_id",
					),
					resource.TestCheckResourceAttr("stackit_server.server", "availability_zone", testutil.ConvertConfigVariable(testConfigServerVarsMax["availability_zone"])),
					resource.TestCheckResourceAttrPair(
						"stackit_key_pair.key_pair", "name",
						"stackit_server.server", "keypair_name",
					),
					// The network interface which was attached by "stackit_server_network_interface_attach" should not appear here
					resource.TestCheckResourceAttr("stackit_server.server", "network_interfaces.#", "1"),
					resource.TestCheckResourceAttrPair(
						"stackit_server.server", "network_interfaces.0",
						"stackit_network_interface.network_interface_init", "network_interface_id",
					),
					resource.TestCheckResourceAttr("stackit_server.server", "user_data", testutil.ConvertConfigVariable(testConfigServerVarsMax["user_data"])),
					resource.TestCheckResourceAttrSet("stackit_server.server", "boot_volume.id"),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.source_type", "volume"),
					resource.TestCheckResourceAttrPair(
						"stackit_server.server", "boot_volume.source_id",
						"stackit_volume.base_volume", "volume_id",
					),
					resource.TestCheckResourceAttr("stackit_server.server", "labels.acc-test", testutil.ConvertConfigVariable(testConfigServerVarsMax["label"])),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigServerVarsMax,
				Config: fmt.Sprintf(`
						%s
						%s
						%s

						data "stackit_server" "server" {
							project_id  = stackit_server.server.project_id
							server_id = stackit_server.server.server_id
						}
						`,
					testutil.IaaSProviderConfig(), resourceServerMaxConfig, resourceServerMaxAttachmentConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Server
					resource.TestCheckResourceAttr("data.stackit_server.server", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_server.server", "server_id"),
					resource.TestCheckResourceAttr("data.stackit_server.server", "name", testutil.ConvertConfigVariable(testConfigServerVarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_server.server", "machine_type", testutil.ConvertConfigVariable(testConfigServerVarsMax["machine_type"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_server.server", "affinity_group",
						"stackit_affinity_group.affinity_group", "affinity_group_id",
					),
					resource.TestCheckResourceAttr("data.stackit_server.server", "availability_zone", testutil.ConvertConfigVariable(testConfigServerVarsMax["availability_zone"])),
					resource.TestCheckResourceAttrPair(
						"stackit_key_pair.key_pair", "name",
						"data.stackit_server.server", "keypair_name",
					),
					// All network interface which was are attached appear here
					resource.TestCheckResourceAttr("data.stackit_server.server", "network_interfaces.#", "2"),
					resource.TestCheckTypeSetElemAttrPair(
						"data.stackit_server.server", "network_interfaces.*",
						"stackit_network_interface.network_interface_init", "network_interface_id",
					),
					resource.TestCheckTypeSetElemAttrPair(
						"data.stackit_server.server", "network_interfaces.*",
						"stackit_network_interface.network_interface_second", "network_interface_id",
					),
					resource.TestCheckResourceAttr("data.stackit_server.server", "user_data", testutil.ConvertConfigVariable(testConfigServerVarsMax["user_data"])),
					resource.TestCheckResourceAttrSet("data.stackit_server.server", "boot_volume.id"),
					resource.TestCheckNoResourceAttr("data.stackit_server.server", "boot_volume.source_type"),
					resource.TestCheckNoResourceAttr("data.stackit_server.server", "boot_volume.source_id"),
					resource.TestCheckResourceAttr("data.stackit_server.server", "labels.acc-test", testutil.ConvertConfigVariable(testConfigServerVarsMax["label"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigServerVarsMax,
				ResourceName:    "stackit_affinity_group.affinity_group",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_affinity_group.affinity_group"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_affinity_group.affinity_group")
					}
					affinityGroupId, ok := r.Primary.Attributes["affinity_group_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute affinity_group_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, affinityGroupId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigServerVarsMax,
				ResourceName:    "stackit_volume.base_volume",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_volume.base_volume"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_volume.base_volume")
					}
					volumeId, ok := r.Primary.Attributes["volume_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute volume_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, volumeId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigServerVarsMax,
				ResourceName:    "stackit_volume.data_volume",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_volume.data_volume"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_volume.data_volume")
					}
					volumeId, ok := r.Primary.Attributes["volume_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute volume_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, volumeId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigServerVarsMax,
				ResourceName:    "stackit_server_volume_attach.data_volume_attachment",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_server_volume_attach.data_volume_attachment"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_server_volume_attach.data_volume_attachment")
					}
					serverId, ok := r.Primary.Attributes["server_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute server_id")
					}
					volumeId, ok := r.Primary.Attributes["volume_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute volume_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, serverId, volumeId), nil
				},
				ImportState:       true,
				ImportStateVerify: false,
			},
			{
				ConfigVariables: testConfigServerVarsMax,
				ResourceName:    "stackit_network.network",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network.network"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network.network")
					}
					networkId, ok := r.Primary.Attributes["network_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, networkId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ipv4_prefix_length", "ipv4_prefix"}, // Field is not returned by the API
			},
			{
				ConfigVariables: testConfigServerVarsMax,
				ResourceName:    "stackit_network_interface.network_interface_init",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network_interface.network_interface_init"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network_interface.network_interface_init")
					}
					networkId, ok := r.Primary.Attributes["network_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_id")
					}
					networkInterfaceId, ok := r.Primary.Attributes["network_interface_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_interface_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, networkId, networkInterfaceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigServerVarsMax,
				ResourceName:    "stackit_network_interface.network_interface_second",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network_interface.network_interface_second"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network_interface.network_interface_second")
					}
					networkId, ok := r.Primary.Attributes["network_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_id")
					}
					networkInterfaceId, ok := r.Primary.Attributes["network_interface_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_interface_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, networkId, networkInterfaceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigServerVarsMax,
				ResourceName:    "stackit_server_network_interface_attach.network_interface_second_attachment",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_server_network_interface_attach.network_interface_second_attachment"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_server_network_interface_attach.network_interface_second_attachment")
					}
					serverId, ok := r.Primary.Attributes["server_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_id")
					}
					networkInterfaceId, ok := r.Primary.Attributes["network_interface_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_interface_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, serverId, networkInterfaceId), nil
				},
				ImportState:       true,
				ImportStateVerify: false,
			},
			{
				ConfigVariables: testConfigServerVarsMax,
				ResourceName:    "stackit_key_pair.key_pair",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_key_pair.key_pair"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_key_pair.key_pair")
					}
					keyPairName, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return keyPairName, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigServerVarsMax,
				ResourceName:    "stackit_server_service_account_attach.attached_service_account",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_server_service_account_attach.attached_service_account"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_server_service_account_attach.attached_service_account")
					}
					serverId, ok := r.Primary.Attributes["server_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute server_id")
					}
					serviceAccountEmail, ok := r.Primary.Attributes["service_account_email"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute volume_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, serverId, serviceAccountEmail), nil
				},
				ImportState:       true,
				ImportStateVerify: false,
			},
			{
				ConfigVariables: testConfigServerVarsMax,
				ResourceName:    "stackit_server.server",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_server.server"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_server.server")
					}
					serverId, ok := r.Primary.Attributes["server_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute server_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, serverId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"boot_volume", "desired_status", "network_interfaces"}, // Field is not mapped as it is only relevant on creation
			},
			// Update
			{
				ConfigVariables: testConfigServerVarsMaxUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceServerMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Affinity group
					resource.TestCheckResourceAttr("stackit_affinity_group.affinity_group", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttr("stackit_affinity_group.affinity_group", "name", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["name_not_updated"])),
					resource.TestCheckResourceAttr("stackit_affinity_group.affinity_group", "policy", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["policy"])),
					resource.TestCheckResourceAttrSet("stackit_affinity_group.affinity_group", "affinity_group_id"),

					// Volume base
					resource.TestCheckResourceAttr("stackit_volume.base_volume", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttr("stackit_volume.base_volume", "availability_zone", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_volume.base_volume", "size", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["size"])),
					resource.TestCheckResourceAttr("stackit_volume.base_volume", "source.id", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["image_id"])),
					resource.TestCheckResourceAttr("stackit_volume.base_volume", "source.type", "image"),
					resource.TestCheckResourceAttrSet("stackit_volume.base_volume", "volume_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_volume.base_volume", "volume_id",
						"stackit_server.server", "boot_volume.source_id",
					),

					// Volume data
					resource.TestCheckResourceAttr("stackit_volume.data_volume", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttr("stackit_volume.data_volume", "availability_zone", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_volume.data_volume", "size", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["size"])),
					resource.TestCheckResourceAttrSet("stackit_volume.data_volume", "volume_id"),

					// Network
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "name", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["name"])),

					// Network interface init
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface_init", "project_id",
						"stackit_network.network", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface_init", "network_id",
						"stackit_network.network", "network_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface_init", "network_interface_id"),

					// Network interface second
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface_second", "project_id",
						"stackit_network.network", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface_second", "network_id",
						"stackit_network.network", "network_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface_second", "network_interface_id"),

					// Keypair
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "name", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["name_not_updated"])),
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "public_key", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["public_key"])),

					// Service account attachment
					resource.TestCheckResourceAttrPair(
						"stackit_server_service_account_attach.attached_service_account", "project_id",
						"stackit_server.server", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_server_service_account_attach.attached_service_account", "server_id",
						"stackit_server.server", "server_id",
					),
					resource.TestCheckResourceAttr(
						"stackit_server_service_account_attach.attached_service_account", "service_account_email",
						testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["service_account_mail"]),
					),

					// Server
					resource.TestCheckResourceAttr("stackit_server.server", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_server.server", "server_id"),
					resource.TestCheckResourceAttr("stackit_server.server", "name", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_server.server", "machine_type", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["machine_type"])),
					resource.TestCheckResourceAttr("stackit_server.server", "desired_status", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["desired_status"])),
					resource.TestCheckResourceAttrPair(
						"stackit_server.server", "affinity_group",
						"stackit_affinity_group.affinity_group", "affinity_group_id",
					),
					resource.TestCheckResourceAttr("stackit_server.server", "availability_zone", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["availability_zone"])),
					resource.TestCheckResourceAttrPair(
						"stackit_key_pair.key_pair", "name",
						"stackit_server.server", "keypair_name",
					),
					// The network interface which was attached by "stackit_server_network_interface_attach" should not appear here
					resource.TestCheckResourceAttr("stackit_server.server", "network_interfaces.#", "1"),
					resource.TestCheckResourceAttrPair(
						"stackit_server.server", "network_interfaces.0",
						"stackit_network_interface.network_interface_init", "network_interface_id",
					),
					resource.TestCheckResourceAttr("stackit_server.server", "user_data", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["user_data"])),
					resource.TestCheckResourceAttrSet("stackit_server.server", "boot_volume.id"),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.source_type", "volume"),
					resource.TestCheckResourceAttrPair(
						"stackit_server.server", "boot_volume.source_id",
						"stackit_volume.base_volume", "volume_id",
					),
					resource.TestCheckResourceAttr("stackit_server.server", "labels.acc-test", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdated["label"])),
				),
			},
			// Updated desired status
			{
				ConfigVariables: testConfigServerVarsMaxUpdatedDesiredStatus,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceServerMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Affinity group
					resource.TestCheckResourceAttr("stackit_affinity_group.affinity_group", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["project_id"])),
					resource.TestCheckResourceAttr("stackit_affinity_group.affinity_group", "name", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["name_not_updated"])),
					resource.TestCheckResourceAttr("stackit_affinity_group.affinity_group", "policy", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["policy"])),
					resource.TestCheckResourceAttrSet("stackit_affinity_group.affinity_group", "affinity_group_id"),

					// Volume base
					resource.TestCheckResourceAttr("stackit_volume.base_volume", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["project_id"])),
					resource.TestCheckResourceAttr("stackit_volume.base_volume", "availability_zone", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_volume.base_volume", "size", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["size"])),
					resource.TestCheckResourceAttr("stackit_volume.base_volume", "source.id", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["image_id"])),
					resource.TestCheckResourceAttr("stackit_volume.base_volume", "source.type", "image"),
					resource.TestCheckResourceAttrSet("stackit_volume.base_volume", "volume_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_volume.base_volume", "volume_id",
						"stackit_server.server", "boot_volume.source_id",
					),

					// Volume data
					resource.TestCheckResourceAttr("stackit_volume.data_volume", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["project_id"])),
					resource.TestCheckResourceAttr("stackit_volume.data_volume", "availability_zone", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["availability_zone"])),
					resource.TestCheckResourceAttr("stackit_volume.data_volume", "size", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["size"])),
					resource.TestCheckResourceAttrSet("stackit_volume.data_volume", "volume_id"),

					// Network
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "name", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["name"])),

					// Network interface init
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface_init", "project_id",
						"stackit_network.network", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface_init", "network_id",
						"stackit_network.network", "network_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface_init", "network_interface_id"),

					// Network interface second
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface_second", "project_id",
						"stackit_network.network", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface_second", "network_id",
						"stackit_network.network", "network_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface_second", "network_interface_id"),

					// Keypair
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "name", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["name_not_updated"])),
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "public_key", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["public_key"])),

					// Service account attachment
					resource.TestCheckResourceAttrPair(
						"stackit_server_service_account_attach.attached_service_account", "project_id",
						"stackit_server.server", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_server_service_account_attach.attached_service_account", "server_id",
						"stackit_server.server", "server_id",
					),
					resource.TestCheckResourceAttr(
						"stackit_server_service_account_attach.attached_service_account", "service_account_email",
						testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["service_account_mail"]),
					),

					// Server
					resource.TestCheckResourceAttr("stackit_server.server", "project_id", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_server.server", "server_id"),
					resource.TestCheckResourceAttr("stackit_server.server", "name", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["name"])),
					resource.TestCheckResourceAttr("stackit_server.server", "machine_type", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["machine_type"])),
					resource.TestCheckResourceAttr("stackit_server.server", "desired_status", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["desired_status"])),
					resource.TestCheckResourceAttrPair(
						"stackit_server.server", "affinity_group",
						"stackit_affinity_group.affinity_group", "affinity_group_id",
					),
					resource.TestCheckResourceAttr("stackit_server.server", "availability_zone", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["availability_zone"])),
					resource.TestCheckResourceAttrPair(
						"stackit_key_pair.key_pair", "name",
						"stackit_server.server", "keypair_name",
					),
					// The network interface which was attached by "stackit_server_network_interface_attach" should not appear here
					resource.TestCheckResourceAttr("stackit_server.server", "network_interfaces.#", "1"),
					resource.TestCheckResourceAttrPair(
						"stackit_server.server", "network_interfaces.0",
						"stackit_network_interface.network_interface_init", "network_interface_id",
					),
					resource.TestCheckResourceAttr("stackit_server.server", "user_data", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["user_data"])),
					resource.TestCheckResourceAttrSet("stackit_server.server", "boot_volume.id"),
					resource.TestCheckResourceAttr("stackit_server.server", "boot_volume.source_type", "volume"),
					resource.TestCheckResourceAttrPair(
						"stackit_server.server", "boot_volume.source_id",
						"stackit_volume.base_volume", "volume_id",
					),
					resource.TestCheckResourceAttr("stackit_server.server", "labels.acc-test", testutil.ConvertConfigVariable(testConfigServerVarsMaxUpdatedDesiredStatus["label"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccAffinityGroupMin(t *testing.T) {
	t.Logf("TestAccAffinityGroupMin name: %s", testutil.ConvertConfigVariable(testConfigAffinityGroupVarsMin["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigAffinityGroupVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceAffinityGroupMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_affinity_group.affinity_group", "project_id", testutil.ConvertConfigVariable(testConfigAffinityGroupVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_affinity_group.affinity_group", "affinity_group_id"),
					resource.TestCheckResourceAttr("stackit_affinity_group.affinity_group", "name", testutil.ConvertConfigVariable(testConfigAffinityGroupVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_affinity_group.affinity_group", "policy", testutil.ConvertConfigVariable(testConfigAffinityGroupVarsMin["policy"])),
					resource.TestCheckNoResourceAttr("stackit_affinity_group.affinity_group", "members.#"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigAffinityGroupVarsMin,
				Config: fmt.Sprintf(`
					%s
					%s
			
					data "stackit_affinity_group" "affinity_group" {
						project_id  = stackit_affinity_group.affinity_group.project_id
						affinity_group_id = stackit_affinity_group.affinity_group.affinity_group_id
					}
					`,
					testutil.IaaSProviderConfig(), resourceAffinityGroupMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stackit_affinity_group.affinity_group", "project_id", testutil.ConvertConfigVariable(testConfigAffinityGroupVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_affinity_group.affinity_group", "affinity_group_id",
						"data.stackit_affinity_group.affinity_group", "affinity_group_id",
					),
					resource.TestCheckResourceAttr("data.stackit_affinity_group.affinity_group", "name", testutil.ConvertConfigVariable(testConfigAffinityGroupVarsMin["name"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigAffinityGroupVarsMin,
				ResourceName:    "stackit_affinity_group.affinity_group",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_affinity_group.affinity_group"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_affinity_group.affinity_group")
					}
					affinityGroupId, ok := r.Primary.Attributes["affinity_group_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute affinity_group_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, affinityGroupId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// In this minimal setup, no update can be performed
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccIaaSSecurityGroupMin(t *testing.T) {
	t.Logf("TestAccIaaSSecurityGroupMin name: %s", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMin["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				ConfigVariables: testConfigSecurityGroupsVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceSecurityGroupMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Security Group
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "project_id", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_security_group.security_group", "security_group_id"),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "name", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMin["name"])),
					resource.TestCheckResourceAttrSet("stackit_security_group.security_group", "stateful"),

					// Security Group Rule
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "project_id",
						"stackit_security_group.security_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "security_group_id",
						"stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrSet("stackit_security_group_rule.security_group_rule", "security_group_rule_id"),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "direction", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMin["direction"])),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigSecurityGroupsVarsMin,
				Config: fmt.Sprintf(`
					%s
					%s
			
					data "stackit_security_group" "security_group" {
						project_id  = stackit_security_group.security_group.project_id
						security_group_id = stackit_security_group.security_group.security_group_id
					}

					data "stackit_security_group_rule" "security_group_rule" {
						project_id             = stackit_security_group.security_group.project_id
						security_group_id      = stackit_security_group.security_group.security_group_id
						security_group_rule_id = stackit_security_group_rule.security_group_rule.security_group_rule_id
					}
					`,
					testutil.IaaSProviderConfig(), resourceSecurityGroupMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_security_group.security_group", "project_id", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_security_group.security_group", "security_group_id",
						"data.stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttr("data.stackit_security_group.security_group", "name", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMin["name"])),

					// Security Group Rule
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "project_id",
						"stackit_security_group.security_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "security_group_id",
						"stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrSet("stackit_security_group_rule.security_group_rule", "security_group_rule_id"),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule", "direction", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMin["direction"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigSecurityGroupsVarsMin,
				ResourceName:    "stackit_security_group.security_group",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_security_group.security_group"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_security_group.security_group")
					}
					securityGroupId, ok := r.Primary.Attributes["security_group_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute security_group_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, securityGroupId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigSecurityGroupsVarsMin,
				ResourceName:    "stackit_security_group_rule.security_group_rule",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_security_group_rule.security_group_rule"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_security_group_rule.security_group_rule")
					}
					securityGroupId, ok := r.Primary.Attributes["security_group_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute security_group_id")
					}
					securityGroupRuleId, ok := r.Primary.Attributes["security_group_rule_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute security_group_rule_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, securityGroupId, securityGroupRuleId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigSecurityGroupsVarsMinUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceSecurityGroupMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Security Group
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "project_id", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMinUpdated()["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_security_group.security_group", "security_group_id"),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "name", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMinUpdated()["name"])),
					resource.TestCheckResourceAttrSet("stackit_security_group.security_group", "stateful"),

					// Security Group Rule
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "project_id",
						"stackit_security_group.security_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "security_group_id",
						"stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrSet("stackit_security_group_rule.security_group_rule", "security_group_rule_id"),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "direction", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMinUpdated()["direction"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccIaaSSecurityGroupMax(t *testing.T) {
	t.Logf("TestAccIaaSSecurityGroupMax name: %s", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				ConfigVariables: testConfigSecurityGroupsVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceSecurityGroupMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Security Group (default)
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "project_id", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_security_group.security_group", "security_group_id"),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "name", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "description", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["description"])),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "stateful", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["stateful"])),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "labels.acc-test", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["label"])),

					// Security Group (remote)
					resource.TestCheckResourceAttr("stackit_security_group.security_group_remote", "project_id", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_security_group.security_group_remote", "security_group_id"),
					resource.TestCheckResourceAttr("stackit_security_group.security_group_remote", "name", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["name_remote"])),

					// Security Group Rule (default)
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "project_id",
						"stackit_security_group.security_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "security_group_id",
						"stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrSet("stackit_security_group_rule.security_group_rule", "security_group_rule_id"),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "direction", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["direction"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "description", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["description_rule"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "ether_type", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["ether_type"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "port_range.min", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["port"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "port_range.max", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["port"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "protocol.name", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["protocol"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "ip_range", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["ip_range"])),

					// Security Group Rule (icmp)
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule_icmp", "project_id",
						"stackit_security_group.security_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule_icmp", "security_group_id",
						"stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrSet("stackit_security_group_rule.security_group_rule_icmp", "security_group_rule_id"),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule_icmp", "direction", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["direction"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule_icmp", "description", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["description_rule"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule_icmp", "ether_type", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["ether_type"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule_icmp", "icmp_parameters.code", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["icmp_code"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule_icmp", "icmp_parameters.type", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["icmp_type"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule_icmp", "protocol.name", "icmp"),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule_icmp", "ip_range", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["ip_range"])),

					// Security Group Rule (remote)
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "project_id",
						"stackit_security_group.security_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "security_group_id",
						"stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule_remote_security_group", "remote_security_group_id",
						"stackit_security_group.security_group_remote", "security_group_id",
					),
					resource.TestCheckResourceAttrSet("stackit_security_group_rule.security_group_rule", "security_group_rule_id"),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "direction", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["direction"])),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigSecurityGroupsVarsMax,
				Config: fmt.Sprintf(`
					%s
					%s
			
					data "stackit_security_group" "security_group" {
						project_id  = stackit_security_group.security_group.project_id
						security_group_id = stackit_security_group.security_group.security_group_id
					}

					data "stackit_security_group" "security_group_remote" {
						project_id  = stackit_security_group.security_group_remote.project_id
						security_group_id = stackit_security_group.security_group_remote.security_group_id
					}

					data "stackit_security_group_rule" "security_group_rule" {
						project_id             = stackit_security_group.security_group.project_id
						security_group_id      = stackit_security_group.security_group.security_group_id
						security_group_rule_id = stackit_security_group_rule.security_group_rule.security_group_rule_id
					}

					data "stackit_security_group_rule" "security_group_rule_icmp" {
						project_id             = stackit_security_group.security_group.project_id
						security_group_id      = stackit_security_group.security_group.security_group_id
						security_group_rule_id = stackit_security_group_rule.security_group_rule_icmp.security_group_rule_id
					}

					data "stackit_security_group_rule" "security_group_rule_remote_security_group" {
						project_id             = stackit_security_group.security_group.project_id
						security_group_id      = stackit_security_group.security_group.security_group_id
						security_group_rule_id = stackit_security_group_rule.security_group_rule_remote_security_group.security_group_rule_id
					}
					`,
					testutil.IaaSProviderConfig(), resourceSecurityGroupMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Security Group (default)
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group.security_group", "project_id",
						"stackit_security_group.security_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group.security_group", "security_group_id",
						"stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttr("data.stackit_security_group.security_group", "project_id", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_security_group.security_group", "security_group_id"),
					resource.TestCheckResourceAttr("data.stackit_security_group.security_group", "name", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_security_group.security_group", "description", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["description"])),
					resource.TestCheckResourceAttr("data.stackit_security_group.security_group", "stateful", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["stateful"])),
					resource.TestCheckResourceAttr("data.stackit_security_group.security_group", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_security_group.security_group", "labels.acc-test", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["label"])),

					// Security Group (remote)
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group.security_group_remote", "project_id",
						"stackit_security_group.security_group_remote", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group.security_group_remote", "security_group_id",
						"stackit_security_group.security_group_remote", "security_group_id",
					),
					resource.TestCheckResourceAttr("data.stackit_security_group.security_group_remote", "project_id", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("data.stackit_security_group.security_group_remote", "security_group_id"),
					resource.TestCheckResourceAttr("data.stackit_security_group.security_group_remote", "name", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["name_remote"])),

					// Security Group Rule (default)
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule", "project_id",
						"data.stackit_security_group.security_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule", "security_group_id",
						"data.stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule", "security_group_rule_id",
						"stackit_security_group_rule.security_group_rule", "security_group_rule_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule", "project_id",
						"stackit_security_group_rule.security_group_rule", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule", "security_group_id",
						"stackit_security_group_rule.security_group_rule", "security_group_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_security_group_rule.security_group_rule", "security_group_rule_id"),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule", "direction", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["direction"])),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule", "description", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["description_rule"])),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule", "ether_type", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["ether_type"])),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule", "port_range.min", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["port"])),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule", "port_range.max", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["port"])),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule", "protocol.name", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["protocol"])),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule", "ip_range", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["ip_range"])),

					// Security Group Rule (icmp)
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule_icmp", "project_id",
						"data.stackit_security_group.security_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule_icmp", "security_group_id",
						"data.stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule_icmp", "security_group_rule_id",
						"stackit_security_group_rule.security_group_rule_icmp", "security_group_rule_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule_icmp", "project_id",
						"stackit_security_group_rule.security_group_rule_icmp", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule_icmp", "security_group_id",
						"stackit_security_group_rule.security_group_rule_icmp", "security_group_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_security_group_rule.security_group_rule_icmp", "security_group_rule_id"),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule_icmp", "direction", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["direction"])),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule_icmp", "description", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["description_rule"])),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule_icmp", "ether_type", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["ether_type"])),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule_icmp", "icmp_parameters.code", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["icmp_code"])),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule_icmp", "icmp_parameters.type", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["icmp_type"])),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule_icmp", "protocol.name", "icmp"),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule_icmp", "ip_range", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["ip_range"])),

					// Security Group Rule (remote)
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule_remote_security_group", "project_id",
						"data.stackit_security_group.security_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule_remote_security_group", "security_group_id",
						"data.stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule_remote_security_group", "security_group_rule_id",
						"stackit_security_group_rule.security_group_rule_remote_security_group", "security_group_rule_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule_remote_security_group", "project_id",
						"stackit_security_group_rule.security_group_rule_remote_security_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule_remote_security_group", "security_group_id",
						"stackit_security_group_rule.security_group_rule_remote_security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule_remote_security_group", "remote_security_group_id",
						"stackit_security_group_rule.security_group_rule_remote_security_group", "remote_security_group_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_security_group_rule.security_group_rule_remote_security_group", "remote_security_group_id",
						"data.stackit_security_group.security_group_remote", "security_group_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_security_group_rule.security_group_rule", "security_group_rule_id"),
					resource.TestCheckResourceAttr("data.stackit_security_group_rule.security_group_rule", "direction", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMax["direction"])),
				),
			},
			// Import
			{
				ConfigVariables: testConfigSecurityGroupsVarsMax,
				ResourceName:    "stackit_security_group.security_group",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_security_group.security_group"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_security_group.security_group")
					}
					securityGroupId, ok := r.Primary.Attributes["security_group_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute security_group_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, securityGroupId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigSecurityGroupsVarsMax,
				ResourceName:    "stackit_security_group_rule.security_group_rule",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_security_group_rule.security_group_rule"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_security_group_rule.security_group_rule")
					}
					securityGroupId, ok := r.Primary.Attributes["security_group_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute security_group_id")
					}
					securityGroupRuleId, ok := r.Primary.Attributes["security_group_rule_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute security_group_rule_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, securityGroupId, securityGroupRuleId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				ConfigVariables: testConfigSecurityGroupsVarsMaxUpdated(),
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceSecurityGroupMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Security Group (default)
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "project_id", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_security_group.security_group", "security_group_id"),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "name", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["name"])),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "description", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["description"])),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "stateful", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["stateful"])),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_security_group.security_group", "labels.acc-test", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["label"])),

					// Security Group (remote)
					resource.TestCheckResourceAttr("stackit_security_group.security_group_remote", "project_id", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_security_group.security_group_remote", "security_group_id"),
					resource.TestCheckResourceAttr("stackit_security_group.security_group_remote", "name", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["name_remote"])),

					// Security Group Rule (default)
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "project_id",
						"stackit_security_group.security_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "security_group_id",
						"stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrSet("stackit_security_group_rule.security_group_rule", "security_group_rule_id"),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "direction", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["direction"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "description", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["description_rule"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "ether_type", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["ether_type"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "port_range.min", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["port"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "port_range.max", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["port"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "protocol.name", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["protocol"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "ip_range", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["ip_range"])),

					// Security Group Rule (icmp)
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule_icmp", "project_id",
						"stackit_security_group.security_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule_icmp", "security_group_id",
						"stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrSet("stackit_security_group_rule.security_group_rule_icmp", "security_group_rule_id"),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule_icmp", "direction", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["direction"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule_icmp", "description", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["description_rule"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule_icmp", "ether_type", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["ether_type"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule_icmp", "icmp_parameters.code", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["icmp_code"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule_icmp", "icmp_parameters.type", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["icmp_type"])),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule_icmp", "protocol.name", "icmp"),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule_icmp", "ip_range", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["ip_range"])),

					// Security Group Rule (remote)
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "project_id",
						"stackit_security_group.security_group", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule", "security_group_id",
						"stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_security_group_rule.security_group_rule_remote_security_group", "remote_security_group_id",
						"stackit_security_group.security_group_remote", "security_group_id",
					),
					resource.TestCheckResourceAttrSet("stackit_security_group_rule.security_group_rule", "security_group_rule_id"),
					resource.TestCheckResourceAttr("stackit_security_group_rule.security_group_rule", "direction", testutil.ConvertConfigVariable(testConfigSecurityGroupsVarsMaxUpdated()["direction"])),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccNetworkInterfaceMin(t *testing.T) {
	t.Logf("TestAccNetworkInterfaceMin name: %s", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMin["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigNetworkInterfaceVarsMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceNetworkInterfaceMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Network interface instance
					resource.TestCheckNoResourceAttr("stackit_network_interface.network_interface", "name"),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface", "ipv4"),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface", "allowed_addresses.#"),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "security", "true"),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "labels.#", "0"),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "security_group_ids.#", "0"),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface", "mac"),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface", "network_interface_id"),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface", "type"),
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface", "network_id",
						"stackit_network.network", "network_id",
					),

					// Network instance
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMin["project_id"])),
					resource.TestCheckResourceAttr("stackit_network.network", "name", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMin["name"])),
					resource.TestCheckResourceAttrSet("stackit_network.network", "ipv4_prefixes.#"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "public_ip"),

					// Public ip
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_public_ip.public_ip", "public_ip_id"),
					resource.TestCheckResourceAttrSet("stackit_public_ip.public_ip", "ip"),
					resource.TestCheckNoResourceAttr("stackit_public_ip.public_ip", "network_interface_id"),
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip", "labels.%", "0"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigNetworkInterfaceVarsMin,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_network_interface" "network_interface" {
						project_id = stackit_network_interface.network_interface.project_id
						network_id = stackit_network_interface.network_interface.network_id
						network_interface_id = stackit_network_interface.network_interface.network_interface_id
					}

					data "stackit_network" "network" {
						project_id  = stackit_network.network.project_id
						network_id  = stackit_network.network.network_id
					}

					data "stackit_public_ip" "public_ip" {
						project_id   = stackit_public_ip.public_ip.project_id
						public_ip_id = stackit_public_ip.public_ip.public_ip_id
					}
					`,
					testutil.IaaSProviderConfig(), resourceNetworkInterfaceMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Network interface instance
					resource.TestCheckNoResourceAttr("data.stackit_network_interface.network_interface", "name"),
					resource.TestCheckResourceAttrSet("data.stackit_network_interface.network_interface", "ipv4"),
					resource.TestCheckNoResourceAttr("data.stackit_network_interface.network_interface", "allowed_addresses.#"),
					resource.TestCheckResourceAttr("data.stackit_network_interface.network_interface", "security", "true"),
					resource.TestCheckResourceAttr("data.stackit_network_interface.network_interface", "labels.#", "0"),
					resource.TestCheckResourceAttr("data.stackit_network_interface.network_interface", "security_group_ids.#", "0"),
					resource.TestCheckResourceAttrSet("data.stackit_network_interface.network_interface", "mac"),
					resource.TestCheckResourceAttrSet("data.stackit_network_interface.network_interface", "network_interface_id"),
					resource.TestCheckResourceAttrSet("data.stackit_network_interface.network_interface", "type"),

					// Network instance
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("data.stackit_network.network", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMin["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_network.network", "name", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMin["name"])),
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "ipv4_prefixes.#"),
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "public_ip"),

					// Public ip
					resource.TestCheckResourceAttr("data.stackit_public_ip.public_ip", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_public_ip.public_ip", "public_ip_id",
						"stackit_public_ip.public_ip", "public_ip_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_public_ip.public_ip", "ip",
						"stackit_public_ip.public_ip", "ip",
					),
					resource.TestCheckNoResourceAttr("data.stackit_public_ip.public_ip", "network_interface_id"),
					resource.TestCheckResourceAttr("data.stackit_public_ip.public_ip", "labels.%", "0"),
				),
			},

			// Import
			{
				ConfigVariables: testConfigNetworkInterfaceVarsMin,
				ResourceName:    "stackit_network_interface.network_interface",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network_interface.network_interface"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network_interface.network_interface")
					}
					networkId, ok := r.Primary.Attributes["network_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_id")
					}
					networkInterfaceId, ok := r.Primary.Attributes["network_interface_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_interface_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, networkId, networkInterfaceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigNetworkInterfaceVarsMin,
				ResourceName:    "stackit_network.network",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network.network"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network.network")
					}
					networkId, ok := r.Primary.Attributes["network_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, networkId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigNetworkInterfaceVarsMin,
				ResourceName:    "stackit_public_ip.public_ip",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_public_ip.public_ip"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_public_ip.public_ip")
					}
					publicIpId, ok := r.Primary.Attributes["public_ip_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute public_ip_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, publicIpId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// In this minimal setup, no update can be performed
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccNetworkInterfaceMax(t *testing.T) {
	t.Logf("TestAccNetworkInterfaceMax name: %s", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigNetworkInterfaceVarsMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceNetworkInterfaceMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Network interface instance
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "name", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "ipv4", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["ipv4"])),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "allowed_addresses.#", "1"),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "allowed_addresses.0", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["allowed_address"])),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "security", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["security"])),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["label"])),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "security_group_ids.#", "1"),
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface", "network_id",
						"stackit_network.network", "network_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface", "security_group_ids.0",
						"stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface", "mac"),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface", "network_interface_id"),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface", "type"),

					// Network instance
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["project_id"])),
					resource.TestCheckResourceAttr("stackit_network.network", "name", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["name"])),
					resource.TestCheckResourceAttrSet("stackit_network.network", "ipv4_prefixes.#"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "public_ip"),

					// Public ip
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_public_ip.public_ip", "public_ip_id"),
					resource.TestCheckResourceAttrSet("stackit_public_ip.public_ip", "ip"),
					resource.TestCheckResourceAttrPair(
						"stackit_public_ip.public_ip", "network_interface_id",
						"stackit_network_interface.network_interface", "network_interface_id",
					),
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["label"])),

					// Network interface simple
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface_simple", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface_simple", "network_interface_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface_simple", "network_id",
						"stackit_network.network", "network_id",
					),

					// Public ip simple
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip_simple", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_public_ip.public_ip_simple", "public_ip_id"),
					resource.TestCheckResourceAttrSet("stackit_public_ip.public_ip_simple", "ip"),
					resource.TestCheckNoResourceAttr("stackit_public_ip.public_ip_simple", "network_interface_id"),
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip_simple", "labels.%", "0"),

					// Nic and public ip attach
					resource.TestCheckResourceAttr("stackit_public_ip_associate.nic_public_ip_attach", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_public_ip_associate.nic_public_ip_attach", "public_ip_id",
						"stackit_public_ip.public_ip_simple", "public_ip_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_public_ip_associate.nic_public_ip_attach", "network_interface_id",
						"stackit_network_interface.network_interface_simple", "network_interface_id",
					),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigNetworkInterfaceVarsMax,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_network_interface" "network_interface" {
						project_id = stackit_network_interface.network_interface.project_id
						network_id = stackit_network_interface.network_interface.network_id
						network_interface_id = stackit_network_interface.network_interface.network_interface_id
					}

					data "stackit_network" "network" {
						project_id  = stackit_network.network.project_id
						network_id  = stackit_network.network.network_id
					}

					data "stackit_public_ip" "public_ip" {
						project_id   = stackit_public_ip.public_ip.project_id
						public_ip_id = stackit_public_ip.public_ip.public_ip_id
					}

					data "stackit_network_interface" "network_interface_simple" {
						project_id = stackit_network_interface.network_interface_simple.project_id
						network_id = stackit_network_interface.network_interface_simple.network_id
						network_interface_id = stackit_network_interface.network_interface_simple.network_interface_id
					}

					data "stackit_public_ip" "public_ip_simple" {
						project_id   = stackit_public_ip.public_ip_simple.project_id
						public_ip_id = stackit_public_ip.public_ip_simple.public_ip_id
					}
					`,
					testutil.IaaSProviderConfig(), resourceNetworkInterfaceMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Network interface instance
					resource.TestCheckResourceAttr("data.stackit_network_interface.network_interface", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_network_interface.network_interface", "project_id",
						"stackit_network_interface.network_interface", "project_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_network_interface.network_interface", "network_interface_id",
						"stackit_network_interface.network_interface", "network_interface_id",
					),
					resource.TestCheckResourceAttr("data.stackit_network_interface.network_interface", "name", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_network_interface.network_interface", "ipv4", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["ipv4"])),
					resource.TestCheckResourceAttr("data.stackit_network_interface.network_interface", "allowed_addresses.#", "1"),
					resource.TestCheckResourceAttr("data.stackit_network_interface.network_interface", "allowed_addresses.0", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["allowed_address"])),
					resource.TestCheckResourceAttr("data.stackit_network_interface.network_interface", "security", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["security"])),
					resource.TestCheckResourceAttr("data.stackit_network_interface.network_interface", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_network_interface.network_interface", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["label"])),
					resource.TestCheckResourceAttr("data.stackit_network_interface.network_interface", "security_group_ids.#", "1"),
					resource.TestCheckResourceAttrPair(
						"data.stackit_network_interface.network_interface", "security_group_ids.0",
						"stackit_security_group.security_group", "security_group_id",
					),
					resource.TestCheckResourceAttrSet("data.stackit_network_interface.network_interface", "mac"),
					resource.TestCheckResourceAttrSet("data.stackit_network_interface.network_interface", "network_interface_id"),
					resource.TestCheckResourceAttrSet("data.stackit_network_interface.network_interface", "type"),

					// Network instance
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("data.stackit_network.network", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["project_id"])),
					resource.TestCheckResourceAttr("data.stackit_network.network", "name", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["name"])),
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "ipv4_prefixes.#"),
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttrSet("data.stackit_network.network", "public_ip"),

					// Public ip
					resource.TestCheckResourceAttr("data.stackit_public_ip.public_ip", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_public_ip.public_ip", "public_ip_id",
						"stackit_public_ip.public_ip", "public_ip_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_public_ip.public_ip", "ip",
						"stackit_public_ip.public_ip", "ip",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_public_ip.public_ip", "network_interface_id",
						"data.stackit_network_interface.network_interface", "network_interface_id",
					),
					resource.TestCheckResourceAttr("data.stackit_public_ip.public_ip", "labels.%", "1"),
					resource.TestCheckResourceAttr("data.stackit_public_ip.public_ip", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["label"])),

					// Network interface simple
					resource.TestCheckNoResourceAttr("data.stackit_network_interface.network_interface_simple", "name"),
					resource.TestCheckResourceAttrSet("data.stackit_network_interface.network_interface_simple", "ipv4"),
					resource.TestCheckNoResourceAttr("data.stackit_network_interface.network_interface_simple", "allowed_addresses.#"),
					resource.TestCheckResourceAttr("data.stackit_network_interface.network_interface_simple", "security", "true"),
					resource.TestCheckResourceAttr("data.stackit_network_interface.network_interface_simple", "labels.#", "0"),
					resource.TestCheckResourceAttr("data.stackit_network_interface.network_interface_simple", "security_group_ids.#", "0"),
					resource.TestCheckResourceAttrSet("data.stackit_network_interface.network_interface_simple", "mac"),
					resource.TestCheckResourceAttrSet("data.stackit_network_interface.network_interface_simple", "network_interface_id"),
					resource.TestCheckResourceAttrSet("data.stackit_network_interface.network_interface_simple", "type"),

					// Public ip simple
					resource.TestCheckResourceAttr("data.stackit_public_ip.public_ip_simple", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"data.stackit_public_ip.public_ip_simple", "public_ip_id",
						"stackit_public_ip.public_ip_simple", "public_ip_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_public_ip.public_ip_simple", "ip",
						"stackit_public_ip.public_ip_simple", "ip",
					),
					resource.TestCheckResourceAttrPair(
						"data.stackit_public_ip.public_ip_simple", "network_interface_id",
						"data.stackit_network_interface.network_interface_simple", "network_interface_id",
					),
					resource.TestCheckResourceAttr("data.stackit_public_ip.public_ip_simple", "labels.%", "0"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigNetworkInterfaceVarsMax,
				ResourceName:    "stackit_network_interface.network_interface",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network_interface.network_interface"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network_interface.network_interface")
					}
					networkId, ok := r.Primary.Attributes["network_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_id")
					}
					networkInterfaceId, ok := r.Primary.Attributes["network_interface_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_interface_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, networkId, networkInterfaceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigNetworkInterfaceVarsMax,
				ResourceName:    "stackit_network.network",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network.network"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network.network")
					}
					networkId, ok := r.Primary.Attributes["network_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, networkId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigNetworkInterfaceVarsMax,
				ResourceName:    "stackit_public_ip.public_ip",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_public_ip.public_ip"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_public_ip.public_ip")
					}
					publicIpId, ok := r.Primary.Attributes["public_ip_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute public_ip_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, publicIpId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigNetworkInterfaceVarsMax,
				ResourceName:    "stackit_network_interface.network_interface_simple",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_network_interface.network_interface_simple"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_network_interface.network_interface_simple")
					}
					networkId, ok := r.Primary.Attributes["network_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_id")
					}
					networkInterfaceId, ok := r.Primary.Attributes["network_interface_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_interface_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, networkId, networkInterfaceId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigNetworkInterfaceVarsMax,
				ResourceName:    "stackit_public_ip.public_ip_simple",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_public_ip.public_ip_simple"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_public_ip.public_ip_simple")
					}
					publicIpId, ok := r.Primary.Attributes["public_ip_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute public_ip_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, publicIpId), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigNetworkInterfaceVarsMax,
				ResourceName:    "stackit_public_ip_associate.nic_public_ip_attach",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_public_ip.public_ip"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_public_ip.public_ip")
					}
					publicIpId, ok := r.Primary.Attributes["public_ip_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute public_ip_id")
					}
					networkInterfaceId, ok := r.Primary.Attributes["network_interface_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute network_interface_id")
					}
					return fmt.Sprintf("%s,%s,%s", testutil.ProjectId, publicIpId, networkInterfaceId), nil
				},
				ImportState: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_public_ip_associate.nic_public_ip_attach", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_public_ip_associate.nic_public_ip_attach", "public_ip_id",
						"stackit_public_ip.public_ip_simple", "public_ip_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_public_ip_associate.nic_public_ip_attach", "network_interface_id",
						"stackit_network_interface.network_interface_simple", "network_interface_id",
					),
				),
			},
			// Update
			{
				ConfigVariables: testConfigNetworkInterfaceVarsMaxUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceNetworkInterfaceMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Network interface instance
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "name", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMaxUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "ipv4", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMaxUpdated["ipv4"])),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "allowed_addresses.#", "0"),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "security", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMaxUpdated["security"])),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMaxUpdated["label"])),
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface", "security_group_ids.#", "0"),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface", "mac"),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface", "network_interface_id"),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface", "type"),

					// Network instance
					resource.TestCheckResourceAttrSet("stackit_network.network", "network_id"),
					resource.TestCheckResourceAttr("stackit_network.network", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttr("stackit_network.network", "name", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMaxUpdated["name"])),
					resource.TestCheckResourceAttrSet("stackit_network.network", "ipv4_prefixes.#"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "ipv6_prefixes.#"),
					resource.TestCheckResourceAttrSet("stackit_network.network", "public_ip"),

					// Public ip
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_public_ip.public_ip", "public_ip_id"),
					resource.TestCheckResourceAttrSet("stackit_public_ip.public_ip", "ip"),
					resource.TestCheckResourceAttrPair(
						"stackit_public_ip.public_ip", "network_interface_id",
						"stackit_network_interface.network_interface", "network_interface_id",
					),
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip", "labels.%", "1"),
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip", "labels.acc-test", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMaxUpdated["label"])),

					// Network interface simple
					resource.TestCheckResourceAttr("stackit_network_interface.network_interface_simple", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_network_interface.network_interface_simple", "network_interface_id"),
					resource.TestCheckResourceAttrPair(
						"stackit_network_interface.network_interface_simple", "network_id",
						"stackit_network.network", "network_id",
					),

					// Public ip simple
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip_simple", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_public_ip.public_ip_simple", "public_ip_id"),
					resource.TestCheckResourceAttrSet("stackit_public_ip.public_ip_simple", "ip"),
					resource.TestCheckResourceAttrPair(
						"stackit_public_ip.public_ip_simple", "network_interface_id",
						"stackit_network_interface.network_interface_simple", "network_interface_id",
					),
					resource.TestCheckResourceAttr("stackit_public_ip.public_ip_simple", "labels.%", "0"),

					// Nic and public ip attach
					resource.TestCheckResourceAttr("stackit_public_ip_associate.nic_public_ip_attach", "project_id", testutil.ConvertConfigVariable(testConfigNetworkInterfaceVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttrPair(
						"stackit_public_ip_associate.nic_public_ip_attach", "public_ip_id",
						"stackit_public_ip.public_ip_simple", "public_ip_id",
					),
					resource.TestCheckResourceAttrPair(
						"stackit_public_ip_associate.nic_public_ip_attach", "network_interface_id",
						"stackit_network_interface.network_interface_simple", "network_interface_id",
					),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccKeyPairMin(t *testing.T) {
	t.Logf("TestAccKeyPairMin name: %s", testutil.ConvertConfigVariable(testConfigKeyPairMin["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigKeyPairMin,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceKeyPairMinConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "name", testutil.ConvertConfigVariable(testConfigKeyPairMin["name"])),
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "public_key", testutil.ConvertConfigVariable(testConfigKeyPairMin["public_key"])),
					resource.TestCheckResourceAttrSet("stackit_key_pair.key_pair", "fingerprint"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigKeyPairMin,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_key_pair" "key_pair" {
						name = stackit_key_pair.key_pair.name
					}
					`,
					testutil.IaaSProviderConfig(), resourceKeyPairMinConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_key_pair.key_pair", "name", testutil.ConvertConfigVariable(testConfigKeyPairMin["name"])),
					resource.TestCheckResourceAttr("data.stackit_key_pair.key_pair", "public_key", testutil.ConvertConfigVariable(testConfigKeyPairMin["public_key"])),
					resource.TestCheckResourceAttrSet("data.stackit_key_pair.key_pair", "fingerprint"),
					resource.TestCheckResourceAttrPair(
						"stackit_key_pair.key_pair", "fingerprint",
						"data.stackit_key_pair.key_pair", "fingerprint",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigKeyPairMin,
				ResourceName:    "stackit_key_pair.key_pair",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_key_pair.key_pair"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_key_pair.key_pair")
					}
					keyPairName, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return keyPairName, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// In this minimal setup, no update can be performed
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccKeyPairMax(t *testing.T) {
	t.Logf("TestAccKeyPairMax name: %s", testutil.ConvertConfigVariable(testConfigKeyPairMax["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{
			// Creation
			{
				ConfigVariables: testConfigKeyPairMax,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceKeyPairMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "name", testutil.ConvertConfigVariable(testConfigKeyPairMax["name"])),
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "public_key", testutil.ConvertConfigVariable(testConfigKeyPairMax["public_key"])),
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "labels.acc-test", testutil.ConvertConfigVariable(testConfigKeyPairMax["label"])),
					resource.TestCheckResourceAttrSet("stackit_key_pair.key_pair", "fingerprint"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigKeyPairMax,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_key_pair" "key_pair" {
						name = stackit_key_pair.key_pair.name
					}
					`,
					testutil.IaaSProviderConfig(), resourceKeyPairMaxConfig,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_key_pair.key_pair", "name", testutil.ConvertConfigVariable(testConfigKeyPairMax["name"])),
					resource.TestCheckResourceAttr("data.stackit_key_pair.key_pair", "public_key", testutil.ConvertConfigVariable(testConfigKeyPairMax["public_key"])),
					resource.TestCheckResourceAttr("data.stackit_key_pair.key_pair", "labels.acc-test", testutil.ConvertConfigVariable(testConfigKeyPairMax["label"])),
					resource.TestCheckResourceAttrSet("data.stackit_key_pair.key_pair", "fingerprint"),
					resource.TestCheckResourceAttrPair(
						"stackit_key_pair.key_pair", "fingerprint",
						"data.stackit_key_pair.key_pair", "fingerprint",
					),
				),
			},
			// Import
			{
				ConfigVariables: testConfigKeyPairMax,
				ResourceName:    "stackit_key_pair.key_pair",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_key_pair.key_pair"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_key_pair.key_pair")
					}
					keyPairName, ok := r.Primary.Attributes["name"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute name")
					}
					return keyPairName, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigVariables: testConfigKeyPairMaxUpdated,
				Config:          fmt.Sprintf("%s\n%s", testutil.IaaSProviderConfig(), resourceKeyPairMaxConfig),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "name", testutil.ConvertConfigVariable(testConfigKeyPairMaxUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "public_key", testutil.ConvertConfigVariable(testConfigKeyPairMaxUpdated["public_key"])),
					resource.TestCheckResourceAttr("stackit_key_pair.key_pair", "labels.acc-test", testutil.ConvertConfigVariable(testConfigKeyPairMaxUpdated["label"])),
					resource.TestCheckResourceAttrSet("stackit_key_pair.key_pair", "fingerprint"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccImageMin(t *testing.T) {
	t.Logf("TestAccImageMin name: %s", testutil.ConvertConfigVariable(testConfigImageVarsMin["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				ConfigVariables: testConfigImageVarsMin,
				Config:          fmt.Sprintf("%s\n%s", resourceImageMinConfig, testutil.IaaSProviderConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_image.image", "project_id", testutil.ConvertConfigVariable(testConfigImageVarsMin["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_image.image", "image_id"),
					resource.TestCheckResourceAttr("stackit_image.image", "name", testutil.ConvertConfigVariable(testConfigImageVarsMin["name"])),
					resource.TestCheckResourceAttr("stackit_image.image", "disk_format", testutil.ConvertConfigVariable(testConfigImageVarsMin["disk_format"])),
					resource.TestCheckResourceAttr("stackit_image.image", "local_file_path", testutil.ConvertConfigVariable(testConfigImageVarsMin["local_file_path"])),
					resource.TestCheckResourceAttrSet("stackit_image.image", "protected"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "scope"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.algorithm"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.digest"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.algorithm"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.digest"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigImageVarsMin,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_image" "image" {
						project_id = stackit_image.image.project_id
						image_id = stackit_image.image.image_id
					}
					`,
					resourceImageMinConfig, testutil.IaaSProviderConfig(),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_image.image", "project_id", testutil.ConvertConfigVariable(testConfigImageVarsMin["project_id"])),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "image_id", "stackit_image.image", "image_id"),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "name", "stackit_image.image", "name"),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "disk_format", "stackit_image.image", "disk_format"),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "min_disk_size", "stackit_image.image", "min_disk_size"),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "min_ram", "stackit_image.image", "min_ram"),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "protected", "stackit_image.image", "protected"),
					resource.TestCheckResourceAttrSet("data.stackit_image.image", "protected"),
					resource.TestCheckResourceAttrSet("data.stackit_image.image", "scope"),
					resource.TestCheckResourceAttrSet("data.stackit_image.image", "checksum.algorithm"),
					resource.TestCheckResourceAttrSet("data.stackit_image.image", "checksum.digest"),
					resource.TestCheckResourceAttrSet("data.stackit_image.image", "checksum.algorithm"),
					resource.TestCheckResourceAttrSet("data.stackit_image.image", "checksum.digest"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigImageVarsMin,
				ResourceName:    "stackit_image.image",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_image.image"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_image.image")
					}
					imageId, ok := r.Primary.Attributes["image_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute image_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, imageId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"local_file_path"},
			},
			// Update
			{
				ConfigVariables: testConfigImageVarsMinUpdated,
				Config:          fmt.Sprintf("%s\n%s", resourceImageMinConfig, testutil.IaaSProviderConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_image.image", "project_id", testutil.ConvertConfigVariable(testConfigImageVarsMinUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_image.image", "image_id"),
					resource.TestCheckResourceAttr("stackit_image.image", "name", testutil.ConvertConfigVariable(testConfigImageVarsMinUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_image.image", "disk_format", testutil.ConvertConfigVariable(testConfigImageVarsMinUpdated["disk_format"])),
					resource.TestCheckResourceAttr("stackit_image.image", "local_file_path", testutil.ConvertConfigVariable(testConfigImageVarsMinUpdated["local_file_path"])),
					resource.TestCheckResourceAttrSet("stackit_image.image", "protected"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "scope"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.algorithm"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.digest"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.algorithm"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.digest"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func TestAccImageMax(t *testing.T) {
	t.Logf("TestAccImageMax name: %s", testutil.ConvertConfigVariable(testConfigImageVarsMax["name"]))
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroy,
		Steps: []resource.TestStep{

			// Creation
			{
				ConfigVariables: testConfigImageVarsMax,
				Config:          fmt.Sprintf("%s\n%s", resourceImageMaxConfig, testutil.IaaSProviderConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_image.image", "project_id", testutil.ConvertConfigVariable(testConfigImageVarsMax["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_image.image", "image_id"),
					resource.TestCheckResourceAttr("stackit_image.image", "name", testutil.ConvertConfigVariable(testConfigImageVarsMax["name"])),
					resource.TestCheckResourceAttr("stackit_image.image", "disk_format", testutil.ConvertConfigVariable(testConfigImageVarsMax["disk_format"])),
					resource.TestCheckResourceAttr("stackit_image.image", "local_file_path", testutil.ConvertConfigVariable(testConfigImageVarsMax["local_file_path"])),
					resource.TestCheckResourceAttr("stackit_image.image", "min_disk_size", testutil.ConvertConfigVariable(testConfigImageVarsMax["min_disk_size"])),
					resource.TestCheckResourceAttr("stackit_image.image", "min_ram", testutil.ConvertConfigVariable(testConfigImageVarsMax["min_ram"])),
					resource.TestCheckResourceAttr("stackit_image.image", "labels.acc-test", testutil.ConvertConfigVariable(testConfigImageVarsMax["label"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.boot_menu", testutil.ConvertConfigVariable(testConfigImageVarsMax["boot_menu"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.cdrom_bus", testutil.ConvertConfigVariable(testConfigImageVarsMax["cdrom_bus"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.disk_bus", testutil.ConvertConfigVariable(testConfigImageVarsMax["disk_bus"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.nic_model", testutil.ConvertConfigVariable(testConfigImageVarsMax["nic_model"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.operating_system", testutil.ConvertConfigVariable(testConfigImageVarsMax["operating_system"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.operating_system_distro", testutil.ConvertConfigVariable(testConfigImageVarsMax["operating_system_distro"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.operating_system_version", testutil.ConvertConfigVariable(testConfigImageVarsMax["operating_system_version"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.rescue_bus", testutil.ConvertConfigVariable(testConfigImageVarsMax["rescue_bus"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.rescue_device", testutil.ConvertConfigVariable(testConfigImageVarsMax["rescue_device"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.secure_boot", testutil.ConvertConfigVariable(testConfigImageVarsMax["secure_boot"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.uefi", testutil.ConvertConfigVariable(testConfigImageVarsMax["uefi"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.video_model", testutil.ConvertConfigVariable(testConfigImageVarsMax["video_model"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.virtio_scsi", testutil.ConvertConfigVariable(testConfigImageVarsMax["virtio_scsi"])),
					resource.TestCheckResourceAttrSet("stackit_image.image", "protected"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "scope"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.algorithm"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.digest"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.algorithm"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.digest"),
				),
			},
			// Data source
			{
				ConfigVariables: testConfigImageVarsMax,
				Config: fmt.Sprintf(`
					%s
					%s

					data "stackit_image" "image" {
						project_id = stackit_image.image.project_id
						image_id = stackit_image.image.image_id
					}
					`,
					resourceImageMaxConfig, testutil.IaaSProviderConfig(),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Instance
					resource.TestCheckResourceAttr("data.stackit_image.image", "project_id", testutil.ConvertConfigVariable(testConfigImageVarsMax["project_id"])),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "image_id", "stackit_image.image", "image_id"),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "name", "stackit_image.image", "name"),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "disk_format", "stackit_image.image", "disk_format"),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "min_disk_size", "stackit_image.image", "min_disk_size"),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "min_ram", "stackit_image.image", "min_ram"),
					resource.TestCheckResourceAttrPair("data.stackit_image.image", "protected", "stackit_image.image", "protected"),
					resource.TestCheckResourceAttr("data.stackit_image.image", "min_disk_size", testutil.ConvertConfigVariable(testConfigImageVarsMax["min_disk_size"])),
					resource.TestCheckResourceAttr("data.stackit_image.image", "min_ram", testutil.ConvertConfigVariable(testConfigImageVarsMax["min_ram"])),
					resource.TestCheckResourceAttr("data.stackit_image.image", "labels.acc-test", testutil.ConvertConfigVariable(testConfigImageVarsMax["label"])),
					resource.TestCheckResourceAttr("data.stackit_image.image", "config.boot_menu", testutil.ConvertConfigVariable(testConfigImageVarsMax["boot_menu"])),
					resource.TestCheckResourceAttr("data.stackit_image.image", "config.cdrom_bus", testutil.ConvertConfigVariable(testConfigImageVarsMax["cdrom_bus"])),
					resource.TestCheckResourceAttr("data.stackit_image.image", "config.disk_bus", testutil.ConvertConfigVariable(testConfigImageVarsMax["disk_bus"])),
					resource.TestCheckResourceAttr("data.stackit_image.image", "config.nic_model", testutil.ConvertConfigVariable(testConfigImageVarsMax["nic_model"])),
					resource.TestCheckResourceAttr("data.stackit_image.image", "config.operating_system", testutil.ConvertConfigVariable(testConfigImageVarsMax["operating_system"])),
					resource.TestCheckResourceAttr("data.stackit_image.image", "config.operating_system_distro", testutil.ConvertConfigVariable(testConfigImageVarsMax["operating_system_distro"])),
					resource.TestCheckResourceAttr("data.stackit_image.image", "config.operating_system_version", testutil.ConvertConfigVariable(testConfigImageVarsMax["operating_system_version"])),
					resource.TestCheckResourceAttr("data.stackit_image.image", "config.rescue_bus", testutil.ConvertConfigVariable(testConfigImageVarsMax["rescue_bus"])),
					resource.TestCheckResourceAttr("data.stackit_image.image", "config.rescue_device", testutil.ConvertConfigVariable(testConfigImageVarsMax["rescue_device"])),
					resource.TestCheckResourceAttr("data.stackit_image.image", "config.secure_boot", testutil.ConvertConfigVariable(testConfigImageVarsMax["secure_boot"])),
					resource.TestCheckResourceAttr("data.stackit_image.image", "config.uefi", testutil.ConvertConfigVariable(testConfigImageVarsMax["uefi"])),
					resource.TestCheckResourceAttr("data.stackit_image.image", "config.video_model", testutil.ConvertConfigVariable(testConfigImageVarsMax["video_model"])),
					resource.TestCheckResourceAttr("data.stackit_image.image", "config.virtio_scsi", testutil.ConvertConfigVariable(testConfigImageVarsMax["virtio_scsi"])),
					resource.TestCheckResourceAttrSet("data.stackit_image.image", "protected"),
					resource.TestCheckResourceAttrSet("data.stackit_image.image", "scope"),
					resource.TestCheckResourceAttrSet("data.stackit_image.image", "checksum.algorithm"),
					resource.TestCheckResourceAttrSet("data.stackit_image.image", "checksum.digest"),
					resource.TestCheckResourceAttrSet("data.stackit_image.image", "checksum.algorithm"),
					resource.TestCheckResourceAttrSet("data.stackit_image.image", "checksum.digest"),
				),
			},
			// Import
			{
				ConfigVariables: testConfigImageVarsMax,
				ResourceName:    "stackit_image.image",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					r, ok := s.RootModule().Resources["stackit_image.image"]
					if !ok {
						return "", fmt.Errorf("couldn't find resource stackit_image.image")
					}
					imageId, ok := r.Primary.Attributes["image_id"]
					if !ok {
						return "", fmt.Errorf("couldn't find attribute image_id")
					}
					return fmt.Sprintf("%s,%s", testutil.ProjectId, imageId), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"local_file_path"},
			},
			// Update
			{
				ConfigVariables: testConfigImageVarsMaxUpdated,
				Config:          fmt.Sprintf("%s\n%s", resourceImageMaxConfig, testutil.IaaSProviderConfig()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stackit_image.image", "project_id", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["project_id"])),
					resource.TestCheckResourceAttrSet("stackit_image.image", "image_id"),
					resource.TestCheckResourceAttr("stackit_image.image", "name", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["name"])),
					resource.TestCheckResourceAttr("stackit_image.image", "disk_format", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["disk_format"])),
					resource.TestCheckResourceAttr("stackit_image.image", "local_file_path", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["local_file_path"])),
					resource.TestCheckResourceAttr("stackit_image.image", "min_disk_size", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["min_disk_size"])),
					resource.TestCheckResourceAttr("stackit_image.image", "min_ram", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["min_ram"])),
					resource.TestCheckResourceAttr("stackit_image.image", "labels.acc-test", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["label"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.boot_menu", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["boot_menu"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.cdrom_bus", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["cdrom_bus"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.disk_bus", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["disk_bus"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.nic_model", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["nic_model"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.operating_system", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["operating_system"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.operating_system_distro", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["operating_system_distro"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.operating_system_version", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["operating_system_version"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.rescue_bus", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["rescue_bus"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.rescue_device", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["rescue_device"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.secure_boot", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["secure_boot"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.uefi", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["uefi"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.video_model", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["video_model"])),
					resource.TestCheckResourceAttr("stackit_image.image", "config.virtio_scsi", testutil.ConvertConfigVariable(testConfigImageVarsMaxUpdated["virtio_scsi"])),
					resource.TestCheckResourceAttrSet("stackit_image.image", "protected"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "scope"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.algorithm"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.digest"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.algorithm"),
					resource.TestCheckResourceAttrSet("stackit_image.image", "checksum.digest"),
				),
			},
			// Deletion is done by the framework implicitly
		},
	})
}

func testAccCheckDestroy(s *terraform.State) error {
	checkFunctions := []func(s *terraform.State) error{
		testAccCheckNetworkV1Destroy,
		testAccCheckNetworkInterfaceDestroy,
		testAccCheckNetworkAreaDestroy,
		testAccCheckIaaSVolumeDestroy,
		testAccCheckServerDestroy,
		testAccCheckAffinityGroupDestroy,
		testAccCheckIaaSSecurityGroupDestroy,
		testAccCheckIaaSPublicIpDestroy,
		testAccCheckIaaSKeyPairDestroy,
		testAccCheckIaaSImageDestroy,
	}
	var errs []error

	wg := sync.WaitGroup{}
	wg.Add(len(checkFunctions))

	for _, f := range checkFunctions {
		go func() {
			err := f(s)
			if err != nil {
				errs = append(errs, err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}

func testAccCheckNetworkV1Destroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaas.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
	} else {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var errs []error
	// networks
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_network" {
			continue
		}
		networkId := strings.Split(rs.Primary.ID, core.Separator)[1]
		err := client.DeleteNetworkExecute(ctx, testutil.ProjectId, networkId)
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			if errors.As(err, &oapiErr) {
				if oapiErr.StatusCode == http.StatusNotFound {
					continue
				}
			}
			errs = append(errs, fmt.Errorf("cannot trigger network deletion %q: %w", networkId, err))
		}
		_, err = wait.DeleteNetworkWaitHandler(ctx, client, testutil.ProjectId, networkId).WaitWithContext(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("cannot delete network %q: %w", networkId, err))
		}
	}

	return errors.Join(errs...)
}

func testAccCheckNetworkV2Destroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaasalpha.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaasalpha.NewAPIClient()
	} else {
		client, err = iaasalpha.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var errs []error
	// networks
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_network" {
			continue
		}
		region := strings.Split(rs.Primary.ID, core.Separator)[1]
		networkId := strings.Split(rs.Primary.ID, core.Separator)[2]
		err := client.DeleteNetworkExecute(ctx, testutil.ProjectId, region, networkId)
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			if errors.As(err, &oapiErr) {
				if oapiErr.StatusCode == http.StatusNotFound {
					continue
				}
			}
			errs = append(errs, fmt.Errorf("cannot trigger network deletion %q: %w", networkId, err))
		}
		_, err = waitAlpha.DeleteNetworkWaitHandler(ctx, client, testutil.ProjectId, region, networkId).WaitWithContext(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("cannot delete network %q: %w", networkId, err))
		}
	}

	return errors.Join(errs...)
}

func testAccCheckNetworkInterfaceDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaas.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
	} else {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	var errs []error
	// network interfaces
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_network_interface" {
			continue
		}
		ids := strings.Split(rs.Primary.ID, core.Separator)
		networkId := ids[1]
		networkInterfaceId := ids[2]
		err := client.DeleteNicExecute(ctx, testutil.ProjectId, networkId, networkInterfaceId)
		if err != nil {
			var oapiErr *oapierror.GenericOpenAPIError
			if errors.As(err, &oapiErr) {
				if oapiErr.StatusCode == http.StatusNotFound || oapiErr.StatusCode == http.StatusBadRequest {
					continue
				}
			}
			errs = append(errs, fmt.Errorf("cannot trigger network interface deletion %q: %w", networkInterfaceId, err))
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("cannot delete network interface %q: %w", networkInterfaceId, err))
		}
	}

	return errors.Join(errs...)
}

func testAccCheckNetworkAreaDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaas.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
	} else {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	// network areas
	networkAreasToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_network_area" {
			continue
		}
		networkAreaId := strings.Split(rs.Primary.ID, core.Separator)[1]
		networkAreasToDestroy = append(networkAreasToDestroy, networkAreaId)
	}

	networkAreasResp, err := client.ListNetworkAreasExecute(ctx, testutil.OrganizationId)
	if err != nil {
		return fmt.Errorf("getting networkAreasResp: %w", err)
	}

	networkAreas := *networkAreasResp.Items
	for i := range networkAreas {
		if networkAreas[i].AreaId == nil {
			continue
		}
		if utils.Contains(networkAreasToDestroy, *networkAreas[i].AreaId) {
			err := client.DeleteNetworkAreaExecute(ctx, testutil.OrganizationId, *networkAreas[i].AreaId)
			if err != nil {
				return fmt.Errorf("destroying network area %s during CheckDestroy: %w", *networkAreas[i].AreaId, err)
			}
		}
	}
	return nil
}

func testAccCheckIaaSVolumeDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaas.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
	} else {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	volumesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_volume" {
			continue
		}
		// volume terraform ID: "[project_id],[volume_id]"
		volumeId := strings.Split(rs.Primary.ID, core.Separator)[1]
		volumesToDestroy = append(volumesToDestroy, volumeId)
	}

	volumesResp, err := client.ListVolumesExecute(ctx, testutil.ProjectId)
	if err != nil {
		return fmt.Errorf("getting volumesResp: %w", err)
	}

	volumes := *volumesResp.Items
	for i := range volumes {
		if volumes[i].Id == nil {
			continue
		}
		if utils.Contains(volumesToDestroy, *volumes[i].Id) {
			err := client.DeleteVolumeExecute(ctx, testutil.ProjectId, *volumes[i].Id)
			if err != nil {
				return fmt.Errorf("destroying volume %s during CheckDestroy: %w", *volumes[i].Id, err)
			}
		}
	}
	return nil
}

func testAccCheckServerDestroy(s *terraform.State) error {
	ctx := context.Background()
	var alphaClient *iaas.APIClient
	var client *iaas.APIClient
	var err error
	var alphaErr error
	if testutil.IaaSCustomEndpoint == "" {
		alphaClient, alphaErr = iaas.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
	} else {
		alphaClient, alphaErr = iaas.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
	}
	if err != nil || alphaErr != nil {
		return fmt.Errorf("creating client: %w, %w", err, alphaErr)
	}

	// Servers

	serversToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_server" {
			continue
		}
		// server terraform ID: "[project_id],[server_id]"
		serverId := strings.Split(rs.Primary.ID, core.Separator)[1]
		serversToDestroy = append(serversToDestroy, serverId)
	}

	serversResp, err := alphaClient.ListServersExecute(ctx, testutil.ProjectId)
	if err != nil {
		return fmt.Errorf("getting serversResp: %w", err)
	}

	servers := *serversResp.Items
	for i := range servers {
		if servers[i].Id == nil {
			continue
		}
		if utils.Contains(serversToDestroy, *servers[i].Id) {
			err := alphaClient.DeleteServerExecute(ctx, testutil.ProjectId, *servers[i].Id)
			if err != nil {
				return fmt.Errorf("destroying server %s during CheckDestroy: %w", *servers[i].Id, err)
			}
		}
	}

	// Networks

	networksToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_network" {
			continue
		}
		// network terraform ID: "[project_id],[network_id]"
		networkId := strings.Split(rs.Primary.ID, core.Separator)[1]
		networksToDestroy = append(networksToDestroy, networkId)
	}

	networksResp, err := client.ListNetworksExecute(ctx, testutil.ProjectId)
	if err != nil {
		return fmt.Errorf("getting networksResp: %w", err)
	}

	networks := *networksResp.Items
	for i := range networks {
		if networks[i].NetworkId == nil {
			continue
		}
		if utils.Contains(networksToDestroy, *networks[i].NetworkId) {
			err := client.DeleteNetworkExecute(ctx, testutil.ProjectId, *networks[i].NetworkId)
			if err != nil {
				return fmt.Errorf("destroying network %s during CheckDestroy: %w", *networks[i].NetworkId, err)
			}
		}
	}

	return nil
}

func testAccCheckAffinityGroupDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaas.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
	} else {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	affinityGroupsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_affinity_group" {
			continue
		}
		// affinity group terraform ID: "[project_id],[affinity_group_id]"
		affinityGroupId := strings.Split(rs.Primary.ID, core.Separator)[1]
		affinityGroupsToDestroy = append(affinityGroupsToDestroy, affinityGroupId)
	}

	affinityGroupsResp, err := client.ListAffinityGroupsExecute(ctx, testutil.ProjectId)
	if err != nil {
		return fmt.Errorf("getting securityGroupsResp: %w", err)
	}

	affinityGroups := *affinityGroupsResp.Items
	for i := range affinityGroups {
		if affinityGroups[i].Id == nil {
			continue
		}
		if utils.Contains(affinityGroupsToDestroy, *affinityGroups[i].Id) {
			err := client.DeleteAffinityGroupExecute(ctx, testutil.ProjectId, *affinityGroups[i].Id)
			if err != nil {
				return fmt.Errorf("destroying affinity group %s during CheckDestroy: %w", *affinityGroups[i].Id, err)
			}
		}
	}
	return nil
}

func testAccCheckIaaSSecurityGroupDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaas.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
	} else {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	securityGroupsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_security_group" {
			continue
		}
		// security group terraform ID: "[project_id],[security_group_id]"
		securityGroupId := strings.Split(rs.Primary.ID, core.Separator)[1]
		securityGroupsToDestroy = append(securityGroupsToDestroy, securityGroupId)
	}

	securityGroupsResp, err := client.ListSecurityGroupsExecute(ctx, testutil.ProjectId)
	if err != nil {
		return fmt.Errorf("getting securityGroupsResp: %w", err)
	}

	securityGroups := *securityGroupsResp.Items
	for i := range securityGroups {
		if securityGroups[i].Id == nil {
			continue
		}
		if utils.Contains(securityGroupsToDestroy, *securityGroups[i].Id) {
			err := client.DeleteSecurityGroupExecute(ctx, testutil.ProjectId, *securityGroups[i].Id)
			if err != nil {
				return fmt.Errorf("destroying security group %s during CheckDestroy: %w", *securityGroups[i].Id, err)
			}
		}
	}
	return nil
}

func testAccCheckIaaSPublicIpDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaas.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
	} else {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	publicIpsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_public_ip" {
			continue
		}
		// public IP terraform ID: "[project_id],[public_ip_id]"
		publicIpId := strings.Split(rs.Primary.ID, core.Separator)[1]
		publicIpsToDestroy = append(publicIpsToDestroy, publicIpId)
	}

	publicIpsResp, err := client.ListPublicIPsExecute(ctx, testutil.ProjectId)
	if err != nil {
		return fmt.Errorf("getting publicIpsResp: %w", err)
	}

	publicIps := *publicIpsResp.Items
	for i := range publicIps {
		if publicIps[i].Id == nil {
			continue
		}
		if utils.Contains(publicIpsToDestroy, *publicIps[i].Id) {
			err := client.DeletePublicIPExecute(ctx, testutil.ProjectId, *publicIps[i].Id)
			if err != nil {
				return fmt.Errorf("destroying public IP %s during CheckDestroy: %w", *publicIps[i].Id, err)
			}
		}
	}
	return nil
}

func testAccCheckIaaSKeyPairDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaas.APIClient
	var err error
	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
	} else {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	keyPairsToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_key_pair" {
			continue
		}
		// Key pair terraform ID: "[name]"
		keyPairsToDestroy = append(keyPairsToDestroy, rs.Primary.ID)
	}

	keyPairsResp, err := client.ListKeyPairsExecute(ctx)
	if err != nil {
		return fmt.Errorf("getting key pairs: %w", err)
	}

	keyPairs := *keyPairsResp.Items
	for i := range keyPairs {
		if keyPairs[i].Name == nil {
			continue
		}
		if utils.Contains(keyPairsToDestroy, *keyPairs[i].Name) {
			err := client.DeleteKeyPairExecute(ctx, *keyPairs[i].Name)
			if err != nil {
				return fmt.Errorf("destroying key pair %s during CheckDestroy: %w", *keyPairs[i].Name, err)
			}
		}
	}
	return nil
}

func testAccCheckIaaSImageDestroy(s *terraform.State) error {
	ctx := context.Background()
	var client *iaas.APIClient
	var err error

	if testutil.IaaSCustomEndpoint == "" {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithRegion("eu01"),
		)
	} else {
		client, err = iaas.NewAPIClient(
			stackitSdkConfig.WithEndpoint(testutil.IaaSCustomEndpoint),
		)
	}
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	imagesToDestroy := []string{}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "stackit_image" {
			continue
		}
		// Image terraform ID: "[project_id],[image_id]"
		imageId := strings.Split(rs.Primary.ID, core.Separator)[1]
		imagesToDestroy = append(imagesToDestroy, imageId)
	}

	imagesResp, err := client.ListImagesExecute(ctx, testutil.ProjectId)
	if err != nil {
		return fmt.Errorf("getting images: %w", err)
	}

	images := *imagesResp.Items
	for i := range images {
		if images[i].Id == nil {
			continue
		}
		if utils.Contains(imagesToDestroy, *images[i].Id) {
			err := client.DeleteImageExecute(ctx, testutil.ProjectId, *images[i].Id)
			if err != nil {
				return fmt.Errorf("destroying image %s during CheckDestroy: %w", *images[i].Id, err)
			}
		}
	}
	return nil
}
