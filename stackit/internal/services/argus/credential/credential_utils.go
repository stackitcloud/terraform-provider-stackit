package argus

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/argus"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

func mapFields(credentials *argus.Credentials, model *Model) error {
	if credentials == nil {
		return fmt.Errorf("no credentials returned from API")
	}

	// Check if model is nil
	if model == nil {
		return fmt.Errorf("model target is nil")
	}

	// Check for nil Username and Password
	if credentials.Username == nil || credentials.Password == nil {
		return fmt.Errorf("API did not return complete credential data")
	}

	// Safely assign Username and Password using pointers from the API response
	model.Username = types.StringPointerValue(credentials.Username)
	model.Password = types.StringPointerValue(credentials.Password)

	// Construct the ID using the provided core.Separator and ensure no additional escaping
	idParts := []string{model.ProjectId.ValueString(), model.InstanceId.ValueString(), model.Username.ValueString()}
	model.Id = types.StringValue(
		strings.Join(idParts, core.Separator),
	)

	return nil
}

func getCredentialsAndHandleErrors(ctx context.Context, instanceId, projectId, userName string, r *credentialResource, resp *resource.ReadResponse) error {
	_, err := r.client.GetCredentials(ctx, instanceId, projectId, userName).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return nil
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading credential", fmt.Sprintf("Calling API: %v", err))
		return err
	}
	return nil
}

func deleteCredentialsAndHandleErrors(ctx context.Context, instanceId, projectId, userName string, r *credentialResource, resp *resource.DeleteResponse) error {
	_, err := r.client.DeleteCredentials(ctx, instanceId, projectId, userName).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting credential", fmt.Sprintf("Calling API: %v", err))
		return err
	}
	return nil
}
