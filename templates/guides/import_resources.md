---
page_title: "How to import an existing resources"
---
# How to import an existing resources?

## 1. **Create a terraform config file and add an import block for your resource**

In order to import an existing resources in terraform you need to add an import block for the corresponding resource in a terraform config file.
There is an example for every resource under the [examples](../../examples/) folder.

E.g. the import statement for a `stackit_volume` looks like the following:

```terraform
import {
  to = stackit_volume.import-example
  id = "${var.project_id},${var.volume_id}"
}
```

## 2. **Generate the destination resource automatically**

Run `terraform plan -generate-config-out=generated.tf` to let terraform generate the configuration for you.
In this step the `stackit_volume.import-example` resource is generated and filled with informations of your existing resource.

## 3. **Finish the import**

Run `terraform apply` to add your resource to the terraform state.