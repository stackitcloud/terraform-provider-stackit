locals {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  display_name = "edge"
  plan_id      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  description  = "cats live on the edge"
  region       = "eu01"
}

resource "stackit_edgecloud_instance" "this" {
  project_id   = local.project_id
  display_name = local.display_name
  plan_id      = local.plan_id
  description  = local.description
}

# Only use the import statement, if you want to import an existing Edge Cloud instance resource
import {
  to = stackit_edgecloud_instance.this
  id = "${local.project_id},${local.region},INSTANCE_ID"
}
