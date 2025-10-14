variable "project_id" {}

data "stackit_machine_type" "two_vcpus_filter" {
  project_id = var.project_id
  filter     = "vcpus==2"
}

data "stackit_machine_type" "filter_sorted_ascending_false" {
  project_id     = var.project_id
  filter         = "vcpus >= 2 && ram >= 2048"
  sort_ascending = false
}

# returns warning
data "stackit_machine_type" "no_match" {
  project_id = var.project_id
  filter     = "vcpus == 99"
}