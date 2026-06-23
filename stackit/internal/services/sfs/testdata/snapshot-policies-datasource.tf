variable "project_id" {}
variable "immutable" {
  default = "all"
}

data "stackit_sfs_snapshot_policies" "snapshot_policies" {
  project_id = var.project_id
  immutable  = var.immutable
}