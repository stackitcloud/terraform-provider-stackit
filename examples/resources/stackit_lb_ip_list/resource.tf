resource "stackit_lb_ip_list" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region     = "eu01"
  name       = "example-ip-list"
  labels = {
    "exampleLabelKey1" = "exampleLabelValue1"
    "exampleLabelKey2" = "exampleLabelValue2"
  }
  # file_content is write-only - it is never stored in state and never returned by the API.
  # To rotate the content, update this value AND increment file_content_version.
  # Changing file_content alone will NOT trigger an update.
  file_content         = <<-EOT
    192.0.2.0/24
    198.51.100.0/24
    203.0.113.0/24
  EOT
  file_content_version = 1
}
