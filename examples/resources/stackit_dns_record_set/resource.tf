resource "stackit_dns_record_set" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  zone_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-record-set"
  type       = "A"
  comment    = "Example comment"
  records    = ["1.2.3.4"]
}

# Only use the import statement, if you want to import an existing dns record set
import {
  to = stackit_dns_record_set.import-example
  id = "${var.project_id},${var.zone_id},${var.record_set_id}"
}