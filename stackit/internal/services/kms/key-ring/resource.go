package kms

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stackitcloud/stackit-sdk-go/services/kms"
)

type Model struct {
}

func NewKeyRingResource() resource.Resource {
	return &keyRingResource{}
}

type keyRingResource struct {
	client *kms.APIClient
}

func (k keyRingResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	//TODO implement me
	panic("implement me")
}

func (k keyRingResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	//TODO implement me
	panic("implement me")
}

func (k keyRingResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	//TODO implement me
	panic("implement me")
}

func (k keyRingResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	//TODO implement me
	panic("implement me")
}

func (k keyRingResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	//TODO implement me
	panic("implement me")
}

func (k keyRingResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	//TODO implement me
	panic("implement me")
}
