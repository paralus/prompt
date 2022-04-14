## Prompt Testing UI

Guide to start Prompt UI for development purpose. To try out, you need
Kubernetes cluster accessible from your local machine and kubeconfig
file to communicate with API server of a cluster.

You can debug prompt by using the development setup in
`internal/dev`. Firstly, setup the prereqisites necessary:

```bash
ln -s ~/.kube/config internal/dev/kubeconfig.yaml # symlink/copy kube config for use in debug
export KUBECTL_BIN=$(which kubectl) # set kubectl bin path
export AUDIT_LOG_FILE=$(pwd)/audit.log # set audit log write path
```

Once you have this setup, you can run the following to start the
development server.

```bash
cd internal/dev # switch to dev directory
go run . # in `internal/dev` directory
```

Once you have started the server, you can navigate to
[http://localhost:7009/v2/debug/ui/](http://localhost:7009/v2/debug/ui/)
in your browser to view the debug UI.

Click on <kbd>kube-shell</kbd> button to start a connection.
