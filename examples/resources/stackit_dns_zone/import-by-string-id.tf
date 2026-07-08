# Only use the import statement, if you want to import an existing dns zone
import {
  to = stackit_dns_zone.import-example
  id = "${var.project_id},${var.zone_id}"
}
