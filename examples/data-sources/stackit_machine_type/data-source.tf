data "stackit_machine_type" "two_vcpus_filter" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  filter     = "vcpus==2"
}

data "stackit_machine_type" "filter_sorted_ascending_false" {
  project_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  filter         = "vcpus >= 2 && ram >= 2048"
  sort_ascending = false
}

data "stackit_machine_type" "intel_icelake_generic_filter" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  filter     = "extraSpecs.cpu==\"intel-icelake-generic\" && vcpus == 2"
}

# returns warning
data "stackit_machine_type" "no_match" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  filter     = "vcpus == 99"
}