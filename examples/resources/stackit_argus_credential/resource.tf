resource "stackit_argus_credential" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

### Example move

#	Example to move the deprecated `stackit_argus_credential` resource to the new `stackit_observability_credential` resource:
#	1. Add a new `stackit_observability_credential` resource with the same values like your previous `stackit_argus_credential` resource.
#	2. Add a moved block which reference the `stackit_argus_credential` and `stackit_observability_credential` resource.
#	3. Remove your old `stackit_argus_credential` resource and run `$ terraform apply`.

resource "stackit_argus_credential" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

moved {
  from = stackit_argus_credential.example
  to   = stackit_observability_credential.example
}

resource "stackit_observability_credential" "example" {
  project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
