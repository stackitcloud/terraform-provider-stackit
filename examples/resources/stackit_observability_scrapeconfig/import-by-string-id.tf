# Only use the import statement, if you want to import an existing observability scrapeconfig
import {
  to = stackit_observability_scrapeconfig.import-example
  id = "${var.project_id},${var.observability_instance_id},${var.observability_scrapeconfig_name}"
}
