resource "stackit_vpc" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name        = "example"
  description = "Example description"
  labels = {
    "key" = "value"
  }
}
