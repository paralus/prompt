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

## DCO Sign off

All authors to the project retain copyright to their work. However, to ensure
that they are only submitting work that they have rights to, we are requiring
everyone to acknowledge this by signing their work.

Any copyright notices in this repo should specify the authors as "the
paralus contributors".

To sign your work, just add a line like this at the end of your commit message:

```
Signed-off-by: Joe Bloggs <joe@example.com>
```

This can easily be done with the `--signoff` option to `git commit`.
You can also mass sign-off a whole PR with `git rebase --signoff master`, replacing
`master` with the branch you are creating a pull request against, if not master.

By doing this you state that you can certify the following (from https://developercertificate.org/):

```
Developer Certificate of Origin
Version 1.1
Copyright (C) 2004, 2006 The Linux Foundation and its contributors.
1 Letterman Drive
Suite D4700
San Francisco, CA, 94129
Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.
Developer's Certificate of Origin 1.1
By making a contribution to this project, I certify that:
(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or
(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or
(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.
(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

## Need Help?

If you are interested to contribute to prompt but are stuck with any of the steps, feel free to reach out to us. Please [create an issue](https://github.com/paralus/prompt/issues/new) in this repository describing your issue and we'll take it up from there.

You can reach out to us via our [Slack Channel](https://join.slack.com/t/paralus/shared_invite/zt-1a9x6y729-ySmAq~I3tjclEG7nDoXB0A) or [Twitter](https://twitter.com/paralus_).
