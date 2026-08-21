data "stackit_postgresflex_flavors" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  # region is taken from the provider configuration
}

# Example usage with an instance
resource "stackit_postgresflex_instance" "example" {
  project_id      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name            = "example"
  flavor_id       = one([for flavor in data.stackit_postgresflex_flavors.example.flavors : flavor.id if flavor.cpu == 2 && flavor.memory == 4 && flavor.node_type == "Single"])
  backup_schedule = "0 16 * * *"
  storage = {
    class = "premium-perf2-stackit"
    size  = 5
  }
  version = "17"
  network = {
    acl = ["192.168.0.0/24"]
  }
}
