resource "stackit_image" "example_image" {
  project_id      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name            = "example-image"
  disk_format     = "qcow2"
  local_file_path = "./path/to/image.qcow2"
  min_disk_size   = 10
  min_ram         = 5
}
