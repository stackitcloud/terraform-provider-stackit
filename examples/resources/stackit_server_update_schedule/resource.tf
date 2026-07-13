resource "stackit_server_update_schedule" "example" {
  project_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  server_id          = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name               = "example_update_schedule_name"
  rrule              = "DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"
  enabled            = true
  maintenance_window = 1
  depends_on = [
    stackit_server_update_enable.enable
  ]
}

resource "stackit_server_update_enable" "enable" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  server_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}