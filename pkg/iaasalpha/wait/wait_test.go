// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: Apache-2.0

package wait

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
)

type apiClientMocked struct {
	getNetworkFails bool
	isDeleted       bool
	resourceState   string
}

func (a *apiClientMocked) GetNetworkExecute(_ context.Context, _, _, _ string) (*iaasalpha.Network, error) {
	if a.isDeleted {
		return nil, &oapierror.GenericOpenAPIError{
			StatusCode: 404,
		}
	}

	if a.getNetworkFails {
		return nil, &oapierror.GenericOpenAPIError{
			StatusCode: 500,
		}
	}

	return &iaasalpha.Network{
		Id:     utils.Ptr("nid"),
		Status: &a.resourceState,
	}, nil
}

func TestCreateNetworkWaitHandler(t *testing.T) {
	tests := []struct {
		desc          string
		getFails      bool
		resourceState string
		wantErr       bool
		wantResp      bool
	}{
		{
			desc:          "create_succeeded",
			getFails:      false,
			resourceState: CreateSuccess,
			wantErr:       false,
			wantResp:      true,
		},
		{
			desc:          "get_fails",
			getFails:      true,
			resourceState: "",
			wantErr:       true,
			wantResp:      false,
		},
		{
			desc:          "timeout",
			getFails:      false,
			resourceState: "ANOTHER STATE",
			wantErr:       true,
			wantResp:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			apiClient := &apiClientMocked{
				getNetworkFails: tt.getFails,
				resourceState:   tt.resourceState,
			}

			var wantRes *iaasalpha.Network
			if tt.wantResp {
				wantRes = &iaasalpha.Network{
					Id:     utils.Ptr("nid"),
					Status: utils.Ptr(tt.resourceState),
				}
			}

			handler := CreateNetworkWaitHandler(context.Background(), apiClient, "pid", "eu01", "nid")

			gotRes, err := handler.SetTimeout(10 * time.Millisecond).SetSleepBeforeWait(1 * time.Millisecond).WaitWithContext(context.Background())

			if (err != nil) != tt.wantErr {
				t.Fatalf("handler error = %v, wantErr %v", err, tt.wantErr)
			}
			if !cmp.Equal(gotRes, wantRes) {
				t.Fatalf("handler gotRes = %v, want %v", gotRes, wantRes)
			}
		})
	}
}

func TestUpdateNetworkWaitHandler(t *testing.T) {
	tests := []struct {
		desc          string
		getFails      bool
		resourceState string
		wantErr       bool
		wantResp      bool
	}{
		{
			desc:          "update_succeeded",
			getFails:      false,
			resourceState: CreateSuccess,
			wantErr:       false,
			wantResp:      true,
		},
		{
			desc:          "get_fails",
			getFails:      true,
			resourceState: "",
			wantErr:       true,
			wantResp:      false,
		},
		{
			desc:          "timeout",
			getFails:      false,
			resourceState: "ANOTHER STATE",
			wantErr:       true,
			wantResp:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			apiClient := &apiClientMocked{
				getNetworkFails: tt.getFails,
				resourceState:   tt.resourceState,
			}

			var wantRes *iaasalpha.Network
			if tt.wantResp {
				wantRes = &iaasalpha.Network{
					Id:     utils.Ptr("nid"),
					Status: utils.Ptr(tt.resourceState),
				}
			}

			handler := UpdateNetworkWaitHandler(context.Background(), apiClient, "pid", "eu01", "nid")

			gotRes, err := handler.SetTimeout(10 * time.Millisecond).SetSleepBeforeWait(1 * time.Millisecond).WaitWithContext(context.Background())

			if (err != nil) != tt.wantErr {
				t.Fatalf("handler error = %v, wantErr %v", err, tt.wantErr)
			}
			if !cmp.Equal(gotRes, wantRes) {
				t.Fatalf("handler gotRes = %v, want %v", gotRes, wantRes)
			}
		})
	}
}

func TestDeleteNetworkWaitHandler(t *testing.T) {
	tests := []struct {
		desc          string
		getFails      bool
		isDeleted     bool
		resourceState string
		wantErr       bool
		wantResp      bool
	}{
		{
			desc:      "delete_succeeded",
			getFails:  false,
			isDeleted: true,
			wantErr:   false,
			wantResp:  false,
		},
		{
			desc:          "get_fails",
			getFails:      true,
			resourceState: "",
			wantErr:       true,
			wantResp:      false,
		},
		{
			desc:          "timeout",
			getFails:      false,
			resourceState: "ANOTHER STATE",
			wantErr:       true,
			wantResp:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			apiClient := &apiClientMocked{
				getNetworkFails: tt.getFails,
				isDeleted:       tt.isDeleted,
				resourceState:   tt.resourceState,
			}

			var wantRes *iaasalpha.Network
			if tt.wantResp {
				wantRes = &iaasalpha.Network{
					Id:     utils.Ptr("nid"),
					Status: utils.Ptr(tt.resourceState),
				}
			}

			handler := DeleteNetworkWaitHandler(context.Background(), apiClient, "pid", "eu01", "nid")

			gotRes, err := handler.SetTimeout(10 * time.Millisecond).WaitWithContext(context.Background())

			if (err != nil) != tt.wantErr {
				t.Fatalf("handler error = %v, wantErr %v", err, tt.wantErr)
			}
			if !cmp.Equal(gotRes, wantRes) {
				t.Fatalf("handler gotRes = %v, want %v", gotRes, wantRes)
			}
		})
	}
}
