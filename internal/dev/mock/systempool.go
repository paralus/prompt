package mock

import (
	"context"

	systemrpc "github.com/paralus/paralus/proto/rpc/system"
	commonv3 "github.com/paralus/paralus/proto/types/commonpb/v3"
	systemv3 "github.com/paralus/paralus/proto/types/systempb/v3"
	"google.golang.org/grpc"
)

type SystemPool struct{}

func (p *SystemPool) Close() {}

func (p *SystemPool) NewClient(ctx context.Context) (systemrpc.SystemClient, error) {
	return &SystemClient{}, nil
}

type SystemClient struct {
	systemrpc.ProjectClient
	systemrpc.OrganizationClient
	systemrpc.PartnerClient
}

func (kcc SystemClient) GetProject(ctx context.Context, in *systemv3.Project, opts ...grpc.CallOption) (*systemv3.Project, error) {
	return &systemv3.Project{
		Metadata: &commonv3.Metadata{
			Name:         "dummyproject",
			Organization: "dummyorganization",
			Partner:      "dummypartner",
		},
	}, nil
}

func (kcc SystemClient) GetOrganization(ctx context.Context, in *systemv3.Organization, opts ...grpc.CallOption) (*systemv3.Organization, error) {
	return &systemv3.Organization{
		Metadata: &commonv3.Metadata{
			Name:    "dummyorganization",
			Partner: "dummypartner",
		},
	}, nil
}

func (kcc SystemClient) GetPartner(ctx context.Context, in *systemv3.Partner, opts ...grpc.CallOption) (*systemv3.Partner, error) {
	return &systemv3.Partner{
		Metadata: &commonv3.Metadata{Name: "dummypartner"},
	}, nil
}

func (c *SystemClient) Unhealthy() {}

func (c *SystemClient) Close() error {
	return nil
}
