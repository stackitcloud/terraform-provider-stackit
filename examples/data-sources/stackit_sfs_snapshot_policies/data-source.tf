data "stackit_sfs_snapshot_policies" "all" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

data "stackit_sfs_snapshot_policies" "immutable_only" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  immutable  = "immutable-only"
}