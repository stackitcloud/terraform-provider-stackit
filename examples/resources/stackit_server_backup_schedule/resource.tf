resource "stackit_server_backup_schedule" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  server_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example_backup_schedule_name"
  rrule      = "DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"
  enabled    = true
  backup_properties = {
    name             = "example_backup_name"
    retention_period = 14
    volume_ids       = null
  }
}
