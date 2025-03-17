# Contribute to the STACKIT Terraform Provider

Your contribution is welcome! Thank you for your interest in contributing to the STACKIT Terraform Provider. We greatly value your feedback, feature requests, additions to the code, bug reports or documentation extensions.

## Table of contents

- [Developer Guide](#developer-guide)
  - [Useful Make commands](#useful-make-commands)
  - [Repository structure](#repository-structure)
  - [Implementing a new resource](#implementing-a-new-resource)
  	- [Resource file structure](#resource-file-structure)
  - [Implementing a new datasource](#implementing-a-new-datasource)
  - [Onboarding a new STACKIT service](#onboarding-a-new-stackit-service)
  - [Local development](#local-development)
  	- [Setup centralized Terraform state](#setup-centralized-terraform-state)
- [Code Contributions](#code-contributions)
- [Bug Reports](#bug-reports)

## Developer Guide

### Useful Make commands

These commands can be executed from the project root:

- `make project-tools`: get the required dependencies
- `make lint`: lint the code and examples
- `make generate-docs`: generate terraform documentation
- `make test`: run unit tests
- `make test-acceptance-tf`: run acceptance tests

### Repository structure

The provider resources and data sources for the STACKIT services are located under `stackit/services`. Inside `stackit` you can find several other useful packages such as `validate` and `testutil`. Examples of usage of the provider are located under the `examples` folder.

We make use of the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework) to write the Terraform provider. [Here](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-provider) you can find a tutorial on how to implement a provider using this framework.

### Implementing a new resource

Let's suppose you want to want to implement a new resource `bar` of service `foo`:

1. You would start by creating a new folder `bar/` inside `stackit/internal/services/foo/`
2. Following with the creation of a file `resource.go` inside your new folder `stackit/internal/services/foo/bar/`
   1. The Go package should be similar to the service name, in this case `foo` would be an adequate package name
   2. Please refer to the [Resource file structure](./CONTRIBUTION.md/#resource-file-structure) section for details on the structure of the file itself
3. To register the new resource `bar` in the provider, add it to the `Resources` in `stackit/provider.go`, using the `NewBarResource` method
4. Add an example in `examples/resources/stackit_foo_bar/resource.tf` with an example configuration for the new resource, e.g.:
   ```tf
    resource "stackit_foo_bar" "example" {
      project_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      my_required_field      = "my-required-field-value"
      my_optional_field      = "my-optional-field-value"
    }
   ```

Please remeber to always add unit tests for the helper functions (in this case `mapFields` and `toCreatePayload`), as well implementing/extending the acceptance (end-to-end) tests. Our acceptance tests are implemented using Hashicorp's [terraform-plugin-testing](https://developer.hashicorp.com/terraform/plugin/testing/acceptance-tests) package.

Additionally, remember to run `make generate-docs` after your changes to keep the commands' documentation in `docs/` updated, which is used as a source for the [Terraform Registry documentation page](https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs).

#### Resource file structure

Below is a typical structure of a STACKIT Terraform provider resource:

```go
package foo

import (
	(...)
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/foo" // Import service "foo" from the STACKIT SDK for Go
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &barResource{}
	_ resource.ResourceWithConfigure   = &barResource{}
	_ resource.ResourceWithImportState = &barResource{}
)

// Provider's internal model
type Model struct {
	Id              types.String `tfsdk:"id"` // needed by TF
	ProjectId       types.String `tfsdk:"project_id"`
	BarId           types.String `tfsdk:"bar_id"`
	MyRequiredField types.String `tfsdk:"my_required_field"`
	MyOptionalField         types.String `tfsdk:"my_optional_field"`
	MyReadOnlyField types.String `tfsdk:"my_read_only_field"`
}

// NewBarResource is a helper function to simplify the provider implementation.
func NewBarResource() resource.Resource {
	return &barResource{}
}

// barResource is the resource implementation.
type barResource struct {
	client *foo.APIClient
}

// Metadata returns the resource type name.
func (r *barResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_foo_bar"
}

// Configure adds the provider configured client to the resource.
func (r *barResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	var apiClient *foo.APIClient
	var err error
	if providerData.FooCustomEndpoint != "" {
		apiClient, err = foo.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.FooCustomEndpoint),
		)
	} else {
		apiClient, err = foo.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithRegion(providerData.DefaultRegion),
		)
	}

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	r.client = apiClient
	tflog.Info(ctx, "Foo bar client configured")
}

// Schema defines the schema for the resource.
func (r *barResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"main":               "Foo bar resource schema.",
		"id":                 "Terraform's internal resource identifier. It is structured as \"`project_id`,`bar_id`\".",
		"project_id":         "STACKIT Project ID to which the bar is associated.",
		"bar_id":             "The bar ID.",
		"my_required_field":  "My required field description.",
		"my_optional_field":  "My optional field description.",
		"my_read_only_field": "My read-only field description.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					// "RequiresReplace" makes the provider recreate the resource when the field is changed in the configuration
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					// Validators can be used to validate the values set to a field
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"bar_id": schema.StringAttribute{
				Description: descriptions["bar_id"],
				Computed:    true,
			},
			"my_required_field": schema.StringAttribute{
				Description: descriptions["my_required_field"],
				Required: true,
				PlanModifiers: []planmodifier.String{
					// "RequiresReplace" makes the provider recreate the resource when the field is changed in the configuration
					stringplanmodifier.RequiresReplace(),
				},
			},
			"my_optional_field": schema.StringAttribute{
				Description: descriptions["my_optional_field"],
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					// "UseStateForUnknown" can be used to copy a prior state value into the planned value. It should be used when it is known that an unconfigured value will remain the same after a resource update
					stringplanmodifier.UseStateForUnknown(),
				}
			},
			"my_read_only_field": schema.StringAttribute{
				Description: descriptions["my_read_only_field"],
				Computed: true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *barResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	barId := model.BarId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)

	// Create new bar
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating credential", fmt.Sprintf("Creating API payload: %v", err))
		return
	}
	resp, err := r.client.CreateBar(ctx, projectId, barId).CreateBarPayload(*payload).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating bar", fmt.Sprintf("Calling API: %v", err))
		return
	}
	ctx = tflog.SetField(ctx, "bar_id", resp.BarId)

	// Map response body to schema
	err = mapFields(resp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating bar", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Foo bar created")
}

// Read refreshes the Terraform state with the latest data.
func (r *barResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	barId := model.BarId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "bar_id", barId)

	barResp, err := r.client.GetBar(ctx, projectId, barId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading bar", fmt.Sprintf("Calling API: %v", err))
		return
	}

	// Map response body to schema
	err = mapFields(barResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading bar", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Foo bar read")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *barResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Similar to Create method, calls r.client.UpdateBar instead
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *barResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectId := model.ProjectId.ValueString()
	barId := model.BarId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "bar_id", barId)

	// Delete existing bar
	_, err := r.client.DeleteBar(ctx, projectId, barId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting bar", fmt.Sprintf("Calling API: %v", err))
	}

	tflog.Info(ctx, "Foo bar deleted")
}

// ImportState imports a resource into the Terraform state on success.
// The expected format of the bar resource import identifier is: project_id,bar_id
func (r *barResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, core.Separator)
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		core.LogAndAddError(ctx, &resp.Diagnostics,
			"Error importing bar",
			fmt.Sprintf("Expected import identifier with format [project_id],[bar_id], got %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bar_id"), idParts[1])...)
	tflog.Info(ctx, "Foo bar state imported")
}

// Maps bar fields to the provider's internal model
func mapFields(barResp *foo.GetBarResponse, model *Model) error {
	if barResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if barResp.Bar == nil {
		return fmt.Errorf("response bar is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}
	bar := barResp.Bar
	model.BarId = types.StringPointerValue(bar.BarId)

	idParts := []string{
		model.ProjectId.ValueString(),
		model.BarId.ValueString(),
	}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	model.MyRequiredField = types.StringPointerValue(bar.MyRequiredField)
	model.MyOptionalField = types.StringPointerValue(bar.MyOptionalField)
	model.MyReadOnlyField = types.StringPointerValue(bar.MyOtherField)
	return nil
}

//  Build CreateBarPayload from provider's model
func toCreatePayload(model *Model) *foo.CreateBarPayload {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}

	myRequiredFieldValue := conversion.StringValueToPointer(model.MyRequiredField)
	myOptionalFieldValue := conversion.StringValueToPointer(model.MyOptionalField)
	return &foo.CreateBarPayload{
		MyRequiredField: myRequiredFieldValue,
		MyOptionalField: myOptionalFieldValue,
	}, nil
}
```

If the new resource `bar` is the first resource in the TFP using a STACKIT service `foo`, please refer to [Onboarding a new STACKIT service](./CONTRIBUTION.md/#onboarding-a-new-stackit-service).

### Implementing a new datasource

The process to implement a new datasource is similar to [implementing a new resource](#implementing-a-new-resource). Some differences worth noting are:

- The datasource schema will have all attributes set as `Computed` only, with the exception of the ones needed to identify the datasource (usually are the same attributes used to compose the `id` field), which will be set as `Required`
- To register the new datasource bar in the provider, it should be added the `DataSources` in `stackit/provider.go`, using the `New...Datasource` method

### Onboarding a new STACKIT service

If you want to onboard resources of a STACKIT service `foo` that was not yet in the provider, you will need to do a few additional steps in `stackit/provider.go`:

1. Add `FooCustomEndpoint` fields to `providerModel` and `ProviderData` types, in `stackit/provider.go` and `stackit/internal/core/core.go`, respectively
2. Add a `foo_custom_endpoint` attribute to the provider's `Schema`, in `stackit/provider.go`
3. Check if the custom endpoint is defined and, if yes, use it. In the `Configure` method, add:
   ```go
   if !(providerConfig.FooCustomEndpoint.IsUnknown() || providerConfig.FooCustomEndpoint.IsNull()) {
     providerData.FooCustomEndpoint = providerConfig.FooCustomEndpoint.ValueString()
   }
   ```

### Local development

To test your changes locally, you have to compile the provider (requires Go 1.22) and configure the Terraform CLI to use the local version.

1. Clone the repository.
1. Create a `.terraformrc` config file in your home directory (`~`) for the terraform CLI with the following content:

	```
	provider_installation {
   	   dev_overrides {
	      "registry.terraform.io/stackitcloud/stackit" = "<PATH_TO_BINARY>"
	   }

	   # For all other providers, install them directly from their origin provider
	   # registries as normal. If you omit this, Terraform will _only_ use
	   # the dev_overrides block, and so no other providers will be available.
	   direct {}
	}
	```
1. Copy one of the folders in the [examples](examples/) folder to a location of your choosing, and define the Terraform variables according to its README. The main.tf file needs some additional configuration to use the local provider:

	```
	terraform {
	   required_providers {
	      stackit = {
	         source = "registry.terraform.io/stackitcloud/stackit"
	      }
	   }
	}
	```

1. Go to the copied example and initialize Terraform by running `terraform init -reconfigure -upgrade`. This will throw an error ("Failed to query available provider packages") which can be ignored since we are using the local provider build.
   > Note: Terraform will store its resources' states locally. To allow multiple people to use the same resources, check [Setup for multi-person usage](#setup-centralized-terraform-state)
1. Setup authentication by setting the env var `STACKIT_SERVICE_ACCOUNT_TOKEN` as a valid token (see [Authentication](#authentication) for more details on how to autenticate).
1. Run `terraform plan` or `terraform apply` commands.
1. To debug the terraform provider, execute the following steps:
	* install the compiled terraform provider to binary path defined in the .terraformrc file
	* run the terraform provider from your IDE with the `-debug` flag set
	* The provider will emit the setting for the env variable `TF_REATTACH_PROVIDERS`, e.g.

	```shell
		TF_REATTACH_PROVIDERS='{"registry.terraform.io/stackitcloud/stackit":{"Protocol":"grpc","ProtocolVersion":6,"Pid":123456,"Test":true,"Addr":{"Network":"unix","String":"/tmp/plugin47110815"}}}'
	```

Starting terraform with this environment variable set will automatically connect to the running IDE session, instead of starting a new GRPC server with the plugin. This allows to set
breakpoints and inspect the state of the provider.
	


#### Setup centralized Terraform state

You'll need a storage bucket to store the Terraform state and a pair of access key/secret key.

- To order the bucket in the STACKIT Portal, go to Object Storage (on the right) > Buckets > Create bucket.
- To create credentials for a bucket in the STACKIT Portal, go Object Storage (on the right) > Credentials & Groups > Create credentials group.

In the main.tf file location, initialize Terraform by running the following:

```
terraform init -reconfigure -upgrade -backend-config="access_key=<STORAGE_BUCKET_ACCESS_KEY>" -backend-config="secret_key=<STORAGE_BUCKET_SECRET_KEY>"
```

This will throw an error ("Failed to query available provider packages") which can be ignored since we are using the local provider build.

## Code Contributions

To make your contribution, follow these steps:

1. Check open or recently closed [Pull Requests](https://github.com/stackitcloud/terraform-provider-stackit/pulls) and [Issues](https://github.com/stackitcloud/terraform-provider-stackit/issues)to make sure the contribution you are making has not been already tackled by someone else.
2. Fork the repo.
3. Make your changes in a branch that is up-to-date with the original repo's `main` branch.
4. Commit your changes including a descriptive message
5. Create a pull request with your changes.
6. The pull request will be reviewed by the repo maintainers. If you need to make further changes, make additional commits to keep commit history. When the PR is merged, commits will be squashed.

## Bug Reports

If you would like to report a bug, please open a [GitHub issue](https://github.com/stackitcloud/terraform-provider-stackit/issues/new).

To ensure we can provide the best support to your issue, follow these guidelines:

1. Go through the existing issues to check if your issue has already been reported.
2. Make sure you are using the latest version of the provider, we will not provide bug fixes for older versions. Also, latest versions may have the fix for your bug.
3. Please provide as much information as you can about your environment, e.g. your version of Go, your version of the provider, which operating system you are using and the corresponding version.
4. Include in your issue the steps to reproduce it, along with code snippets and/or information about your specific use case. This will make the support process much easier and efficient.
