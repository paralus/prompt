package mock

import (
	"context"
	"os"
	"path"

	sentryrpcv2 "github.com/paralus/paralus/proto/rpc/sentry"
	commnopbv3 "github.com/paralus/paralus/proto/types/commonpb/v3"
	commonv3 "github.com/paralus/paralus/proto/types/commonpb/v3"
	"google.golang.org/grpc"
)

type SentryPool struct{}

func (p *SentryPool) Close() {}

func (p *SentryPool) NewClient(ctx context.Context) (sentryrpcv2.SentryClient, error) {
	return &SentryClient{}, nil
}

type SentryClient struct {
	sentryrpcv2.BootstrapClient
	sentryrpcv2.ClusterAuthorizationClient
	KubeConfigClient
}

func (c *SentryClient) Unhealthy() {}

func (c *SentryClient) Close() error {
	return nil
}

type KubeConfigClient struct{}

func (kcc KubeConfigClient) GetForClusterWebSession(ctx context.Context, in *sentryrpcv2.GetForClusterRequest, opts ...grpc.CallOption) (*commnopbv3.HttpBody, error) {
	d, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path.Join(d, "kubeconfig.yaml"))
	if err != nil {
		return nil, err
	}
	return &commonv3.HttpBody{
		Data: data,
	}, nil
}

func (kcc KubeConfigClient) GetForClusterSystemSession(ctx context.Context, in *sentryrpcv2.GetForClusterRequest, opts ...grpc.CallOption) (*commnopbv3.HttpBody, error) {
	return nil, nil
}
func (kcc KubeConfigClient) GetForUser(ctx context.Context, in *sentryrpcv2.GetForUserRequest, opts ...grpc.CallOption) (*commnopbv3.HttpBody, error) {
	return nil, nil
}
func (kcc KubeConfigClient) RevokeKubeconfig(ctx context.Context, in *sentryrpcv2.RevokeKubeconfigRequest, opts ...grpc.CallOption) (*sentryrpcv2.RevokeKubeconfigResponse, error) {
	return nil, nil
}
func (kcc KubeConfigClient) GetOrganizationSetting(ctx context.Context, in *sentryrpcv2.GetKubeconfigSettingRequest, opts ...grpc.CallOption) (*sentryrpcv2.GetKubeconfigSettingResponse, error) {
	return nil, nil
}
func (kcc KubeConfigClient) GetUserSetting(ctx context.Context, in *sentryrpcv2.GetKubeconfigSettingRequest, opts ...grpc.CallOption) (*sentryrpcv2.GetKubeconfigSettingResponse, error) {
	return nil, nil
}
func (kcc KubeConfigClient) GetSSOUserSetting(ctx context.Context, in *sentryrpcv2.GetKubeconfigSettingRequest, opts ...grpc.CallOption) (*sentryrpcv2.GetKubeconfigSettingResponse, error) {
	return nil, nil
}
func (kcc KubeConfigClient) UpdateOrganizationSetting(ctx context.Context, in *sentryrpcv2.UpdateKubeconfigSettingRequest, opts ...grpc.CallOption) (*sentryrpcv2.UpdateKubeconfigSettingResponse, error) {
	return nil, nil
}
func (kcc KubeConfigClient) UpdateUserSetting(ctx context.Context, in *sentryrpcv2.UpdateKubeconfigSettingRequest, opts ...grpc.CallOption) (*sentryrpcv2.UpdateKubeconfigSettingResponse, error) {
	return nil, nil
}
func (kcc KubeConfigClient) UpdateSSOUserSetting(ctx context.Context, in *sentryrpcv2.UpdateKubeconfigSettingRequest, opts ...grpc.CallOption) (*sentryrpcv2.UpdateKubeconfigSettingResponse, error) {
	return nil, nil
}
