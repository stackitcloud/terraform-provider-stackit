# Create a key pair
resource "stackit_key_pair" "keypair" {
  name       = "example-key-pair"
  public_key = chomp(file("path/to/id_rsa.pub"))
}

# Only use the import statement, if you want to import an existing key pair
import {
  to = stackit_key_pair.import-example
  id = var.keypair_name
}