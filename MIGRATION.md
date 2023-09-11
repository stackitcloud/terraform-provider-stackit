# Migration from the Terraform community provider

In this guide, we want to offer some strategy for a migration of configurations from the Terraform community provider to this new official provider. In this provider, some attribute names and structure have changed, as well as the internal resource ID structure.

To import your existing infrastructure resources to the new provider, you'll need the internal ID of each resource. The structure of the new provider's internal ID can be located in the [documentation](./docs/resources) file for each resource, specifically within the description of the `id` attribute.

## How-to
Before you begin the migration process, please ensure that you have done the necessary steps for the [authentication](./README.md#authentication).

For existing resources created with the old provider, you'll need to import them into your new configuration. Terraform provides a feature for importing existing resources and auto-generating new Terraform configuration files. To generate configuration code for the imported resources, refer to the official [Terraform documentation](https://developer.hashicorp.com/terraform/language/import/generating-configuration) for step-by-step guidance.

Once the configuration is generated, compare the generated file with your existing configuration. Be aware that field names may have changed so you should adapt the configuration accordingly. However, not all attributes from the generated configuration are needed for managing the infrastructure, meaning this set of fields can be reduced to the relevant ones from your previous configuration. Check the Terraform plan for the imported resource to identify any differences.

### Example (SKE service)
Import configuration:
```terraform
# Import
import {
  id = "project_id"
  to = stackit_ske_project.project-example
}

import {
  id = "project_id,example-cluster"
  to = stackit_ske_cluster.cluster-example
}
```

Generated configuration:
```terraform
# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "project_id"
resource "stackit_ske_project" "project-example" {
  project_id = "project_id"
}

# __generated__ by Terraform from "project_id,example-cluster"
resource "stackit_ske_cluster" "cluster-example" {
  allow_privileged_containers = null
  extensions                  = null
  hibernations                = null
  kubernetes_version          = "1.25"
  maintenance = {
    enable_kubernetes_version_updates    = true
    enable_machine_image_version_updates = true
    end                                  = "09:47:00Z"
    start                                = "08:47:00Z"
  }
  name = "example-cluster"
  node_pools = [
    {
      availability_zones = ["region-a", "region-b"]
      cri                = "containerd"
      labels = {
        l1 = "value1"
        l2 = "value2"
      }
      machine_type    = "b1.2"
      max_surge       = 1
      max_unavailable = 1
      maximum         = 10
      minimum         = 3
      name            = "example-np"
      os_name         = "flatcar"
      os_version      = "3510.2.3"
      taints = [
        {
          effect = "PreferNoSchedule"
          key    = "tk"
          value  = "tkv"
        },
      ]
      volume_size = 40
      volume_type = "example-type"
    },
  ]
  project_id = "project_id"
}

```