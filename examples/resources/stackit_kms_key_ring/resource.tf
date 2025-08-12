resource "stackit_kms_key_ring" "example" {
  description  = "example description"
  display_name = "example name"
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region_id    = "eu01"
}
