resource "stackit_volume_automation" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  template_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name        = "example-volume-automation"
  description = "Creates daily volume snapshots and keeps the last 7."
  input = {
    volume_recovery_point_management = {
      inherit_volume_labels = true
      recovery_point_labels = {
        "created-by" = "terraform"
      }
      volume_label_selector = "backup=daily"
      snapshot_retention_policy = {
        kind  = "count"
        value = 4
      }
    }
  }
  triggers = {
    schedule = {
      rrule = "DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"
    }
  }
}
