resource "stackit_sfs_export_policy" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example"
  rules = [
    {
      ip_acl = ["172.16.0.0/24", "172.16.0.250/32"]
      order  = 1
    }
  ]
}
