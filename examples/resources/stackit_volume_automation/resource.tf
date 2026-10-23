resource "stackit_volume_automation" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  template_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name        = "example-volume-automation"
  description = "Creates daily volume snapshots and keeps the last 7."
  input = jsonencode({
    kind                = "VolumeRecoveryPointManagement"
    inheritVolumeLabels = false
    recoveryPointLabels = {
      "exampleLabelKey1" = "exampleLabelValue1"
      "exampleLabelKey2" = "exampleLabelValue2"
    }
    snapshotRetentionPolicy = {
      kind  = "count"
      value = 2
    }
    volumeLabelSelector = "myLabelkey1=myLabelValue,myLabelKey2=myOtherLabelValue"
  })
  triggers = {
    schedule = {
      rrule = "DTSTART;TZID=Europe/Sofia:20200803T023000 RRULE:FREQ=DAILY;INTERVAL=1"
    }
  }
}
