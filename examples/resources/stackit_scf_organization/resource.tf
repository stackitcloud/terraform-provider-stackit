resource "stackit_scf_organization" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example"
}

resource "stackit_scf_organization" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name        = "example"
  platform_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  quota_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  suspended   = false
}
