resource "stackit_security_group" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "my_security_group"
  labels = {
    "key" = "value"
  }
}
