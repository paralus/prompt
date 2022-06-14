package mock

import (
	"context"

	userrpc "github.com/paralus/paralus/proto/rpc/user"
	commonv3 "github.com/paralus/paralus/proto/types/commonpb/v3"
	userv3 "github.com/paralus/paralus/proto/types/userpb/v3"
	"google.golang.org/grpc"
)

type UGPool struct{}

func (p *UGPool) Close() {}

func (p *UGPool) NewClient(ctx context.Context) (userrpc.UGClient, error) {
	return &UGClient{}, nil
}

type UGClient struct {
	userrpc.UserClient
}

func (kcc UGClient) GetUser(ctx context.Context, in *userv3.User, opts ...grpc.CallOption) (*userv3.User, error) {
	return &userv3.User{
		Metadata: &commonv3.Metadata{Name: in.Metadata.Name},
	}, nil
}

func (c *UGClient) Unhealthy() {}

func (c *UGClient) Close() error {
	return nil
}
