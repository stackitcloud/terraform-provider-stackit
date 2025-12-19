# Copyright (c) STACKIT

data "stackitprivatepreview_postgresflexalpha_user" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  user_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
