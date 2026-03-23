package utils

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/services/sfs"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func ConfigureClient(ctx context.Context, providerData *core.ProviderData, diags *diag.Diagnostics) *sfs.APIClient {
	apiClientConfigOptions := []config.ConfigurationOption{
		config.WithCustomAuth(providerData.RoundTripper),
		utils.UserAgentConfigOption(providerData.Version),
	}
	if providerData.SfsCustomEndpoint != "" {
		apiClientConfigOptions = append(apiClientConfigOptions, config.WithEndpoint(providerData.SfsCustomEndpoint))
	}
	apiClient, err := sfs.NewAPIClient(apiClientConfigOptions...)
	if err != nil {
		core.LogAndAddError(ctx, diags, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return nil
	}

	return apiClient
}

func DescribeValidationError(err sfs.ValidationError) string {
	var sb strings.Builder
	if err.Title != nil {
		sb.WriteString(*err.Title)
		sb.WriteRune('\n')
	}
	if fields := err.Fields; fields != nil {
		for _, field := range *fields {
			sb.WriteRune('\n')
			sb.WriteString("Field: ")
			if field.Field != nil {
				sb.WriteString(*field.Field)
			}
			sb.WriteString(" | Reason: ")
			if field.Reason != nil {
				sb.WriteString(*field.Reason)
			}
		}
	}
	return sb.String()
}

func LogAndAddError(ctx context.Context, diags *diag.Diagnostics, summary, detail string, err error) {
	if err == nil {
		return
	}
	message := err.Error()
	var oapiErr *oapierror.GenericOpenAPIError
	if errors.As(err, &oapiErr) {
		errModel := oapiErr.Model
		if validationErr, ok := errModel.(sfs.ValidationError); ok {
			message = DescribeValidationError(validationErr)
		}
	}
	core.LogAndAddError(ctx, diags, summary, fmt.Sprintf("%s: %v", detail, message))
}
