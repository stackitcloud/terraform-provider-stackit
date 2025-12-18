resource "stackit_sfs_share" "example" {
  project_id                 = "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
  resource_pool_id           = "YYYYYYYY-YYYY-YYYY-YYYY-YYYYYYYYYYYY"
  name                       = "my-nfs-share"
  export_policy              = "high-performance-class"
  space_hard_limit_gigabytes = 32
}
