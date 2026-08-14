package vpcregion

import (
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
)

const resourceDescription = "VPC region resource schema"
const datasourceDescription = "VPC region datasource schema"

var descriptions = map[string]string{
	"resource.markdown":   features.AddExperimentDescription(resourceDescription, features.VpcExperiment, core.Resource),
	"datasource.markdown": features.AddExperimentDescription(datasourceDescription, features.VpcExperiment, core.Datasource),
	"id":                  "Terraform's internal resource ID. It is structured as \"`project_id`, `vpc_id`, `region`\".",
	"project_id":          "STACKIT project ID to which the VPC region is associated.",
	"vpc_id":              "The VPC ID to which the VPC region is associated.",
	"region":              "The resource region. If not defined, the provider region is used.",
}
