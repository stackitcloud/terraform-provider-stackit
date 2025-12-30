// Copyright (c) STACKIT

package wait

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	sqlserverflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/sqlserverflexalpha"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
)

// Used for testing instance operations
type apiClientInstanceMocked struct {
	instanceId        string
	instanceState     string
	instanceNetwork   sqlserverflex.InstanceNetwork
	instanceIsDeleted bool
	instanceGetFails  bool
}

func (a *apiClientInstanceMocked) GetInstanceRequestExecute(_ context.Context, _, _, _ string) (*sqlserverflex.GetInstanceResponse, error) {
	if a.instanceGetFails {
		return nil, &oapierror.GenericOpenAPIError{
			StatusCode: 500,
		}
	}

	if a.instanceIsDeleted {
		return nil, &oapierror.GenericOpenAPIError{
			StatusCode: 404,
		}
	}

	return &sqlserverflex.GetInstanceResponse{
		Id:      &a.instanceId,
		Status:  sqlserverflex.GetInstanceResponseGetStatusAttributeType(&a.instanceState),
		Network: &a.instanceNetwork,
	}, nil
}
func TestCreateInstanceWaitHandler(t *testing.T) {
	t.Skip("skipping - needs refactoring")
	tests := []struct {
		desc                string
		instanceGetFails    bool
		instanceState       string
		instanceNetwork     sqlserverflex.InstanceNetwork
		usersGetErrorStatus int
		wantErr             bool
		wantRes             *sqlserverflex.GetInstanceResponse
	}{
		{
			desc:             "create_succeeded",
			instanceGetFails: false,
			instanceState:    InstanceStateSuccess,
			instanceNetwork: sqlserverflex.InstanceNetwork{
				AccessScope:     nil,
				Acl:             nil,
				InstanceAddress: utils.Ptr("10.0.0.1"),
				RouterAddress:   utils.Ptr("10.0.0.2"),
			},
			wantErr: false,
			wantRes: &sqlserverflex.GetInstanceResponse{
				BackupSchedule: nil,
				Edition:        nil,
				Encryption:     nil,
				FlavorId:       nil,
				Id:             nil,
				IsDeletable:    nil,
				Name:           nil,
				Network: &sqlserverflex.InstanceNetwork{
					AccessScope:     nil,
					Acl:             nil,
					InstanceAddress: utils.Ptr("10.0.0.1"),
					RouterAddress:   utils.Ptr("10.0.0.2"),
				},
				Replicas:      nil,
				RetentionDays: nil,
				Status:        nil,
				Storage:       nil,
				Version:       nil,
			},
		},
		{
			desc:             "create_failed",
			instanceGetFails: false,
			instanceState:    InstanceStateFailed,
			wantErr:          true,
			wantRes:          nil,
		},
		{
			desc:             "create_failed_2",
			instanceGetFails: false,
			instanceState:    InstanceStateEmpty,
			wantErr:          true,
			wantRes:          nil,
		},
		{
			desc:             "instance_get_fails",
			instanceGetFails: true,
			wantErr:          true,
			wantRes:          nil,
		},
		{
			desc:             "timeout",
			instanceGetFails: false,
			instanceState:    InstanceStateProcessing,
			wantErr:          true,
			wantRes:          nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			instanceId := "foo-bar"

			apiClient := &apiClientInstanceMocked{
				instanceId:       instanceId,
				instanceState:    tt.instanceState,
				instanceGetFails: tt.instanceGetFails,
			}

			handler := CreateInstanceWaitHandler(context.Background(), apiClient, "", instanceId, "")

			gotRes, err := handler.SetTimeout(10 * time.Millisecond).SetSleepBeforeWait(1 * time.Millisecond).WaitWithContext(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("handler error = %v, wantErr %v", err, tt.wantErr)
			}

			if !cmp.Equal(gotRes, tt.wantRes) {
				t.Fatalf("handler gotRes = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

func TestUpdateInstanceWaitHandler(t *testing.T) {
	t.Skip("skipping - needs refactoring")
	tests := []struct {
		desc             string
		instanceGetFails bool
		instanceState    string
		wantErr          bool
		wantResp         bool
	}{
		{
			desc:             "update_succeeded",
			instanceGetFails: false,
			instanceState:    InstanceStateSuccess,
			wantErr:          false,
			wantResp:         true,
		},
		{
			desc:             "update_failed",
			instanceGetFails: false,
			instanceState:    InstanceStateFailed,
			wantErr:          true,
			wantResp:         true,
		},
		{
			desc:             "update_failed_2",
			instanceGetFails: false,
			instanceState:    InstanceStateEmpty,
			wantErr:          true,
			wantResp:         true,
		},
		{
			desc:             "get_fails",
			instanceGetFails: true,
			wantErr:          true,
			wantResp:         false,
		},
		{
			desc:             "timeout",
			instanceGetFails: false,
			instanceState:    InstanceStateProcessing,
			wantErr:          true,
			wantResp:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			instanceId := "foo-bar"

			apiClient := &apiClientInstanceMocked{
				instanceId:       instanceId,
				instanceState:    tt.instanceState,
				instanceGetFails: tt.instanceGetFails,
			}

			var wantRes *sqlserverflex.GetInstanceResponse
			if tt.wantResp {
				wantRes = &sqlserverflex.GetInstanceResponse{
					Id:     &instanceId,
					Status: sqlserverflex.GetInstanceResponseGetStatusAttributeType(utils.Ptr(tt.instanceState)),
				}
			}

			handler := UpdateInstanceWaitHandler(context.Background(), apiClient, "", instanceId, "")

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

func TestDeleteInstanceWaitHandler(t *testing.T) {
	tests := []struct {
		desc             string
		instanceGetFails bool
		instanceState    string
		wantErr          bool
	}{
		{
			desc:             "delete_succeeded",
			instanceGetFails: false,
			instanceState:    InstanceStateSuccess,
			wantErr:          false,
		},
		{
			desc:             "delete_failed",
			instanceGetFails: false,
			instanceState:    InstanceStateFailed,
			wantErr:          true,
		},
		{
			desc:             "get_fails",
			instanceGetFails: true,
			wantErr:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			instanceId := "foo-bar"

			apiClient := &apiClientInstanceMocked{
				instanceGetFails:  tt.instanceGetFails,
				instanceIsDeleted: tt.instanceState == InstanceStateSuccess,
				instanceId:        instanceId,
				instanceState:     tt.instanceState,
			}

			handler := DeleteInstanceWaitHandler(context.Background(), apiClient, "", instanceId, "")

			_, err := handler.SetTimeout(10 * time.Millisecond).WaitWithContext(context.Background())

			if (err != nil) != tt.wantErr {
				t.Fatalf("handler error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
