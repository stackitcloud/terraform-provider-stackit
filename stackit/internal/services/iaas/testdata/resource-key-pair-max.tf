variable "name" {}
variable "public_key" {}
variable "label" {}

resource "stackit_key_pair" "key_pair" {
  name       = var.name
  public_key = var.public_key
  labels = {
    "acc-test" : var.label
  }
}