# List all available machine types in the provider region.
data "stackit_machine_types" "all" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

# Match exactly 1 vCPU and 1024 MB of RAM. All filters are combined with AND.
data "stackit_machine_types" "small" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  filter = {
    vcpu = 1
    ram  = 1024
  }
}

# Filter by extra specs in an explicitly selected region.
data "stackit_machine_types" "intel" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region     = "eu01"
  filter = {
    vcpu = 2
    extra_specs = {
      cpu = "intel-icelake-generic"
    }
  }
}

output "small_machine_types" {
  value = data.stackit_machine_types.small.results
}

# Use HCL for more complex filtering, such as a minimum RAM size.
output "high_memory_machine_types" {
  value = [
    for machine_type in data.stackit_machine_types.all.results : machine_type
    if machine_type.ram >= 4096
  ]
}
