resource "stackit_server_update_schedule" "example" {
  project_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  server_id          = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name               = "example_update_schedule_name"
  rrule              = "DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"
  enabled            = true
  maintenance_window = 1
}

# Only use the import statement, if you want to import an existing server update schedule
import {
  to = stackit_server_update_schedule.import-example
  id = "${var.project_id},${var.region},${var.server_id},${var.server_update_schedule_id}"
}