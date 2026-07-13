resource "stackit_affinity_group" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-affinity-group-name"
  policy     = "hard-anti-affinity"
}