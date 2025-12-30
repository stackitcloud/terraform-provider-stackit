package postgresflexalpha

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	postgresflex "github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
)

type postgresFlexClientMocked struct {
	returnError bool
	firstItem   int
	lastItem    int
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
	{
		Cpu:         1,
		Description: "flavor 1.2",
		Id:          "flv1.2",
		MaxGB:       500,
		Memory:      2,
		MinGB:       5,
		NodeType:    "single",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         1,
		Description: "flavor 1.3",
		Id:          "flv1.3",
		MaxGB:       500,
		Memory:      3,
		MinGB:       5,
		NodeType:    "single",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         1,
		Description: "flavor 1.4",
		Id:          "flv1.4",
		MaxGB:       500,
		Memory:      4,
		MinGB:       5,
		NodeType:    "single",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         1,
		Description: "flavor 1.5",
		Id:          "flv1.5",
		MaxGB:       500,
		Memory:      5,
		MinGB:       5,
		NodeType:    "single",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         1,
		Description: "flavor 1.6",
		Id:          "flv1.6",
		MaxGB:       500,
		Memory:      6,
		MinGB:       5,
		NodeType:    "single",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         1,
		Description: "flavor 1.7",
		Id:          "flv1.7",
		MaxGB:       500,
		Memory:      7,
		MinGB:       5,
		NodeType:    "single",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         1,
		Description: "flavor 1.8",
		Id:          "flv1.8",
		MaxGB:       500,
		Memory:      8,
		MinGB:       5,
		NodeType:    "single",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         1,
		Description: "flavor 1.9",
		Id:          "flv1.9",
		MaxGB:       500,
		Memory:      9,
		MinGB:       5,
		NodeType:    "single",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	/* ......................................................... */
	{
		Cpu:         2,
		Description: "flavor 2.1",
		Id:          "flv2.1",
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
	{
		Cpu:         2,
		Description: "flavor 2.2",
		Id:          "flv2.2",
		MaxGB:       500,
		Memory:      2,
		MinGB:       5,
		NodeType:    "single",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         2,
		Description: "flavor 2.3",
		Id:          "flv2.3",
		MaxGB:       500,
		Memory:      3,
		MinGB:       5,
		NodeType:    "single",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         2,
		Description: "flavor 2.4",
		Id:          "flv2.4",
		MaxGB:       500,
		Memory:      4,
		MinGB:       5,
		NodeType:    "single",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         2,
		Description: "flavor 2.5",
		Id:          "flv2.5",
		MaxGB:       500,
		Memory:      5,
		MinGB:       5,
		NodeType:    "single",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         2,
		Description: "flavor 2.6",
		Id:          "flv2.6",
		MaxGB:       500,
		Memory:      6,
		MinGB:       5,
		NodeType:    "single",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	/* ......................................................... */
	{
		Cpu:         1,
		Description: "flavor 1.1",
		Id:          "flv1.1",
		MaxGB:       500,
		Memory:      1,
		MinGB:       5,
		NodeType:    "replica",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         1,
		Description: "flavor 1.2",
		Id:          "flv1.2",
		MaxGB:       500,
		Memory:      2,
		MinGB:       5,
		NodeType:    "replica",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         1,
		Description: "flavor 1.3",
		Id:          "flv1.3",
		MaxGB:       500,
		Memory:      3,
		MinGB:       5,
		NodeType:    "replica",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         1,
		Description: "flavor 1.4",
		Id:          "flv1.4",
		MaxGB:       500,
		Memory:      4,
		MinGB:       5,
		NodeType:    "replica",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         1,
		Description: "flavor 1.5",
		Id:          "flv1.5",
		MaxGB:       500,
		Memory:      5,
		MinGB:       5,
		NodeType:    "replica",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         1,
		Description: "flavor 1.6",
		Id:          "flv1.6",
		MaxGB:       500,
		Memory:      6,
		MinGB:       5,
		NodeType:    "replica",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	/* ......................................................... */
	{
		Cpu:         2,
		Description: "flavor 2.1",
		Id:          "flv2.1",
		MaxGB:       500,
		Memory:      1,
		MinGB:       5,
		NodeType:    "replica",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         2,
		Description: "flavor 2.2",
		Id:          "flv2.2",
		MaxGB:       500,
		Memory:      2,
		MinGB:       5,
		NodeType:    "replica",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         2,
		Description: "flavor 2.3",
		Id:          "flv2.3",
		MaxGB:       500,
		Memory:      3,
		MinGB:       5,
		NodeType:    "replica",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         2,
		Description: "flavor 2.4",
		Id:          "flv2.4",
		MaxGB:       500,
		Memory:      4,
		MinGB:       5,
		NodeType:    "replica",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         2,
		Description: "flavor 2.5",
		Id:          "flv2.5",
		MaxGB:       500,
		Memory:      5,
		MinGB:       5,
		NodeType:    "replica",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	{
		Cpu:         2,
		Description: "flavor 2.6",
		Id:          "flv2.6",
		MaxGB:       500,
		Memory:      6,
		MinGB:       5,
		NodeType:    "replica",
		StorageClasses: []testFlavorStorageClass{
			{Class: "sc1", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc2", MaxIoPerSec: 0, MaxThroughInMb: 0},
			{Class: "sc3", MaxIoPerSec: 0, MaxThroughInMb: 0},
		},
	},
	/* ......................................................... */
}

func testFlavorListToResponseFlavorList(f []testFlavor) []postgresflex.ListFlavors {
	result := make([]postgresflex.ListFlavors, len(f))
	for i, flavor := range f {
		result[i] = testFlavorToResponseFlavor(flavor)
	}
	return result
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

func (c postgresFlexClientMocked) GetFlavorsRequestExecute(
	_ context.Context,
	_, _ string,
	page, size *int64,
	_ *postgresflex.FlavorSort,
) (*postgresflex.GetFlavorsResponse, error) {
	if c.returnError {
		return nil, fmt.Errorf("get flavors failed")
	}

	var res postgresflex.GetFlavorsResponse
	var resFlavors []postgresflex.ListFlavors

	myList := responseList[c.firstItem : c.lastItem+1]

	firstItem := *page**size - *size
	if firstItem > int64(len(myList)) {
		firstItem = int64(len(myList))
	}

	lastItem := firstItem + *size
	if lastItem > int64(len(myList)) {
		lastItem = int64(len(myList))
	}

	for _, flv := range myList[firstItem:lastItem] {
		resFlavors = append(resFlavors, testFlavorToResponseFlavor(flv))
	}

	res.Flavors = &resFlavors
	res.Pagination = &postgresflex.Pagination{
		Page:       page,
		Size:       size,
		Sort:       utils.Ptr("id.asc"),
		TotalPages: utils.Ptr(int64(1)),
		TotalRows:  utils.Ptr(int64(len(myList))),
	}

	return &res, nil
}

func Test_getAllFlavors(t *testing.T) {
	type args struct {
		projectId string
		region    string
	}
	tests := []struct {
		name      string
		args      args
		firstItem int
		lastItem  int
		want      []postgresflex.ListFlavors
		wantErr   bool
	}{
		{
			name: "find exactly one flavor",
			args: args{
				projectId: "project",
				region:    "region",
			},
			firstItem: 0,
			lastItem:  0,
			want: []postgresflex.ListFlavors{
				testFlavorToResponseFlavor(responseList[0]),
			},
			wantErr: false,
		},
		{
			name: "get exactly 1 page flavors",
			args: args{
				projectId: "project",
				region:    "region",
			},
			firstItem: 0,
			lastItem:  9,
			want:      testFlavorListToResponseFlavorList(responseList[0:10]),
			wantErr:   false,
		},
		{
			name: "get exactly 20 flavors",
			args: args{
				projectId: "project",
				region:    "region",
			},
			firstItem: 0,
			lastItem:  20,
			// 0 indexed therefore we want :21
			want:    testFlavorListToResponseFlavorList(responseList[0:21]),
			wantErr: false,
		},
		{
			name: "get all flavors",
			args: args{
				projectId: "project",
				region:    "region",
			},
			firstItem: 0,
			// we take care of max value at another place
			lastItem: 20000,
			// 0 indexed therefore we want :21
			want:    testFlavorListToResponseFlavorList(responseList),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := tt.firstItem
			if first > len(responseList)-1 {
				first = len(responseList) - 1
			}
			last := tt.lastItem
			if last > len(responseList)-1 {
				last = len(responseList) - 1
			}
			mockClient := postgresFlexClientMocked{
				returnError: tt.wantErr,
				firstItem:   first,
				lastItem:    last,
			}
			got, err := getAllFlavors(context.TODO(), mockClient, tt.args.projectId, tt.args.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAllFlavors() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAllFlavors() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_loadFlavorId(t *testing.T) {
	type args struct {
		ctx     context.Context
		model   *Model
		flavor  *flavorModel
		storage *storageModel
	}
	tests := []struct {
		name      string
		args      args
		firstItem int
		lastItem  int
		want      []postgresflex.ListFlavors
		wantErr   bool
	}{
		{
			name: "find a single flavor",
			args: args{
				ctx: context.Background(),
				model: &Model{
					ProjectId: basetypes.NewStringValue("project"),
					Region:    basetypes.NewStringValue("region"),
				},
				flavor: &flavorModel{
					CPU:      basetypes.NewInt64Value(1),
					RAM:      basetypes.NewInt64Value(1),
					NodeType: basetypes.NewStringValue("Single"),
				},
				storage: &storageModel{
					Class: basetypes.NewStringValue("sc1"),
					Size:  basetypes.NewInt64Value(100),
				},
			},
			firstItem: 0,
			lastItem:  3,
			want: []postgresflex.ListFlavors{
				testFlavorToResponseFlavor(responseList[0]),
			},
			wantErr: false,
		},
		{
			name: "find a single flavor by replicas option",
			args: args{
				ctx: context.Background(),
				model: &Model{
					ProjectId: basetypes.NewStringValue("project"),
					Region:    basetypes.NewStringValue("region"),
					Replicas:  basetypes.NewInt64Value(1),
				},
				flavor: &flavorModel{
					CPU: basetypes.NewInt64Value(1),
					RAM: basetypes.NewInt64Value(1),
				},
				storage: &storageModel{
					Class: basetypes.NewStringValue("sc1"),
					Size:  basetypes.NewInt64Value(100),
				},
			},
			firstItem: 0,
			lastItem:  3,
			want: []postgresflex.ListFlavors{
				testFlavorToResponseFlavor(responseList[0]),
			},
			wantErr: false,
		},
		{
			name: "fail finding find a single flavor by replicas option",
			args: args{
				ctx: context.Background(),
				model: &Model{
					ProjectId: basetypes.NewStringValue("project"),
					Region:    basetypes.NewStringValue("region"),
					Replicas:  basetypes.NewInt64Value(1),
				},
				flavor: &flavorModel{
					CPU: basetypes.NewInt64Value(1),
					RAM: basetypes.NewInt64Value(1),
				},
				storage: &storageModel{
					Class: basetypes.NewStringValue("sc1"),
					Size:  basetypes.NewInt64Value(100),
				},
			},
			firstItem: 13,
			lastItem:  23,
			want:      []postgresflex.ListFlavors{},
			wantErr:   true,
		},
		{
			name: "find a replicas flavor",
			args: args{
				ctx: context.Background(),
				model: &Model{
					ProjectId: basetypes.NewStringValue("project"),
					Region:    basetypes.NewStringValue("region"),
				},
				flavor: &flavorModel{
					CPU:      basetypes.NewInt64Value(1),
					RAM:      basetypes.NewInt64Value(1),
					NodeType: basetypes.NewStringValue("Replica"),
				},
				storage: &storageModel{
					Class: basetypes.NewStringValue("sc1"),
					Size:  basetypes.NewInt64Value(100),
				},
			},
			firstItem: 0,
			lastItem:  len(responseList) - 1,
			want: []postgresflex.ListFlavors{
				testFlavorToResponseFlavor(responseList[11]),
			},
			wantErr: false,
		},
		{
			name: "find a replicas flavor by replicas option",
			args: args{
				ctx: context.Background(),
				model: &Model{
					ProjectId: basetypes.NewStringValue("project"),
					Region:    basetypes.NewStringValue("region"),
					Replicas:  basetypes.NewInt64Value(3),
				},
				flavor: &flavorModel{
					CPU: basetypes.NewInt64Value(1),
					RAM: basetypes.NewInt64Value(1),
				},
				storage: &storageModel{
					Class: basetypes.NewStringValue("sc1"),
					Size:  basetypes.NewInt64Value(100),
				},
			},
			firstItem: 0,
			lastItem:  len(responseList) - 1,
			want: []postgresflex.ListFlavors{
				testFlavorToResponseFlavor(responseList[11]),
			},
			wantErr: false,
		},
		{
			name: "fail finding a replica flavor",
			args: args{
				ctx: context.Background(),
				model: &Model{
					ProjectId: basetypes.NewStringValue("project"),
					Region:    basetypes.NewStringValue("region"),
					Replicas:  basetypes.NewInt64Value(3),
				},
				flavor: &flavorModel{
					CPU: basetypes.NewInt64Value(1),
					RAM: basetypes.NewInt64Value(1),
				},
				storage: &storageModel{
					Class: basetypes.NewStringValue("sc1"),
					Size:  basetypes.NewInt64Value(100),
				},
			},
			firstItem: 0,
			lastItem:  10,
			want:      []postgresflex.ListFlavors{},
			wantErr:   true,
		},
		{
			name: "no flavor found error",
			args: args{
				ctx: context.Background(),
				model: &Model{
					ProjectId: basetypes.NewStringValue("project"),
					Region:    basetypes.NewStringValue("region"),
				},
				flavor: &flavorModel{
					CPU:      basetypes.NewInt64Value(10),
					RAM:      basetypes.NewInt64Value(1000),
					NodeType: basetypes.NewStringValue("Single"),
				},
				storage: &storageModel{
					Class: basetypes.NewStringValue("sc1"),
					Size:  basetypes.NewInt64Value(100),
				},
			},
			firstItem: 0,
			lastItem:  3,
			want:      []postgresflex.ListFlavors{},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := tt.firstItem
			if first > len(responseList)-1 {
				first = len(responseList) - 1
			}
			last := tt.lastItem
			if last > len(responseList)-1 {
				last = len(responseList) - 1
			}
			mockClient := postgresFlexClientMocked{
				returnError: tt.wantErr,
				firstItem:   first,
				lastItem:    last,
			}
			if err := loadFlavorId(tt.args.ctx, mockClient, tt.args.model, tt.args.flavor, tt.args.storage); (err != nil) != tt.wantErr {
				t.Errorf("loadFlavorId() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
