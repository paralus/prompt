We 💚 Opensource!

Yes, because we feel that it’s the best way to build and improve a product. It allows people like you from across the globe to contribute and improve a product over time. And we’re super happy to see that you’d like to contribute to Paralus.

We are always on the lookout for anything that can improve the product. Be it feature requests, issues/bugs, code or content, we’d love to see what you’ve got to make this better. If you’ve got anything exciting and would love to contribute, this is the right place to begin your journey as a contributor to Paralus and the larger open source community.

**How to get started?**
The easiest way to start is to look at existing issues and see if there’s something there that you’d like to work on. You can filter issues with the label [“[Good first issue](https://github.com/paralus/prompt/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)” which are relatively self sufficient issues and great for first time contributors.

Once you decide on an issue, please comment on it so that all of us know that you’re on it.

If you’re looking to add a new feature, [raise a new issue](https://github.com/paralus/prompt/issues/new) and start a discussion with the community. Engage with the maintainers of the project and work your way through.

# Prompt

Paralus Prompt is built on top of kube-prompt, this is integrated in the dashboard. kube-prompt accepts the same commands as the kubectl, except you don't need to provide the kubectl prefix. So it doesn't require the additional cost to use this cli.

### Development setup

Guide to start Prompt UI for development purpose. To try out, you need Kubernetes cluster accessible from your local machine and kubeconfig file to communicate with API server of a cluster.

You can debug prompt by using the development setup in internal/dev.

Firstly, setup the prereqisites necessary:

```bash
ln -s ~/.kube/config internal/dev/kubeconfig.yaml # symlink/copy kube config for use in debug
export KUBECTL_BIN=$(which kubectl) # set kubectl bin path
export AUDIT_LOG_FILE=$(pwd)/audit.log # set audit log write path
```

Once you have this setup, you can run the following to start the development server.

`cd internal/dev # switch to dev directory`

`go run . # in internal/dev directory`

Once you have started the server, you can navigate to `http://localhost:7009/v2/debug/ui/` in your browser to view the debug UI.
Click on kube-shell button to start a connection.

## Code Structure
The following section lists out the code structure for prompt repo: https://github.com/paralus/prompt 

```
components
├── prompt
│   ├── internal
│   ├── pkg
│   │   └── service
│   ├── Dockerfile.prompt
│   └── main.go
```

## Need Help?

If you are interested to contribute to prompt but are stuck with any of the steps, feel free to reach out to us. Please [create an issue](https://github.com/paralus/prompt/issues/new) in this repository describing your issue and we'll take it up from there.

You can reach out to us via our [Slack Channel](https://join.slack.com/t/paralus/shared_invite/zt-1a9x6y729-ySmAq~I3tjclEG7nDoXB0A) or [Twitter](https://twitter.com/paralus_).
