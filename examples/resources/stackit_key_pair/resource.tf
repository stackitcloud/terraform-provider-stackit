# Create a key pair
resource "stackit_key_pair" "keypair" {
  name       = "example-key-pair"
  public_key = chomp(file("path/to/id_rsa.pub"))
}