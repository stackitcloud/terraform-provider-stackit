variable "project_id" {}

data "stackit_image_v2" "name_match_ubuntu_22_04" {
  project_id = var.project_id
  name       = "Ubuntu 22.04"
}

data "stackit_image_v2" "ubuntu_by_image_id" {
  project_id = var.project_id
  image_id   = data.stackit_image_v2.name_match_ubuntu_22_04.image_id
}

data "stackit_image_v2" "regex_match_ubuntu_22_04" {
  project_id = var.project_id
  name_regex = "(?i)^ubuntu 22.04$"
}

data "stackit_image_v2" "filter_debian_11" {
  project_id = var.project_id
  filter = {
    distro  = "debian"
    version = "11"
  }
}

data "stackit_image_v2" "filter_uefi_ubuntu" {
  project_id = var.project_id
  filter = {
    distro = "ubuntu"
    uefi   = true
  }
}

data "stackit_image_v2" "name_regex_and_filter_rhel_9_1" {
  project_id = var.project_id
  name_regex = "^Red Hat Enterprise Linux 9.1$"
  filter = {
    distro  = "rhel"
    version = "9.1"
    uefi    = true
  }
}

data "stackit_image_v2" "name_windows_2022_standard" {
  project_id = var.project_id
  name       = "Windows Server 2022 Standard"
}

data "stackit_image_v2" "ubuntu_arm64_latest" {
  project_id = var.project_id
  filter = {
    distro = "ubuntu-arm64"
  }
}

data "stackit_image_v2" "ubuntu_arm64_oldest" {
  project_id = var.project_id
  filter = {
    distro = "ubuntu-arm64"
  }
  sort_ascending = true
}