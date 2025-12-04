# e.g. http_response_result_code metric will arrive in the observability stack
resource "stackit_observability_http_check" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  url         = "https://www.stackit.de"
}

# Only use the import statement, if you want to import an existing http-check
import {
  to = stackit_observability_http_check.example
  id = "${var.project_id},${var.observability_instance_id},${var.http_check_id}"
}
