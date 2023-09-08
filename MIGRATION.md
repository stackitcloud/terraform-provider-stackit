# Migration from the Terraform community provider

In this guide, we want to offer some strategy for a migration of configurations from the Terraform community provider to this new official provider. In this provider, some attribute names and structure have changed, as well as the internal resource ID structure.

To import your existing infrastructure resources to the new provider, you'll need the internal ID of each resource. The structure of the new provider's internal ID can be located in the [documentation](./docs/resources) file for each resource, specifically within the description of the `id` attribute.

## How-to
Before you begin the migration process, please ensure that you have done the necessary steps for the [authentication](./README.md#authentication).

For existing resources created with the old provider, you'll need to import them into your new configuration. Terraform provides a feature for importing existing resources and auto-generating new Terraform configuration files. To generate configuration code for the imported resources, refer to the official [Terraform documentation](https://developer.hashicorp.com/terraform/language/import/generating-configuration) for step-by-step guidance.

Once the configuration is generated, compare the generated file with your existing configuration. Be aware that field names may have changed so you should adapt the configuration accordingly. However, not all attributes from the generated configuration are needed for managing the infrastructure, meaning this set of fields can be reduced to the relevant ones from your previous configuration. Check the Terraform plan for the imported resource to identify any differences.
