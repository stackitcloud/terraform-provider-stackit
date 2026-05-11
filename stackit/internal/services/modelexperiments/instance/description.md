AI Model Experiment Instance Resource schema.

## Example Usage

```terraform

resource "stackit_modelexperiments_instance" "example" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name         = "Example instance"
  region       = "eu01"
  description = "Example description"
}
```