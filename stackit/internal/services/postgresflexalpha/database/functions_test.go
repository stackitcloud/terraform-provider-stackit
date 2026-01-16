package postgresflexalpha

import (
	"context"
	"testing"

	postgresflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
)

type mockRequest struct {
	executeFunc func() (*postgresflex.ListDatabasesResponse, error)
}

func (m *mockRequest) Page(_ int64) postgresflex.ApiListDatabasesRequestRequest { return m }
func (m *mockRequest) Size(_ int64) postgresflex.ApiListDatabasesRequestRequest { return m }
func (m *mockRequest) Sort(_ postgresflex.DatabaseSort) postgresflex.ApiListDatabasesRequestRequest {
	return m
}
func (m *mockRequest) Execute() (*postgresflex.ListDatabasesResponse, error) {
	return m.executeFunc()
}

type mockDBClient struct {
	executeRequest func() postgresflex.ApiListDatabasesRequestRequest
}

var _ databaseClientReader = (*mockDBClient)(nil)

func (m *mockDBClient) ListDatabasesRequest(
	_ context.Context,
	_, _, _ string,
) postgresflex.ApiListDatabasesRequestRequest {
	return m.executeRequest()
}

func TestGetDatabase(t *testing.T) {
	mockResp := func(page int64) (*postgresflex.ListDatabasesResponse, error) {
		if page == 1 {
			return &postgresflex.ListDatabasesResponse{
				Databases: &[]postgresflex.ListDatabase{
					{Id: utils.Ptr(int64(1)), Name: utils.Ptr("first")},
					{Id: utils.Ptr(int64(2)), Name: utils.Ptr("second")},
				},
				Pagination: &postgresflex.Pagination{
					Page:       utils.Ptr(int64(1)),
					TotalPages: utils.Ptr(int64(2)),
					Size:       utils.Ptr(int64(3)),
				},
			}, nil
		}

		if page == 2 {
			return &postgresflex.ListDatabasesResponse{
				Databases: &[]postgresflex.ListDatabase{{Id: utils.Ptr(int64(3)), Name: utils.Ptr("three")}},
				Pagination: &postgresflex.Pagination{
					Page:       utils.Ptr(int64(2)),
					TotalPages: utils.Ptr(int64(2)),
					Size:       utils.Ptr(int64(3)),
				},
			}, nil
		}

		return &postgresflex.ListDatabasesResponse{
			Databases: &[]postgresflex.ListDatabase{},
			Pagination: &postgresflex.Pagination{
				Page:       utils.Ptr(int64(3)),
				TotalPages: utils.Ptr(int64(2)),
				Size:       utils.Ptr(int64(3)),
			},
		}, nil
	}

	tests := []struct {
		description string
		projectId   string
		region      string
		instanceId  string
		wantErr     bool
		wantDbName  string
		wantDbId    int64
	}{
		{
			description: "Success - Found by name on first page",
			projectId:   "pid", region: "reg", instanceId: "inst",
			wantErr:    false,
			wantDbName: "second",
		},
		{
			description: "Success - Found by id on first page",
			projectId:   "pid", region: "reg", instanceId: "inst",
			wantErr:  false,
			wantDbId: 2,
		},
		{
			description: "Success - Found by name on second page",
			projectId:   "pid", region: "reg", instanceId: "inst",
			wantErr:    false,
			wantDbName: "three",
		},
		{
			description: "Success - Found by id on second page",
			projectId:   "pid", region: "reg", instanceId: "inst",
			wantErr:  false,
			wantDbId: 1,
		},
		{
			description: "Error - API failure",
			projectId:   "pid", region: "reg", instanceId: "inst",
			wantErr: true,
		},
		{
			description: "Error - Missing parameters",
			projectId:   "", region: "reg", instanceId: "inst",
			wantErr: true,
		},
		{
			description: "Error - Search by name not found after all pages",
			projectId:   "pid", region: "reg", instanceId: "inst",
			wantDbName: "non-existent",
			wantErr:    true,
		},
		{
			description: "Error - Search by id not found after all pages",
			projectId:   "pid", region: "reg", instanceId: "inst",
			wantDbId: 999999,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.description, func(t *testing.T) {
				var currentPage int64
				client := &mockDBClient{
					executeRequest: func() postgresflex.ApiListDatabasesRequestRequest {
						return &mockRequest{
							executeFunc: func() (*postgresflex.ListDatabasesResponse, error) {
								currentPage++
								return mockResp(currentPage)
							},
						}
					},
				}

				var actual *postgresflex.ListDatabase
				var errDB error

				if tt.wantDbName != "" {
					actual, errDB = getDatabaseByName(
						t.Context(),
						client,
						tt.projectId,
						tt.region,
						tt.instanceId,
						tt.wantDbName,
					)
				} else if tt.wantDbId != 0 {
					actual, errDB = getDatabaseById(
						t.Context(),
						client,
						tt.projectId,
						tt.region,
						tt.instanceId,
						tt.wantDbId,
					)
				} else {
					actual, errDB = getDatabase(
						context.Background(),
						client,
						tt.projectId,
						tt.region,
						tt.instanceId,
						func(_ postgresflex.ListDatabase) bool { return false },
					)
				}

				if (errDB != nil) != tt.wantErr {
					t.Errorf("getDatabase() error = %v, wantErr %v", errDB, tt.wantErr)
					return
				}
				if !tt.wantErr && tt.wantDbName != "" && actual != nil {
					if *actual.Name != tt.wantDbName {
						t.Errorf("getDatabase() got name = %v, want %v", *actual.Name, tt.wantDbName)
					}
				}

				if !tt.wantErr && tt.wantDbId != 0 && actual != nil {
					if *actual.Id != tt.wantDbId {
						t.Errorf("getDatabase() got id = %v, want %v", *actual.Id, tt.wantDbId)
					}
				}
			},
		)
	}
}
