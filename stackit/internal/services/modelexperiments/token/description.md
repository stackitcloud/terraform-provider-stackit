AI Model Experiment Instance Token Resource schema.

## Example Usage

```terraform

resource "stackit_modelexperiments_instance" "example" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name         = "Example instance"
  region       = "eu01"
  description = "Example description"
}

resource "stackit_modelexperiments_token" "token" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name         = "Example token"
  region       = "eu01"
  instance_id = stackit_modelexperiments_instance.example.instance_id
  description =  "Example description"
}
```