resource "stackit_logs_access_token" "accessToken" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region       = "eu01"
  display_name = "logs-access-token-example"
  permissions = [
    "read"
  ]
}

resource "stackit_logs_access_token" "accessToken2" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region       = "eu01"
  display_name = "logs-access-token-example"
  lifetime     = 30
  permissions = [
    "write"
  ]
  description = "Example description"
}

resource "time_rotating" "rotate" {
  rotation_days = 10
}

resource "stackit_logs_access_token" "accessToken_rotate" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region       = "eu01"
  display_name = "logs-access-token-example"
  lifetime     = 30
  permissions = [
    "write"
  ]
  description = "Example description"

  rotate_when_changed = {
    rotation = time_rotating.rotate.id
  }
}