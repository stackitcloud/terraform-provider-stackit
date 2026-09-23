# Only use the import statement, if you want to import an existing volume automation resource
import {
  to = stackit_volume_automation.import-example
  id = "${var.project_id},${var.region},${var.automation_id}"
}
