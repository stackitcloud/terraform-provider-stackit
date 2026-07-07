# Only use the import statement, if you want to import an existing cdn distribution
import {
  to = stackit_cdn_distribution.import-example
  id = "${var.project_id},${var.distribution_id}"
}
