// Copyright (c) STACKIT

package utils

import (
	"fmt"

	"github.com/stackitcloud/stackit-sdk-go/core/config"
)

func UserAgentConfigOption(providerVersion string) config.ConfigurationOption {
	return config.WithUserAgent(fmt.Sprintf("stackit-terraform-provider/%s", providerVersion))
}
