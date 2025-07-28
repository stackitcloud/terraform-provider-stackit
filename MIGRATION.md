# Migration Guide

In this guide we want to offer some strategy for a migration of configurations or existing resources to the STACKIT official provider. In relation to the [Terraform community provider](https://github.com/SchwarzIT/terraform-provider-stackit), some attribute names and structure have changed, as well as the internal resource ID structure.

To import your existing infrastructure resources to the new provider, you'll need the internal ID of each resource. The structure of the new provider's internal ID can be located in the [documentation](./docs/resources) file for each resource, specifically within the description of the `id` attribute.

## How-to

Before you begin the migration process, please ensure that you have done the necessary steps for the [authentication](./README.md#authentication).

For existing resources created with the old provider, you'll need to import them into your new configuration. Terraform provides a feature for importing existing resources and auto-generating new Terraform configuration files. To generate configuration code for the imported resources, refer to the official [Terraform documentation](https://developer.hashicorp.com/terraform/language/import/generating-configuration) for step-by-step guidance.

Once the configuration is generated, compare the generated file with your existing configuration. Be aware that field names may have changed so you should consider that when comparing. However, possibly not all attributes from the generated configuration will be needed for managing the infrastructure, as the generator seems to create configurations containing read-only (computed-only) attributes (that make the `terraform apply` fail). Check the Terraform plan for the imported resource to identify any differences.

If you encounter any other issues or have additional questions, don't hesitate to raise them by [opening an issue](https://github.com/stackitcloud/terraform-provider-stackit/issues/new/choose).

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
  allow_privileged_containers     = null
  extensions                      = null
  hibernations                    = null
  kubernetes_version_min          = "1.25"
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

### Example (LogMe service)

Import configuration:

```terraform
import {
  id = "project_id,instance_id"
  to = stackit_logme_instance.example-instance
}

import {
  id = "project_id,instance_id,credential_id"
  to = stackit_logme_credential.example-credential
}
```

Generated configuration:

```terraform
# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform
resource "stackit_logme_instance" "example-instance" {
  name = "example-instance"
  parameters = {
    sgw_acl = "0.0.0.0/0"
  }
  plan_name  = null
  project_id = "project_id"
  version    = null
}

# __generated__ by Terraform from "project_id,instance_id,credential_id"
resource "stackit_logme_credential" "example-credential" {
  instance_id = "instance_id"
  project_id  = "project_id"
}
```

## Available resources

| Community provider                       | Official provider                       | Import available? | `id` format | Notes                                                            |
|------------------------------------------|-----------------------------------------|-|-|------------------------------------------------------------------|
| stackit_argus_credential | stackit_observability_credential | :x: |  | Service deprecated, use stackit_observability_credential instead |
| stackit_argus_instance | stackit_observability_instance | :white_check_mark: | [project_id],[instance_id] | Service deprecated, use stackit_observability_instance instead   |
| stackit_argus_job | stackit_observability_scrapeconfig | :white_check_mark: | [project_id],[instance_id],[name] | Service deprecated, use stackit_observability_scrapeconfig instead        |
| stackit_elasticsearch_credential         |                                         | | | Service deprecated                                               |
| stackit_elasticsearch_instance           |                                         | | | Service deprecated                                               |
| stackit_kubernetes_cluster               | stackit_ske_cluster                     | :white_check_mark: | [project_id],[name] |                                                                  |
| stackit_kubernetes_project               | stackit_ske_project                     | :white_check_mark: | [project_id] |                                                                  |
| stackit_load_balancer                    | stackit_loadbalancer                    | :white_check_mark: | [project_id],[name] |                                                                  |
| stackit_logme_credential                 | stackit_logme_credential                | :white_check_mark: | [project_id],[instance_id],[credential_id] |                                                                  |
| stackit_logme_instance                   | stackit_logme_instance                  | :white_check_mark: | [project_id],[instance_id] |                                                                  |
| stackit_mariadb_credential               | stackit_mariadb_credential              | :white_check_mark: | [project_id],[instance_id],[credential_id] |                                                                  |
| stackit_mariadb_instance                 | stackit_mariadb_instance                | :white_check_mark: | [project_id],[instance_id] |                                                                  |
| stackit_mongodb_flex_instance            | stackit_mongodbflex_instance            | :white_check_mark: | [project_id],[instance_id] |                                                                  |
| stackit_mongodb_flex_user                | stackit_mongodbflex_user                | :warning: | [project_id],[instance_id],[user_id] | `password` field will be empty                                   |
| stackit_object_storage_bucket            | stackit_objectstorage_bucket            | :white_check_mark: | [project_id],[name] |                                                                  |
| stackit_object_storage_credential        | stackit_objectstorage_credential        | :white_check_mark: | [project_id],[credentials_group_id],[credential_id] |                                                                  |
| stackit_object_storage_credentials_group | stackit_objectstorage_credentials_group | :white_check_mark: | [project_id],[credentials_group_id] |                                                                  |
| stackit_object_storage_project           |                                         | | | Resource deprecated                                              |
| stackit_observability_credential         | stackit_observability_credential        | :x: | |                                                                  |
| stackit_observability_instance           | stackit_observability_instance          | :white_check_mark: | [project_id],[instance_id] |                                                                  |
| stackit_observability_job                | stackit_observability_scrapeconfig      | :white_check_mark: | [project_id],[instance_id],[name] |                                                                  |
| stackit_opensearch_credential            | stackit_opensearch_credential           | :white_check_mark: | [project_id],[credentials_group_id],[credential_id] |                                                                  |
| stackit_opensearch_instance              | stackit_opensearch_instance             | :white_check_mark: | [project_id],[instance_id] |                                                                  |                                 |
| stackit_postgres_flex_instance           | stackit_postgresflex_instance           | :white_check_mark: | [project_id],[instance_id] |                                                                  |
| stackit_postgres_flex_user               | stackit_postgresflex_user               | :warning: | [project_id],[instance_id],[user_id] | `password` field will be empty                                   |
| stackit_postgres_instance                |                                          | | | Resource deprecated                                            |
| stackit_postgres_credential              |                                         | | | Resource deprecated           
| stackit_project                          | stackit_resourcemanager_project         | :white_check_mark: | [container_id] |                                                                  |
| stackit_rabbitmq_credential              | stackit_rabbitmq_credential             | :white_check_mark: | [project_id],[credentials_group_id],[credential_id] |                                                                  |
| stackit_rabbitmq_instance                | stackit_rabbitmq_instance               | :white_check_mark: | [project_id],[instance_id] |                                                                  |
| stackit_redis_credential                 | stackit_redis_credential                | :white_check_mark: | [project_id],[credentials_group_id],[credential_id] |                                                                  |
| stackit_redis_instance                   | stackit_redis_instance                  | :white_check_mark: | [project_id],[instance_id] |                                                                  |
| stackit_secrets_manager_instance         | stackit_secretsmanager_instance         | :white_check_mark: | [project_id],[instance_id] |                                                                  |
| stackit_secrets_manager_user             | stackit_secretsmanager_user             | :warning: | [project_id],[instance_id],[user_id] | `password` field will be empty                                   |
