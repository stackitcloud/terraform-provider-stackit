package utils

import (
	ske "github.com/stackitcloud/stackit-sdk-go/services/ske/v2api"
)

func IsEmptyNetwork(network *ske.Network) bool {
	if !network.HasId() && !network.HasControlPlane() {
		return true
	}
	return false
}

func IsEmptyExtension(extension *ske.Extension) bool {
	if !extension.HasDns() && !extension.HasAcl() && !extension.HasObservability() {
		return true
	}
	return false
}
