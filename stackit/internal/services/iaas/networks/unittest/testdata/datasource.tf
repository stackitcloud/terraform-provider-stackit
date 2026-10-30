variable "project_id" {}

data "stackit_networks" "networks" {
  project_id = var.project_id
}
