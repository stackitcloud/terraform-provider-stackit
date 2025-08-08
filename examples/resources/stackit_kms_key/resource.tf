resource "stackit_kms_key" "name" {
  algorithm = "example algorithm"
  backend = "software"
  description = "new descr"
  display_name = "example name"
  import_only = false
  key_ring_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  purpose = "example purpose"
}