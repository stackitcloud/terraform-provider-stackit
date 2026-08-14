package vpcnetworkrange

import (
	sdkUtils "github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2alpha1api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

const resourceDescription = "VPC network range resource schema"
const datasourceDescription = "VPC network range datasource schema"

var descriptions = map[string]string{
	"resource.markdown":     features.AddExperimentDescription(resourceDescription, features.VpcExperiment, core.Resource),
	"datasource.markdown":   features.AddExperimentDescription(datasourceDescription, features.VpcExperiment, core.Datasource),
	"id":                    "Terraform's internal resource identifier. It is structured as \"`project_id`,`vpc_id`,`region`,`network_range_id`\".",
	"project_id":            "STACKIT project ID to which the VPC network range is associated.",
	"vpc_id":                "The VPC ID to which the VPC network range is associated.",
	"region":                "The resource region. If not defined, the provider region is used.",
	"description":           "Description of the VPC network range",
	"ip_version":            "IP version of the VPC network range. " + utils.FormatPossibleValues(sdkUtils.EnumSliceToStringSlice(iaas.AllowedNetworkRangeIPv4RequestIpVersionEnumValues)...),
	"default_prefix_length": "Default prefix length for the VPC network range",
	"max_prefix_length":     "Maximum prefix length for the VPC network range",
	"min_prefix_length":     "Minimum prefix length for the VPC network range",
	"nameservers":           "List of nameservers for the VPC network range",
	"prefix":                "The IP prefix for the VPC network range",
	"labels":                "Labels are key-value string pairs which can be attached to a resource container",
}
