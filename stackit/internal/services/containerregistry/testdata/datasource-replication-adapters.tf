variable "project_id" {}

data "stackit_containerregistry_replication_adapters" "this" {
  project_id = var.project_id
}
