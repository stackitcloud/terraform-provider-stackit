package postgresFlexAlphaFlavor

import (
	"context"
	"testing"

	postgresflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
)

type mockRequest struct {
	executeFunc func() (*postgresflex.GetFlavorsResponse, error)
}

func (m *mockRequest) Page(_ int64) postgresflex.ApiGetFlavorsRequestRequest { return m }
func (m *mockRequest) Size(_ int64) postgresflex.ApiGetFlavorsRequestRequest { return m }
func (m *mockRequest) Sort(_ postgresflex.FlavorSort) postgresflex.ApiGetFlavorsRequestRequest {
	return m
}
func (m *mockRequest) Execute() (*postgresflex.GetFlavorsResponse, error) {
	return m.executeFunc()
}

type mockFlavorsClient struct {
	executeRequest func() postgresflex.ApiGetFlavorsRequestRequest
}

func (m *mockFlavorsClient) GetFlavorsRequest(_ context.Context, _, _ string) postgresflex.ApiGetFlavorsRequestRequest {
	return m.executeRequest()
}

var mockResp = func(page int64) (*postgresflex.GetFlavorsResponse, error) {
	if page == 1 {
		return &postgresflex.GetFlavorsResponse{
			Flavors: &[]postgresflex.ListFlavors{
				{Id: utils.Ptr("flavor-1"), Description: utils.Ptr("first")},
				{Id: utils.Ptr("flavor-2"), Description: utils.Ptr("second")},
			},
		}, nil
	}
	if page == 2 {
		return &postgresflex.GetFlavorsResponse{
			Flavors: &[]postgresflex.ListFlavors{
				{Id: utils.Ptr("flavor-3"), Description: utils.Ptr("three")},
			},
		}, nil
	}

	return &postgresflex.GetFlavorsResponse{
		Flavors: &[]postgresflex.ListFlavors{},
	}, nil
}

func TestGetFlavorsByFilter(t *testing.T) {
	tests := []struct {
		description string
		projectId   string
		region      string
		mockErr     error
		filter      func(postgresflex.ListFlavors) bool
		wantCount   int
		wantErr     bool
	}{
		{
			description: "Success - Get all flavors (2 pages)",
			projectId:   "pid", region: "reg",
			filter:    func(_ postgresflex.ListFlavors) bool { return true },
			wantCount: 3,
			wantErr:   false,
		},
		{
			description: "Success - Filter flavors by description",
			projectId:   "pid", region: "reg",
			filter:    func(f postgresflex.ListFlavors) bool { return *f.Description == "first" },
			wantCount: 1,
			wantErr:   false,
		},
		{
			description: "Error - Missing parameters",
			projectId:   "", region: "reg",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.description, func(t *testing.T) {
				var currentPage int64
				client := &mockFlavorsClient{
					executeRequest: func() postgresflex.ApiGetFlavorsRequestRequest {
						return &mockRequest{
							executeFunc: func() (*postgresflex.GetFlavorsResponse, error) {
								currentPage++
								return mockResp(currentPage)
							},
						}
					},
				}
				actual, err := getFlavorsByFilter(context.Background(), client, tt.projectId, tt.region, tt.filter)

				if (err != nil) != tt.wantErr {
					t.Errorf("getFlavorsByFilter() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				if !tt.wantErr && len(actual) != tt.wantCount {
					t.Errorf("getFlavorsByFilter() got %d flavors, want %d", len(actual), tt.wantCount)
				}
			},
		)
	}
}

func TestGetAllFlavors(t *testing.T) {
	var currentPage int64
	client := &mockFlavorsClient{
		executeRequest: func() postgresflex.ApiGetFlavorsRequestRequest {
			return &mockRequest{
				executeFunc: func() (*postgresflex.GetFlavorsResponse, error) {
					currentPage++
					return mockResp(currentPage)
				},
			}
		},
	}

	res, err := getAllFlavors(context.Background(), client, "pid", "reg")
	if err != nil {
		t.Errorf("getAllFlavors() unexpected error: %v", err)
	}
	if len(res) != 3 {
		t.Errorf("getAllFlavors() expected 3 flavor, got %d", len(res))
	}
}
