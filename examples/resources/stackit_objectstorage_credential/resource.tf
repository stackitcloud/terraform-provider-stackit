resource "stackit_objectstorage_credential" "example" {
  project_id           = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  credentials_group_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  expiration_timestamp = "2027-01-02T03:04:05Z"
}

resource "time_rotating" "rotate" {
  rotation_days = 80
}

resource "stackit_objectstorage_credential" "rotate_example" {
  project_id           = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  credentials_group_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  expiration_timestamp = "2027-01-02T03:04:05Z"

  rotate_when_changed = {
    rotation = time_rotating.rotate.id
  }
}
