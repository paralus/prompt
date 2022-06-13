# Prompt

Paralus Prompt is built on top of kube-prompt, this is integrated in the dashboard. kube-prompt accepts the same commands as the kubectl, except you don't need to provide the kubectl prefix. So it doesn't require the additional cost to use this cli.

<img src="https://website-git-namespace-paralus.vercel.app/img/docs/importcluster-kubectl.png" alt="Paralus Prompt in action" height="50%" widht="50%"/>

## kube-prompt

![Software License](https://img.shields.io/badge/license-apache-brightgreen.svg?style=flat-square)

An interactive kubernetes client featuring auto-complete using [go-prompt](https://github.com/paralus/prompt/pkg/prompt).

![demo](https://github.com/c-bata/assets/raw/master/kube-prompt/kube-prompt.gif)

kube-prompt accepts the same commands as the kubectl, except you don't need to provide the `kubectl` prefix.
So it doesn't require the additional cost to use this cli.

And you can integrate other commands via pipe (`|`).

```
>>> get pod | grep web
web-1144924021-2spbr        1/1     Running     4       25d
web-1144924021-5r1fg        1/1     Running     4       25d
web-1144924021-pqmfq        1/1     Running     4       25d
```

### Installation for development

For local development and setup, follow the steps mentioned in [dev-installation](https://github.com/paralus/prompt/tree/main/internal/dev) document.

## Community & Support

- Visit [Paralus website](https://paralus.io) for the complete documentation and helpful links.
- Join our [Slack channel](https://join.slack.com/t/paralus/shared_invite/zt-1a9x6y729-ySmAq~I3tjclEG7nDoXB0A) to post your queries and discuss features.
- Tweet to [@paralus_](https://twitter.com/paralus_/) on Twitter.
- Create [GitHub Issues](https://github.com/paralus/paralus/issues) to report bugs or request features.

## Contributing

The easiest way to start is to look at existing issues and see if there’s something there that you’d like to work on. You can filter issues with the label “Good first issue” which are relatively self sufficient issues and great for first time contributors.

Once you decide on an issue, please comment on it so that all of us know that you’re on it.

If you’re looking to add a new feature, raise a [new issue](https://github.com/paralus/prompt/issues) and start a discussion with the community. Engage with the maintainers of the project and work your way through.
