package v2

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	logv2 "github.com/RafaySystems/rafay-common/pkg/log/v2"
	ctypesv2 "github.com/RafaySystems/rafay-common/pkg/types/v2"
	rpcv2 "github.com/RafaySystems/ztka/components/prompt/proto/rpc/v2"
	sentryrpcv2 "github.com/RafaySystems/rafay-sentry/proto/rpc/v2"
	"github.com/pkg/errors"
	"github.com/rs/xid"
)

type virtualMachineServer struct {
	sp      sentryrpcv2.SentryPool
	tmpPath string
}

var (
	_log = logv2.GetLogger()
)

var _ rpcv2.VirtualMachineServer = (*virtualMachineServer)(nil)

// NewPipelineServer returns new placement server implementation
func NewVirtualMachineServer(sentryPool sentryrpcv2.SentryPool, tmpPath string) rpcv2.VirtualMachineServer {
	return &virtualMachineServer{
		sp:      sentryPool,
		tmpPath: tmpPath,
	}
}

func (vms *virtualMachineServer) StartVM(ctx context.Context, req *rpcv2.VMRequest) (*rpcv2.VMResponse, error) {
	return vms.HandleVirtctlCommand(ctx, req, "start")
}

func (vms *virtualMachineServer) StopVM(ctx context.Context, req *rpcv2.VMRequest) (*rpcv2.VMResponse, error) {
	return vms.HandleVirtctlCommand(ctx, req, "stop")
}

func (vms *virtualMachineServer) RestartVM(ctx context.Context, req *rpcv2.VMRequest) (*rpcv2.VMResponse, error) {
	return vms.HandleVirtctlCommand(ctx, req, "restart")
}

func (vms *virtualMachineServer) PauseVM(ctx context.Context, req *rpcv2.VMRequest) (*rpcv2.VMResponse, error) {
	return vms.HandleVirtctlCommand(ctx, req, "pause")
}

func (vms *virtualMachineServer) UnpauseVM(ctx context.Context, req *rpcv2.VMRequest) (*rpcv2.VMResponse, error) {
	return vms.HandleVirtctlCommand(ctx, req, "unpause")
}

func (vms *virtualMachineServer) HandleVirtctlCommand(ctx context.Context, req *rpcv2.VMRequest, action string) (*rpcv2.VMResponse, error) {
	sc, err := vms.sp.NewClient(ctx)
	if err != nil {
		err = errors.Wrap(err, "unable to create sentry client")
		_log.Errorw("Error in HandleVirtctlCommand", "err", err)
		return nil, err
	}
	defer sc.Close()

	resp, err := sc.GetForClusterSystemSession(ctx, getClusterRequest(req))
	if err != nil {
		_log.Errorw("unable to get kubeconfig using GetForClusterWebSession", "error", err)
		return nil, err
	}

	kubeConfig := resp.Data

	dPath := xid.New().String()

	args, err := vms.setupVirtctlEnv(dPath, kubeConfig)
	if err != nil {
		_log.Errorw("unable to setup prompt env", "error", err)
		return nil, err
	}

	var execArgs []string
	var decodedCmd string

	switch action {
	case "start", "stop", "restart":
		decodedCmd = action
	case "pause", "unpause":
		decodedCmd = action + " vm"
	}

	decodedCmd += " " + req.VMName
	for _, arg := range args {
		if strings.TrimSpace(arg) != "" {
			execArgs = append(execArgs, arg)
		}
	}

	for _, arg := range strings.Split(decodedCmd, " ") {
		if strings.TrimSpace(arg) != "" {
			execArgs = append(execArgs, arg)
		}
	}

	output, err := exec.Command("/opt/rafay/virtctl", execArgs...).Output()
	if err != nil {
		_log.Errorw("Error: Output of the command", "out", string(output), "err", err)
		return nil, fmt.Errorf(strings.ReplaceAll(strings.ReplaceAll(string(output), "\n", ""), "\"", ""))
	}

	_log.Infow("Output of the command", "out", string(output))
	return &rpcv2.VMResponse{
		Output: strings.ReplaceAll(strings.ReplaceAll(string(output), "\n", ""), "\"", ""),
	}, nil
}

func getClusterRequest(req *rpcv2.VMRequest) *sentryrpcv2.GetForClusterRequest {
	var selector []string

	selector = append(selector, req.UrlScope)
	selector = append(selector, fmt.Sprintf("rafay.dev/clusterName=%s", req.Name))

	return &sentryrpcv2.GetForClusterRequest{
		QueryOptions: ctypesv2.QueryOptions{
			PartnerID:          req.PartnerID,
			OrganizationID:     req.OrganizationID,
			AccountID:          ctypesv2.RafayID(req.AccountID),
			IsSSOUser:          req.IsSSOUser,
			Username:           req.Username,
			Groups:             req.Groups,
			IgnoreScopeDefault: req.IgnoreScopeDefault,
			GlobalScope:        req.GlobalScope,
			Selector:           strings.Join(selector, ","),
		},
		Namespace: req.Namespace,
	}
}

func (vms *virtualMachineServer) setupVirtctlEnv(dPath string, kubeConfig []byte) (args []string, err error) {
	path := fmt.Sprintf("%s/%s", vms.tmpPath, dPath)
	err = os.MkdirAll(path, 0755)
	if err != nil {
		_log.Errorw("Error in setupVirtctlEnv", "err", err)
		return
	}

	kubeConfigPath := fmt.Sprintf("%s/kubeconfig.yaml", path)

	err = ioutil.WriteFile(kubeConfigPath, kubeConfig, 0644)
	if err != nil {
		_log.Errorw("Error in setupVirtctlEnv", "err", err)
		return
	}

	args = append(args, fmt.Sprintf("--kubeconfig=%s", kubeConfigPath))

	return
}
