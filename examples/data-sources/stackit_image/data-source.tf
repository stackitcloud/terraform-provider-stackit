data "stackit_image" "default" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  image_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

data "stackit_image" "name_match" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "Ubuntu 22.04"
}

data "stackit_image" "name_regex_latest" {
  project_id      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name_regex      = "^Ubuntu .*"
  sort_descending = true
}

data "stackit_image" "name_regex_oldest" {
  project_id      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name_regex      = "^Ubuntu .*"
  sort_descending = false
}

data "stackit_image" "filter_distro_version" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  filter = {
    distro  = "debian"
    version = "11"
  }
}