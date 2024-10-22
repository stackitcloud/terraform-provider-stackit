resource "stackit_volume" "example" {
  project_id        = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name              = "my_volume"
  availability_zone = "eu01-1"
  size              = 64
  labels = {
    "key" = "value"
  }
}
