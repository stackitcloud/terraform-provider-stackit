variable "name" {}
variable "public_key" {}

resource "stackit_key_pair" "key_pair" {
  name = var.name
  public_key = var.public_key
}