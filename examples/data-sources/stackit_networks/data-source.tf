# List all networks in a project
data "stackit_networks" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

# List all networks in a project with a specific name and label
data "stackit_networks" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name_regex = "example"
  labels = {
    foo = "bar"
  }
}
