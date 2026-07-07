# Only use the import statement, if you want to import an existing server backup schedule
import {
  to = stackit_server_backup_schedule.import-example
  id = "${var.project_id},${var.region},${var.server_id},${var.server_backup_schedule_id}"
}
