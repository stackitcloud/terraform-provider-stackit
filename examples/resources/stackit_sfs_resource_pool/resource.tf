resource "stackit_sfs_resource_pool" "resourcepool" {
  project_id        = "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
  name              = "some-resourcepool"
  availability_zone = "eu01-m"
  performance_class = "Standard"
  size_gigabytes    = 512
  ip_acl = [
    "192.168.42.1/32",
    "192.168.42.2/32"
  ]
  snapshots_are_visible = true
}
