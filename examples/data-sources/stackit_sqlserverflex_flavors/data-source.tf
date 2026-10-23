data "stackit_sqlserverflex_flavors" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  // region taken from provider
}

// example usage with instance
resource "stackit_sqlserverflex_instance" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example"
  flavor_id  = one([for f in data.stackit_sqlserverflex_flavors.example.flavors : f.id if f.cpu == 4 && f.memory == 16])
}

