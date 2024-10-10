resource "stackit_public_ip" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  labels = {
    "key" = "value"
  }
}
