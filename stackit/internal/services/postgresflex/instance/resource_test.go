package postgresflex

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	postgresflex "github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v2api"
)

// move to generator - START
//var _ postgresflex.DefaultAPI = &ApiDefaultMock{}
//
//type ApiDefaultMock struct {
//	CloneInstanceMock                 *func(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiCloneInstanceRequest
//	CloneInstanceExecuteMock          *func(r postgresflex.ApiCloneInstanceRequest) (*postgresflex.CloneInstanceResponse, *http.Response, error)
//	CreateDatabaseMock                *func(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiCreateDatabaseRequest
//	CreateDatabaseExecuteMock         *func(r postgresflex.ApiCreateDatabaseRequest) (*postgresflex.InstanceCreateDatabaseResponse, *http.Response, error)
//	CreateInstanceMock                *func(ctx context.Context, projectId string, region string) postgresflex.ApiCreateInstanceRequest
//	CreateInstanceExecuteMock         *func(r postgresflex.ApiCreateInstanceRequest) (*postgresflex.CreateInstanceResponse, *http.Response, error)
//	CreateUserMock                    *func(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiCreateUserRequest
//	CreateUserExecuteMock             *func(r postgresflex.ApiCreateUserRequest) (*postgresflex.CreateUserResponse, *http.Response, error)
//	DeleteDatabaseMock                *func(ctx context.Context, projectId string, region string, instanceId string, databaseId string) postgresflex.ApiDeleteDatabaseRequest
//	DeleteDatabaseExecuteMock         *func(r postgresflex.ApiDeleteDatabaseRequest) (*http.Response, error)
//	DeleteInstanceMock                *func(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiDeleteInstanceRequest
//	DeleteInstanceExecuteMock         *func(r postgresflex.ApiDeleteInstanceRequest) (*http.Response, error)
//	DeleteUserMock                    *func(ctx context.Context, projectId string, region string, instanceId string, userId string) postgresflex.ApiDeleteUserRequest
//	DeleteUserExecuteMock             *func(r postgresflex.ApiDeleteUserRequest) (*http.Response, error)
//	ForceDeleteInstanceMock           *func(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiForceDeleteInstanceRequest
//	ForceDeleteInstanceExecuteMock    *func(r postgresflex.ApiForceDeleteInstanceRequest) (*http.Response, error)
//	GetBackupMock                     *func(ctx context.Context, projectId string, region string, instanceId string, backupId string) postgresflex.ApiGetBackupRequest
//	GetBackupExecuteMock              *func(r postgresflex.ApiGetBackupRequest) (*postgresflex.GetBackupResponse, *http.Response, error)
//	GetInstanceMock                   *func(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiGetInstanceRequest
//	GetInstanceExecuteMock            *func(r postgresflex.ApiGetInstanceRequest) (*postgresflex.InstanceResponse, *http.Response, error)
//	GetUserMock                       *func(ctx context.Context, projectId string, region string, instanceId string, userId string) postgresflex.ApiGetUserRequest
//	GetUserExecuteMock                *func(r postgresflex.ApiGetUserRequest) (*postgresflex.GetUserResponse, *http.Response, error)
//	ListBackupsMock                   *func(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiListBackupsRequest
//	ListBackupsExecuteMock            *func(r postgresflex.ApiListBackupsRequest) (*postgresflex.ListBackupsResponse, *http.Response, error)
//	ListDatabaseParametersMock        *func(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiListDatabaseParametersRequest
//	ListDatabaseParametersExecuteMock *func(r postgresflex.ApiListDatabaseParametersRequest) (*postgresflex.PostgresDatabaseParameterResponse, *http.Response, error)
//	ListDatabasesMock                 *func(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiListDatabasesRequest
//	ListDatabasesExecuteMock          *func(r postgresflex.ApiListDatabasesRequest) (*postgresflex.InstanceListDatabasesResponse, *http.Response, error)
//	ListFlavorsMock                   *func(ctx context.Context, projectId string, region string) postgresflex.ApiListFlavorsRequest
//	ListFlavorsExecuteMock            *func(r postgresflex.ApiListFlavorsRequest) (*postgresflex.ListFlavorsResponse, *http.Response, error)
//	ListInstancesMock                 *func(ctx context.Context, projectId string, region string) postgresflex.ApiListInstancesRequest
//	ListInstancesExecuteMock          *func(r postgresflex.ApiListInstancesRequest) (*postgresflex.ListInstancesResponse, *http.Response, error)
//	ListMetricsMock                   *func(ctx context.Context, projectId string, region string, instanceId string, metric string) postgresflex.ApiListMetricsRequest
//	ListMetricsExecuteMock            *func(r postgresflex.ApiListMetricsRequest) (*postgresflex.InstanceMetricsResponse, *http.Response, error)
//	ListStoragesMock                  *func(ctx context.Context, projectId string, region string, flavorId string) postgresflex.ApiListStoragesRequest
//	ListStoragesExecuteMock           *func(r postgresflex.ApiListStoragesRequest) (*postgresflex.ListStoragesResponse, *http.Response, error)
//	ListUsersMock                     *func(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiListUsersRequest
//	ListUsersExecuteMock              *func(r postgresflex.ApiListUsersRequest) (*postgresflex.ListUsersResponse, *http.Response, error)
//	ListVersionsMock                  *func(ctx context.Context, projectId string, region string) postgresflex.ApiListVersionsRequest
//	ListVersionsExecuteMock           *func(r postgresflex.ApiListVersionsRequest) (*postgresflex.ListVersionsResponse, *http.Response, error)
//	PartialUpdateInstanceMock         *func(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiPartialUpdateInstanceRequest
//	PartialUpdateInstanceExecuteMock  *func(r postgresflex.ApiPartialUpdateInstanceRequest) (*postgresflex.PartialUpdateInstanceResponse, *http.Response, error)
//	PartialUpdateUserMock             *func(ctx context.Context, projectId string, region string, instanceId string, userId string) postgresflex.ApiPartialUpdateUserRequest
//	PartialUpdateUserExecuteMock      *func(r postgresflex.ApiPartialUpdateUserRequest) (*http.Response, error)
//	ResetUserMock                     *func(ctx context.Context, projectId string, region string, instanceId string, userId string) postgresflex.ApiResetUserRequest
//	ResetUserExecuteMock              *func(r postgresflex.ApiResetUserRequest) (*postgresflex.ResetUserResponse, *http.Response, error)
//	UpdateBackupScheduleMock          *func(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiUpdateBackupScheduleRequest
//	UpdateBackupScheduleExecuteMock   *func(r postgresflex.ApiUpdateBackupScheduleRequest) (*http.Response, error)
//	UpdateInstanceMock                *func(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiUpdateInstanceRequest
//	UpdateInstanceExecuteMock         *func(r postgresflex.ApiUpdateInstanceRequest) (*postgresflex.PartialUpdateInstanceResponse, *http.Response, error)
//	UpdateUserMock                    *func(ctx context.Context, projectId string, region string, instanceId string, userId string) postgresflex.ApiUpdateUserRequest
//	UpdateUserExecuteMock             *func(r postgresflex.ApiUpdateUserRequest) (*http.Response, error)
//}
//
//func (a ApiDefaultMock) CloneInstance(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiCloneInstanceRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) CloneInstanceExecute(r postgresflex.ApiCloneInstanceRequest) (*postgresflex.CloneInstanceResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) CreateDatabase(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiCreateDatabaseRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) CreateDatabaseExecute(r postgresflex.ApiCreateDatabaseRequest) (*postgresflex.InstanceCreateDatabaseResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) CreateInstance(ctx context.Context, projectId string, region string) postgresflex.ApiCreateInstanceRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) CreateInstanceExecute(r postgresflex.ApiCreateInstanceRequest) (*postgresflex.CreateInstanceResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) CreateUser(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiCreateUserRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) CreateUserExecute(r postgresflex.ApiCreateUserRequest) (*postgresflex.CreateUserResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) DeleteDatabase(ctx context.Context, projectId string, region string, instanceId string, databaseId string) postgresflex.ApiDeleteDatabaseRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) DeleteDatabaseExecute(r postgresflex.ApiDeleteDatabaseRequest) (*http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) DeleteInstance(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiDeleteInstanceRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) DeleteInstanceExecute(r postgresflex.ApiDeleteInstanceRequest) (*http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) DeleteUser(ctx context.Context, projectId string, region string, instanceId string, userId string) postgresflex.ApiDeleteUserRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) DeleteUserExecute(r postgresflex.ApiDeleteUserRequest) (*http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ForceDeleteInstance(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiForceDeleteInstanceRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ForceDeleteInstanceExecute(r postgresflex.ApiForceDeleteInstanceRequest) (*http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) GetBackup(ctx context.Context, projectId string, region string, instanceId string, backupId string) postgresflex.ApiGetBackupRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) GetBackupExecute(r postgresflex.ApiGetBackupRequest) (*postgresflex.GetBackupResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) GetInstance(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiGetInstanceRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) GetInstanceExecute(r postgresflex.ApiGetInstanceRequest) (*postgresflex.InstanceResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) GetUser(ctx context.Context, projectId string, region string, instanceId string, userId string) postgresflex.ApiGetUserRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) GetUserExecute(r postgresflex.ApiGetUserRequest) (*postgresflex.GetUserResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ListBackups(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiListBackupsRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ListBackupsExecute(r postgresflex.ApiListBackupsRequest) (*postgresflex.ListBackupsResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ListDatabaseParameters(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiListDatabaseParametersRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ListDatabaseParametersExecute(r postgresflex.ApiListDatabaseParametersRequest) (*postgresflex.PostgresDatabaseParameterResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ListDatabases(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiListDatabasesRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ListDatabasesExecute(r postgresflex.ApiListDatabasesRequest) (*postgresflex.InstanceListDatabasesResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//// these can be generated 100%, no need to add overwrites via mocks, only the "Execute" suffix funcs need to be able to be mocked
//func (a ApiDefaultMock) ListFlavors(ctx context.Context, projectId string, region string) postgresflex.ApiListFlavorsRequest {
//	return postgresflex.ApiListFlavorsRequest{
//		ApiService: a,
//	}
//}
//
//func (a ApiDefaultMock) ListFlavorsExecute(r postgresflex.ApiListFlavorsRequest) (*postgresflex.ListFlavorsResponse, *http.Response, error) {
//	if a.ListFlavorsExecuteMock == nil {
//		return nil, nil, nil
//	}
//
//	return (*a.ListFlavorsExecuteMock)(r)
//}
//
//func (a ApiDefaultMock) ListInstances(ctx context.Context, projectId string, region string) postgresflex.ApiListInstancesRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ListInstancesExecute(r postgresflex.ApiListInstancesRequest) (*postgresflex.ListInstancesResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ListMetrics(ctx context.Context, projectId string, region string, instanceId string, metric string) postgresflex.ApiListMetricsRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ListMetricsExecute(r postgresflex.ApiListMetricsRequest) (*postgresflex.InstanceMetricsResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ListStorages(ctx context.Context, projectId string, region string, flavorId string) postgresflex.ApiListStoragesRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ListStoragesExecute(r postgresflex.ApiListStoragesRequest) (*postgresflex.ListStoragesResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ListUsers(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiListUsersRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ListUsersExecute(r postgresflex.ApiListUsersRequest) (*postgresflex.ListUsersResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ListVersions(ctx context.Context, projectId string, region string) postgresflex.ApiListVersionsRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ListVersionsExecute(r postgresflex.ApiListVersionsRequest) (*postgresflex.ListVersionsResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) PartialUpdateInstance(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiPartialUpdateInstanceRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) PartialUpdateInstanceExecute(r postgresflex.ApiPartialUpdateInstanceRequest) (*postgresflex.PartialUpdateInstanceResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) PartialUpdateUser(ctx context.Context, projectId string, region string, instanceId string, userId string) postgresflex.ApiPartialUpdateUserRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) PartialUpdateUserExecute(r postgresflex.ApiPartialUpdateUserRequest) (*http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ResetUser(ctx context.Context, projectId string, region string, instanceId string, userId string) postgresflex.ApiResetUserRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) ResetUserExecute(r postgresflex.ApiResetUserRequest) (*postgresflex.ResetUserResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) UpdateBackupSchedule(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiUpdateBackupScheduleRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) UpdateBackupScheduleExecute(r postgresflex.ApiUpdateBackupScheduleRequest) (*http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) UpdateInstance(ctx context.Context, projectId string, region string, instanceId string) postgresflex.ApiUpdateInstanceRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) UpdateInstanceExecute(r postgresflex.ApiUpdateInstanceRequest) (*postgresflex.PartialUpdateInstanceResponse, *http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) UpdateUser(ctx context.Context, projectId string, region string, instanceId string, userId string) postgresflex.ApiUpdateUserRequest {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (a ApiDefaultMock) UpdateUserExecute(r postgresflex.ApiUpdateUserRequest) (*http.Response, error) {
//	//TODO implement me
//	panic("implement me")
//}

// move to generator - END

func newApiMock(
	returnError bool,
	getFlavorsResp *postgresflex.ListFlavorsResponse,
) postgresflex.DefaultAPI {
	return &postgresflex.DefaultAPIServiceMock{
		ListFlavorsExecuteMock: utils.Ptr(func(r postgresflex.ApiListFlavorsRequest) (*postgresflex.ListFlavorsResponse, *http.Response, error) {
			if returnError {
				return nil, nil, fmt.Errorf("get flavors failed")
			}

			return getFlavorsResp, &http.Response{}, nil
		}),
	}
}

//type postgresFlexClientMocked struct {
//	returnError    bool
//	getFlavorsResp *postgresflex.ListFlavorsResponse
//
//	ApiDefaultMock
//}
//
//func (c *postgresFlexClientMocked) ListFlavors(_ context.Context, _, _ string) postgresflex.ApiListFlavorsRequest {
//	return postgresflex.ApiListFlavorsRequest{}
//}
//
//func (c *postgresFlexClientMocked) ListFlavorsExecute(_ postgresflex.ApiListFlavorsRequest) (*postgresflex.ListFlavorsResponse, *http.Response, error) {
//	if c.returnError {
//		return nil, nil, fmt.Errorf("get flavors failed")
//	}
//
//	return c.getFlavorsResp, &http.Response{}, nil
//}

func TestMapFields(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		state       Model
		input       *postgresflex.InstanceResponse
		flavor      *flavorModel
		storage     *storageModel
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&postgresflex.InstanceResponse{
				Item: &postgresflex.Instance{},
			},
			&flavorModel{},
			&storageModel{},
			testRegion,
			Model{
				Id:             types.StringValue("pid,region,iid"),
				InstanceId:     types.StringValue("iid"),
				ProjectId:      types.StringValue("pid"),
				Name:           types.StringNull(),
				ACL:            types.ListNull(types.StringType),
				BackupSchedule: types.StringNull(),
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringNull(),
					"description": types.StringNull(),
					"cpu":         types.Int64Null(),
					"ram":         types.Int64Null(),
				}),
				Replicas: types.Int32Null(),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringNull(),
					"size":  types.Int64Null(),
				}),
				Version: types.StringNull(),
				Region:  types.StringValue(testRegion),
			},
			true,
		},
		{
			"simple_values",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&postgresflex.InstanceResponse{
				Item: &postgresflex.Instance{
					Acl: &postgresflex.ACL{
						Items: []string{
							"ip1",
							"ip2",
							"",
						},
					},
					BackupSchedule: utils.Ptr("schedule"),
					Flavor: &postgresflex.Flavor{
						Cpu:         utils.Ptr(int64(12)),
						Description: utils.Ptr("description"),
						Id:          utils.Ptr("flavor_id"),
						Memory:      utils.Ptr(int64(34)),
					},
					Id:       utils.Ptr("iid"),
					Name:     utils.Ptr("name"),
					Replicas: utils.Ptr(int32(56)),
					Status:   utils.Ptr("status"),
					Storage: &postgresflex.Storage{
						Class: utils.Ptr("class"),
						Size:  utils.Ptr(int64(78)),
					},
					Version: utils.Ptr("version"),
				},
			},
			&flavorModel{},
			&storageModel{},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				ACL: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ip1"),
					types.StringValue("ip2"),
					types.StringValue(""),
				}),
				BackupSchedule: types.StringValue("schedule"),
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringValue("flavor_id"),
					"description": types.StringValue("description"),
					"cpu":         types.Int64Value(12),
					"ram":         types.Int64Value(34),
				}),
				Replicas: types.Int32Value(56),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("class"),
					"size":  types.Int64Value(78),
				}),
				Version: types.StringValue("version"),
				Region:  types.StringValue(testRegion),
			},
			true,
		},
		{
			"simple_values_no_flavor_and_storage",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&postgresflex.InstanceResponse{
				Item: &postgresflex.Instance{
					Acl: &postgresflex.ACL{
						Items: []string{
							"ip1",
							"ip2",
							"",
						},
					},
					BackupSchedule: utils.Ptr("schedule"),
					Flavor:         nil,
					Id:             utils.Ptr("iid"),
					Name:           utils.Ptr("name"),
					Replicas:       utils.Ptr(int32(56)),
					Status:         utils.Ptr("status"),
					Storage:        nil,
					Version:        utils.Ptr("version"),
				},
			},
			&flavorModel{
				CPU: types.Int64Value(12),
				RAM: types.Int64Value(34),
			},
			&storageModel{
				Class: types.StringValue("class"),
				Size:  types.Int64Value(78),
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				ACL: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ip1"),
					types.StringValue("ip2"),
					types.StringValue(""),
				}),
				BackupSchedule: types.StringValue("schedule"),
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringNull(),
					"description": types.StringNull(),
					"cpu":         types.Int64Value(12),
					"ram":         types.Int64Value(34),
				}),
				Replicas: types.Int32Value(56),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("class"),
					"size":  types.Int64Value(78),
				}),
				Version: types.StringValue("version"),
				Region:  types.StringValue(testRegion),
			},
			true,
		},
		{
			"acl_unordered",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				ACL: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ip2"),
					types.StringValue(""),
					types.StringValue("ip1"),
				}),
			},
			&postgresflex.InstanceResponse{
				Item: &postgresflex.Instance{
					Acl: &postgresflex.ACL{
						Items: []string{
							"",
							"ip1",
							"ip2",
						},
					},
					BackupSchedule: utils.Ptr("schedule"),
					Flavor:         nil,
					Id:             utils.Ptr("iid"),
					Name:           utils.Ptr("name"),
					Replicas:       utils.Ptr(int32(56)),
					Status:         utils.Ptr("status"),
					Storage:        nil,
					Version:        utils.Ptr("version"),
				},
			},
			&flavorModel{
				CPU: types.Int64Value(12),
				RAM: types.Int64Value(34),
			},
			&storageModel{
				Class: types.StringValue("class"),
				Size:  types.Int64Value(78),
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("name"),
				ACL: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("ip2"),
					types.StringValue(""),
					types.StringValue("ip1"),
				}),
				BackupSchedule: types.StringValue("schedule"),
				Flavor: types.ObjectValueMust(flavorTypes, map[string]attr.Value{
					"id":          types.StringNull(),
					"description": types.StringNull(),
					"cpu":         types.Int64Value(12),
					"ram":         types.Int64Value(34),
				}),
				Replicas: types.Int32Value(56),
				Storage: types.ObjectValueMust(storageTypes, map[string]attr.Value{
					"class": types.StringValue("class"),
					"size":  types.Int64Value(78),
				}),
				Version: types.StringValue("version"),
				Region:  types.StringValue(testRegion),
			},
			true,
		},
		{
			"nil_response",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			nil,
			&flavorModel{},
			&storageModel{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&postgresflex.InstanceResponse{},
			&flavorModel{},
			&storageModel{},
			testRegion,
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, &tt.state, tt.flavor, tt.storage, tt.region)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.state, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description  string
		input        *Model
		inputAcl     []string
		inputFlavor  *flavorModel
		inputStorage *storageModel
		expected     *postgresflex.CreateInstancePayload
		isValid      bool
	}{
		{
			"default_values",
			&Model{},
			[]string{},
			&flavorModel{},
			&storageModel{},
			&postgresflex.CreateInstancePayload{
				Acl: postgresflex.ACL{
					Items: []string{},
				},
				Storage: postgresflex.Storage{},
			},
			true,
		},
		{
			"simple_values",
			&Model{
				BackupSchedule: types.StringValue("schedule"),
				Name:           types.StringValue("name"),
				Replicas:       types.Int32Value(12),
				Version:        types.StringValue("version"),
			},
			[]string{
				"ip_1",
				"ip_2",
			},
			&flavorModel{
				Id: types.StringValue("flavor_id"),
			},
			&storageModel{
				Class: types.StringValue("class"),
				Size:  types.Int64Value(34),
			},
			&postgresflex.CreateInstancePayload{
				Acl: postgresflex.ACL{
					Items: []string{
						"ip_1",
						"ip_2",
					},
				},
				BackupSchedule: ("schedule"),
				FlavorId:       ("flavor_id"),
				Name:           ("name"),
				Replicas:       int32(12),
				Storage: postgresflex.Storage{
					Class: utils.Ptr("class"),
					Size:  utils.Ptr(int64(34)),
				},
				Version: "version",
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				BackupSchedule: types.StringNull(),
				Name:           types.StringNull(),
				Replicas:       types.Int32Value(2123456789),
				Version:        types.StringNull(),
			},
			[]string{
				"",
			},
			&flavorModel{
				Id: types.StringNull(),
			},
			&storageModel{
				Class: types.StringNull(),
				Size:  types.Int64Null(),
			},
			&postgresflex.CreateInstancePayload{
				Acl: postgresflex.ACL{
					Items: []string{
						"",
					},
				},
				BackupSchedule: "",
				FlavorId:       "",
				Name:           "",
				Replicas:       int32(2123456789),
				Storage: postgresflex.Storage{
					Class: nil,
					Size:  nil,
				},
				Version: "",
			},
			true,
		},
		{
			"nil_model",
			nil,
			[]string{},
			&flavorModel{},
			&storageModel{},
			nil,
			false,
		},
		{
			"nil_acl",
			&Model{},
			nil,
			&flavorModel{},
			&storageModel{},
			nil,
			false,
		},
		{
			"nil_flavor",
			&Model{},
			[]string{},
			nil,
			&storageModel{},
			nil,
			false,
		},
		{
			"nil_storage",
			&Model{},
			[]string{},
			&flavorModel{},
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input, tt.inputAcl, tt.inputFlavor, tt.inputStorage)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description  string
		input        *Model
		inputAcl     []string
		inputFlavor  *flavorModel
		inputStorage *storageModel
		expected     *postgresflex.PartialUpdateInstancePayload
		isValid      bool
	}{
		{
			"default_values",
			&Model{},
			[]string{},
			&flavorModel{},
			&storageModel{},
			&postgresflex.PartialUpdateInstancePayload{
				Acl: &postgresflex.ACL{
					Items: []string{},
				},
			},
			true,
		},
		{
			"simple_values",
			&Model{
				BackupSchedule: types.StringValue("schedule"),
				Name:           types.StringValue("name"),
				Replicas:       types.Int32Value(12),
				Version:        types.StringValue("version"),
			},
			[]string{
				"ip_1",
				"ip_2",
			},
			&flavorModel{
				Id: types.StringValue("flavor_id"),
			},
			&storageModel{
				Class: types.StringValue("class"),
				Size:  types.Int64Value(34),
			},
			&postgresflex.PartialUpdateInstancePayload{
				Acl: &postgresflex.ACL{
					Items: []string{
						"ip_1",
						"ip_2",
					},
				},
				BackupSchedule: utils.Ptr("schedule"),
				FlavorId:       utils.Ptr("flavor_id"),
				Name:           utils.Ptr("name"),
				Replicas:       utils.Ptr(int32(12)),
				Version:        utils.Ptr("version"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&Model{
				BackupSchedule: types.StringNull(),
				Name:           types.StringNull(),
				Replicas:       types.Int32Value(2123456789),
				Version:        types.StringNull(),
			},
			[]string{
				"",
			},
			&flavorModel{
				Id: types.StringNull(),
			},
			&storageModel{
				Class: types.StringNull(),
				Size:  types.Int64Null(),
			},
			&postgresflex.PartialUpdateInstancePayload{
				Acl: &postgresflex.ACL{
					Items: []string{
						"",
					},
				},
				BackupSchedule: nil,
				FlavorId:       nil,
				Name:           nil,
				Replicas:       utils.Ptr(int32(2123456789)),
				Version:        nil,
			},
			true,
		},
		{
			"nil_model",
			nil,
			[]string{},
			&flavorModel{},
			&storageModel{},
			nil,
			false,
		},
		{
			"nil_acl",
			&Model{},
			nil,
			&flavorModel{},
			&storageModel{},
			nil,
			false,
		},
		{
			"nil_flavor",
			&Model{},
			[]string{},
			nil,
			&storageModel{},
			nil,
			false,
		},
		{
			"nil_storage",
			&Model{},
			[]string{},
			&flavorModel{},
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(tt.input, tt.inputAcl, tt.inputFlavor, tt.inputStorage)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestLoadFlavorId(t *testing.T) {
	tests := []struct {
		description     string
		inputFlavor     *flavorModel
		mockedResp      *postgresflex.ListFlavorsResponse
		expected        *flavorModel
		getFlavorsFails bool
		isValid         bool
	}{
		{
			"ok_flavor",
			&flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			&postgresflex.ListFlavorsResponse{
				Flavors: []postgresflex.Flavor{
					{
						Id:          utils.Ptr("fid-1"),
						Cpu:         utils.Ptr(int64(2)),
						Description: utils.Ptr("description"),
						Memory:      utils.Ptr(int64(8)),
					},
				},
			},
			&flavorModel{
				Id:          types.StringValue("fid-1"),
				Description: types.StringValue("description"),
				CPU:         types.Int64Value(2),
				RAM:         types.Int64Value(8),
			},
			false,
			true,
		},
		{
			"ok_flavor_2",
			&flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			&postgresflex.ListFlavorsResponse{
				Flavors: []postgresflex.Flavor{
					{
						Id:          utils.Ptr("fid-1"),
						Cpu:         utils.Ptr(int64(2)),
						Description: utils.Ptr("description"),
						Memory:      utils.Ptr(int64(8)),
					},
					{
						Id:          utils.Ptr("fid-2"),
						Cpu:         utils.Ptr(int64(1)),
						Description: utils.Ptr("description"),
						Memory:      utils.Ptr(int64(4)),
					},
				},
			},
			&flavorModel{
				Id:          types.StringValue("fid-1"),
				Description: types.StringValue("description"),
				CPU:         types.Int64Value(2),
				RAM:         types.Int64Value(8),
			},
			false,
			true,
		},
		{
			"no_matching_flavor",
			&flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			&postgresflex.ListFlavorsResponse{
				Flavors: []postgresflex.Flavor{
					{
						Id:          utils.Ptr("fid-1"),
						Cpu:         utils.Ptr(int64(1)),
						Description: utils.Ptr("description"),
						Memory:      utils.Ptr(int64(8)),
					},
					{
						Id:          utils.Ptr("fid-2"),
						Cpu:         utils.Ptr(int64(1)),
						Description: utils.Ptr("description"),
						Memory:      utils.Ptr(int64(4)),
					},
				},
			},
			&flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			false,
			false,
		},
		{
			"nil_response",
			&flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			&postgresflex.ListFlavorsResponse{},
			&flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			false,
			false,
		},
		{
			"error_response",
			&flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			&postgresflex.ListFlavorsResponse{},
			&flavorModel{
				CPU: types.Int64Value(2),
				RAM: types.Int64Value(8),
			},
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			//client := &postgresFlexClientMocked{
			//	returnError:    tt.getFlavorsFails,
			//	getFlavorsResp: tt.mockedResp,
			//}
			client := newApiMock(tt.getFlavorsFails, tt.mockedResp)

			model := &Model{
				ProjectId: types.StringValue("pid"),
			}
			flavorModel := &flavorModel{
				CPU: tt.inputFlavor.CPU,
				RAM: tt.inputFlavor.RAM,
			}
			err := loadFlavorId(context.Background(), client, model, flavorModel)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(flavorModel, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
