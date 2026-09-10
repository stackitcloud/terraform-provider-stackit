package servicecall

import (
	"context"

	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api/wait"
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

func waiterCallIsServiceCall(client iaas.DefaultAPI) { // want waiterCallIsServiceCall:"serviceCall"
	ctx := context.Background()
	wait.CreateSnapshotWaitHandler(ctx, client, "", "", "").WaitWithContext(ctx)
}

func waiterCreationIsNotAServiceCall(client iaas.DefaultAPI) {
	ctx := context.Background()
	wait.CreateSnapshotWaitHandler(ctx, client, "", "", "")
}
