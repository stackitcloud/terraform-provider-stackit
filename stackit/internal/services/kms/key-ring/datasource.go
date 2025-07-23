package kms

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/stackitcloud/stackit-sdk-go/services/kms"
)

var (
	_ datasource.DataSource = &keyRingDataSource{}
)

func NewKeyRingDataSource() datasource.DataSource {
	return &keyRingDataSource{}
}

type keyRingDataSource struct {
	client *kms.APIClient
}

func (k keyRingDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	//TODO implement me
	panic("implement me")
}

func (k keyRingDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	//TODO implement me
	panic("implement me")
}

func (k keyRingDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	//TODO implement me
	panic("implement me")
}
