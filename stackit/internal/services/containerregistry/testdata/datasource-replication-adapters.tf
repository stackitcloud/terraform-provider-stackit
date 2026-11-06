variable "project_id" {
  type = string
}

data "stackit_containerregistry_replication_adapters" "this" {
  project_id = var.project_id
}