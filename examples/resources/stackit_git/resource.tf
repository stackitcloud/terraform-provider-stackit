resource "stackit_git" "git" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "git-example-instance"
}

resource "stackit_git" "git" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "git-example-instance"
  acl = [
    "0.0.0.0/0"
  ]
  flavor = "git-100"
}