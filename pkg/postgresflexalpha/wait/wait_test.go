// Copyright (c) STACKIT

package wait

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresflex"
)

// Used for testing instance operations
type apiClientInstanceMocked struct {
	instanceId             string
	instanceState          string
	instanceIsForceDeleted bool
	instanceGetFails       bool
	usersGetErrorStatus    int
}

func (a *apiClientInstanceMocked) GetInstanceExecute(_ context.Context, _, _, _ string) (*postgresflex.InstanceResponse, error) {
	if a.instanceGetFails {
		return nil, &oapierror.GenericOpenAPIError{
			StatusCode: 500,
		}
	}

	if a.instanceIsForceDeleted {
		return nil, &oapierror.GenericOpenAPIError{
			StatusCode: 404,
		}
	}

	return &postgresflex.InstanceResponse{
		Item: &postgresflex.Instance{
			Id:     &a.instanceId,
			Status: &a.instanceState,
		},
	}, nil
}

func (a *apiClientInstanceMocked) ListUsersExecute(_ context.Context, _, _, _ string) (*postgresflex.ListUsersResponse, error) {
	if a.usersGetErrorStatus != 0 {
		return nil, &oapierror.GenericOpenAPIError{
			StatusCode: a.usersGetErrorStatus,
		}
	}

	aux := int64(0)
	return &postgresflex.ListUsersResponse{
		Count: &aux,
		Items: &[]postgresflex.ListUsersResponseItem{},
	}, nil
}

// Used for testing user operations
type apiClientUserMocked struct {
	getFails      bool
	userId        string
	isUserDeleted bool
}

func (a *apiClientUserMocked) GetUserExecute(_ context.Context, _, _, _, _ string) (*postgresflex.GetUserResponse, error) {
	if a.getFails {
		return nil, &oapierror.GenericOpenAPIError{
			StatusCode: 500,
		}
	}

	if a.isUserDeleted {
		return nil, &oapierror.GenericOpenAPIError{
			StatusCode: 404,
		}
	}

	return &postgresflex.GetUserResponse{
		Item: &postgresflex.UserResponse{
			Id: &a.userId,
		},
	}, nil
}

func TestCreateInstanceWaitHandler(t *testing.T) {
	tests := []struct {
		desc                string
		instanceGetFails    bool
		instanceState       string
		usersGetErrorStatus int
		wantErr             bool
		wantResp            bool
	}{
		{
			desc:             "create_succeeded",
			instanceGetFails: false,
			instanceState:    InstanceStateSuccess,
			wantErr:          false,
			wantResp:         true,
		},
		{
			desc:             "create_failed",
			instanceGetFails: false,
			instanceState:    InstanceStateFailed,
			wantErr:          true,
			wantResp:         true,
		},
		{
			desc:             "create_failed_2",
			instanceGetFails: false,
			instanceState:    InstanceStateEmpty,
			wantErr:          true,
			wantResp:         false,
		},
		{
			desc:             "instance_get_fails",
			instanceGetFails: true,
			wantErr:          true,
			wantResp:         false,
		},
		{
			desc:                "users_get_fails",
			instanceGetFails:    false,
			instanceState:       InstanceStateSuccess,
			usersGetErrorStatus: 500,
			wantErr:             true,
			wantResp:            false,
		},
		{
			desc:                "users_get_fails_2",
			instanceGetFails:    false,
			instanceState:       InstanceStateSuccess,
			usersGetErrorStatus: 400,
			wantErr:             true,
			wantResp:            true,
		},
		{
			desc:             "timeout",
			instanceGetFails: false,
			instanceState:    InstanceStateProgressing,
			wantErr:          true,
			wantResp:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			instanceId := "foo-bar"

			apiClient := &apiClientInstanceMocked{
				instanceId:          instanceId,
				instanceState:       tt.instanceState,
				instanceGetFails:    tt.instanceGetFails,
				usersGetErrorStatus: tt.usersGetErrorStatus,
			}

			var wantRes *postgresflex.InstanceResponse
			if tt.wantResp {
				wantRes = &postgresflex.InstanceResponse{
					Item: &postgresflex.Instance{
						Id:     &instanceId,
						Status: utils.Ptr(tt.instanceState),
					},
				}
			}

			handler := CreateInstanceWaitHandler(context.Background(), apiClient, "", "", instanceId)

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

func TestUpdateInstanceWaitHandler(t *testing.T) {
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
			wantResp:         false,
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
			instanceState:    InstanceStateProgressing,
			wantErr:          true,
			wantResp:         false,
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

			var wantRes *postgresflex.InstanceResponse
			if tt.wantResp {
				wantRes = &postgresflex.InstanceResponse{
					Item: &postgresflex.Instance{
						Id:     &instanceId,
						Status: utils.Ptr(tt.instanceState),
					},
				}
			}

			handler := PartialUpdateInstanceWaitHandler(context.Background(), apiClient, "", "", instanceId)

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
			instanceState:    InstanceStateDeleted,
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
				instanceGetFails: tt.instanceGetFails,
				instanceId:       instanceId,
				instanceState:    tt.instanceState,
			}

			handler := DeleteInstanceWaitHandler(context.Background(), apiClient, "", "", instanceId)

			_, err := handler.SetTimeout(10 * time.Millisecond).WaitWithContext(context.Background())

			if (err != nil) != tt.wantErr {
				t.Fatalf("handler error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestForceDeleteInstanceWaitHandler(t *testing.T) {
	tests := []struct {
		desc             string
		instanceGetFails bool
		instanceState    string
		wantErr          bool
	}{
		{
			desc:             "delete_succeeded",
			instanceGetFails: false,
			instanceState:    InstanceStateDeleted,
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
				instanceGetFails:       tt.instanceGetFails,
				instanceIsForceDeleted: tt.instanceState == InstanceStateDeleted,
				instanceId:             instanceId,
				instanceState:          tt.instanceState,
			}

			handler := ForceDeleteInstanceWaitHandler(context.Background(), apiClient, "", "", instanceId)

			_, err := handler.SetTimeout(10 * time.Millisecond).WaitWithContext(context.Background())

			if (err != nil) != tt.wantErr {
				t.Fatalf("handler error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteUserWaitHandler(t *testing.T) {
	tests := []struct {
		desc        string
		deleteFails bool
		getFails    bool
		wantErr     bool
	}{
		{
			desc:        "delete_succeeded",
			deleteFails: false,
			getFails:    false,
			wantErr:     false,
		},
		{
			desc:        "delete_failed",
			deleteFails: true,
			getFails:    false,
			wantErr:     true,
		},
		{
			desc:        "get_fails",
			deleteFails: false,
			getFails:    true,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			userId := "foo-bar"

			apiClient := &apiClientUserMocked{
				getFails:      tt.getFails,
				userId:        userId,
				isUserDeleted: !tt.deleteFails,
			}

			handler := DeleteUserWaitHandler(context.Background(), apiClient, "", "", "", userId)

			_, err := handler.SetTimeout(10 * time.Millisecond).WaitWithContext(context.Background())

			if (err != nil) != tt.wantErr {
				t.Fatalf("handler error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
