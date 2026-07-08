resource "stackit_telemetryrouter_instance" "router" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region       = "eu01"
  display_name = "router-instance"
}

resource "stackit_telemetryrouter_instance" "router2" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region       = "eu01"
  display_name = "router-instance"
  description  = "Example description"
  filter = {
    attributes = [
      {
        key     = "key"
        level   = "logRecord"
        matcher = "!="
        values  = ["test1", "test2"]
      },
      {
        key     = "key2"
        level   = "resource"
        matcher = "="
        values  = ["test3"]
      }
    ]
  }
}
