variable "project_id" {}

data "stackit_machine_types" "all" {
  project_id = var.project_id
}

data "stackit_machine_types" "empty_filter" {
  project_id = var.project_id
  filter     = {}
}

data "stackit_machine_types" "hardware" {
  project_id = var.project_id
  filter = {
    vcpu = data.stackit_machine_types.all.results[0].vcpus
    ram  = data.stackit_machine_types.all.results[0].ram
  }
}

data "stackit_machine_types" "exact" {
  project_id = var.project_id
  filter = {
    name        = data.stackit_machine_types.all.results[0].name
    disk        = data.stackit_machine_types.all.results[0].disk
    vcpu        = data.stackit_machine_types.all.results[0].vcpus
    ram         = data.stackit_machine_types.all.results[0].ram
    extra_specs = data.stackit_machine_types.all.results[0].extra_specs
  }
}

data "stackit_machine_types" "no_match" {
  project_id = var.project_id
  filter = {
    name = "terraform-acc-no-such-machine-type-866"
  }
}

output "machine_types_sorted" {
  value = tolist([for mt in data.stackit_machine_types.all.results : mt.name]) == sort([for mt in data.stackit_machine_types.all.results : mt.name])
}

output "machine_types_empty_filter_matches_all" {
  value = data.stackit_machine_types.empty_filter.results == data.stackit_machine_types.all.results
}

output "machine_types_hardware_matches" {
  value = length(data.stackit_machine_types.hardware.results) > 0 && alltrue([
    for mt in data.stackit_machine_types.hardware.results :
    mt.vcpus == data.stackit_machine_types.all.results[0].vcpus && mt.ram == data.stackit_machine_types.all.results[0].ram
  ])
}

output "machine_types_exact_matches" {
  value = data.stackit_machine_types.exact.results == tolist([data.stackit_machine_types.all.results[0]])
}
