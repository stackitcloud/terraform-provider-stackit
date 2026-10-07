variable "project_id" {
  type = string
}

data "stackit_images" "all" {
  project_id = var.project_id
}

data "stackit_images" "filtered" {
  project_id = var.project_id
  filter = {
    label_selector = "linux="
  }
  timeouts = { read = "5m" }
}

output "image_ids" {
  value = [for image in data.stackit_images.all.results : image.image_id]
}

# These expressions filter locally in Terraform; they are not API filters.
locals {
  exact_name     = [for image in data.stackit_images.all.results : image if image.name == "Ubuntu 22.04"]
  matching_names = [for image in data.stackit_images.all.results : image if can(regex("Ubuntu.*", image.name))]
  ubuntu_images  = [for image in data.stackit_images.all.results : image if try(image.config.operating_system_distro, "") == "ubuntu"]
}
