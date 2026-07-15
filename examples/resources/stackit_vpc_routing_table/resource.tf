resource "stackit_vpc_region" "example-region" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  vpc_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

resource "stackit_vpc_routing_table" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  vpc_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example"
  labels = {
    "key" = "value"
  }
  # a routing table needs a configured region
  depends_on = [stackit_vpc_region.example-region]
}
