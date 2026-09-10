data "stackit_image_v2" "default" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  image_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

data "stackit_image_v2" "name_match" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "Ubuntu 22.04"
  filter = {
    architecture = "x86"
  }
}

data "stackit_image_v2" "name_regex_latest" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name_regex = "^Ubuntu .*"
  filter = {
    architecture = "x86"
  }
}

data "stackit_image_v2" "name_regex_oldest" {
  project_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name_regex     = "^Ubuntu .*"
  sort_ascending = true
  filter = {
    architecture = "x86"
  }
}

data "stackit_image_v2" "filter_distro_version" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  filter = {
    distro       = "debian"
    version      = "11"
    architecture = "x86"
  }
}

data "stackit_image_v2" "filter_architecture_x86" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  filter = {
    distro       = "ubuntu"
    architecture = "x86"
  }
}

data "stackit_image_v2" "filter_architecture_arm64" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  filter = {
    distro       = "ubuntu"
    architecture = "arm64"
  }
}