# Only use the import statement, if you want to import an existing dns record set
import {
  to = stackit_dns_record_set.import-example
  id = "${var.project_id},${var.zone_id},${var.record_set_id}"
}
