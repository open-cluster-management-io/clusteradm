[comment]: # ( Copyright Contributors to the Open Cluster Management project )
# Open Cluster Management CLI & CLI Plugin

A CLI and kubernetes CLI plugin that allows you to interact with OCM/ACM to provision and managed your Hybrid Cloud presence from the command-line.

## Requierments 

Go 1.16 is required in order to build or contribute on this project as it leverage the `go:embed` tip.

## Installation

The binary will be installed in GOPATH/bin
### CLI

```bash
git clone https://github.com/open-cluster-management/cm-cli.git
cd cm-cli
make build
```

### Plugin

This will create a binary `oc-cm` and `kubectl-cm` in the `$GOPATH/go/bin` allowing you to call `oc cm` or `kubectl cm`

```bash
git clone https://github.com/open-cluster-management/cm-cli.git
cd cm-cli
make plugin
```
## Dislaimer

This CLI (and plugin) is still in development, but aims to expose OCM/ACM's functional through a useful and lightweight CLI and kubectl/oc CLI plugin.  Some features may not be present, fully implemented, and it might be buggy!  

## Contributing

See our [Contributing Document](CONTRIBUTING.md) for more information.  

## Commands

The commands are composed of a verb and a noum and then a number of parameters.

## Cluster commands

[applier](docs/applier.md)

[cluster](docs/cluster.md)

