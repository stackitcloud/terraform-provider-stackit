package servicecall

import (
	"context"

	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

func callsService(service *iaas.DefaultAPIService) { // want callsService:"serviceCall"
	service.AddNetworkToServerExecute(iaas.ApiAddNetworkToServerRequest{})
}

func propagatesServiceCall(service *iaas.DefaultAPIService) { // want propagatesServiceCall:"serviceCall"
	callsService(service)
}

func propagatesAgain(service *iaas.DefaultAPIService) { // want propagatesAgain:"serviceCall"
	propagatesServiceCall(service)
}

func doesNotCallService(client iaas.DefaultAPI) {
	client.AddNetworkToServer(context.Background(), "", "", "", "")
}

func callsServiceInterface(client iaas.DefaultAPI) { // want callsServiceInterface:"serviceCall"
	client.AddNetworkToServer(context.Background(), "", "", "", "").Execute()
}
