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
