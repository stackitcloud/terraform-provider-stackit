# see other files

data "stackitprivatepreview_sqlserverflexalpha_instance" "existing" {
  project_id = var.project_id
  instance_id = "b31575e9-9dbd-4ff6-b341-82d89c34f14f"
  region = "eu01"
}

output "myinstance" {
  value = data.stackitprivatepreview_sqlserverflexalpha_instance.existing
}
