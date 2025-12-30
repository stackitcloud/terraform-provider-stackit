package postgresflexalpha

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	postgresflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
)

type postgresFlexClientMocked struct {
	returnError bool
}

type testFlavor struct {
	Cpu            int64
	Description    string
	Id             string
	MaxGB          int64
	Memory         int64
	MinGB          int64
	NodeType       string
	StorageClasses []testFlavorStorageClass
}

type testFlavorStorageClass struct {
	Class          string
	MaxIoPerSec    int64
	MaxThroughInMb int64
}

var responseList = []testFlavor{
	{
		Cpu:         1,
		Description: "flavor 1.1",
		Id:          "flv1.1",
		MaxGB:       500,
		Memory:      1,
		MinGB:       5,
		NodeType:    "single",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
}

func testFlavorToResponseFlavor(f testFlavor) postgresflex.ListFlavors {
	var scList []postgresflex.FlavorStorageClassesStorageClass
	for _, fl := range f.StorageClasses {
		scList = append(scList, postgresflex.FlavorStorageClassesStorageClass{
			Class:          utils.Ptr(fl.Class),
			MaxIoPerSec:    utils.Ptr(fl.MaxIoPerSec),
			MaxThroughInMb: utils.Ptr(fl.MaxThroughInMb),
		})
	}
	return postgresflex.ListFlavors{
		Cpu:            utils.Ptr(f.Cpu),
		Description:    utils.Ptr(f.Description),
		Id:             utils.Ptr(f.Id),
		MaxGB:          utils.Ptr(f.MaxGB),
		Memory:         utils.Ptr(f.Memory),
		MinGB:          utils.Ptr(f.MinGB),
		NodeType:       utils.Ptr(f.NodeType),
		StorageClasses: &scList,
	}
}

func (c postgresFlexClientMocked) GetFlavorsRequestExecute(_ context.Context, _, _ string, _ *int64, _ *int64, _ *postgresflex.FlavorSort) (*postgresflex.GetFlavorsResponse, error) {
	if c.returnError {
		return nil, fmt.Errorf("get flavors failed")
	}

	var res postgresflex.GetFlavorsResponse
	var resFlavors []postgresflex.ListFlavors

	for _, flv := range responseList {
		resFlavors = append(resFlavors, testFlavorToResponseFlavor(flv))
	}

	res.Flavors = &resFlavors
	res.Pagination = &postgresflex.Pagination{
		Page:       utils.Ptr(int64(1)),
		Size:       utils.Ptr(int64(10)),
		Sort:       utils.Ptr("id.asc"),
		TotalPages: utils.Ptr(int64(1)),
		TotalRows:  utils.Ptr(int64(len(responseList))),
	}

	return &res, nil
}

func Test_getAllFlavors(t *testing.T) {
	type args struct {
		projectId string
		region    string
	}
	tests := []struct {
		name    string
		args    args
		want    []postgresflex.ListFlavors
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				projectId: "project",
				region:    "region",
			},
			want: []postgresflex.ListFlavors{
				testFlavorToResponseFlavor(responseList[0]),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := postgresFlexClientMocked{
				returnError: tt.wantErr,
			}
			got, err := getAllFlavors(context.TODO(), mockClient, tt.args.projectId, tt.args.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAllFlavors() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAllFlavors() got = %v, want %v", got, tt.want)
			}
		})
	}
}
