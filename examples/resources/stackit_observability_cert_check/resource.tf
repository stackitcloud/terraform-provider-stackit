# e.g. x509_cert_expiry metric will arrive in the observability stack
resource "stackit_observability_http_check" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  url         = "tcp://stackit.de:443"
}

# Only use the import statement, if you want to import an existing cert-check
import {
  to = stackit_observability_http_check.example
  id = "${var.project_id},${var.observability_instance_id},${var.cert_check_id}"
}
