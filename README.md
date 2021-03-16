[comment]: # ( Copyright Contributors to the Open Cluster Management project )
# Open Cluster Management CLI & CLI Plugin

A CLI and kubernetes CLI plugin that allows you to interact with OCM/ACM to provision and managed your Hybrid Cloud presence from the command-line.

## Installation

### CLI
```bash
git clone https://github.com/open-cluster-management/cm-cli.git
cd cm-cli
make build
```

### oc/kubectl Plugin


```bash
git clone https://github.com/open-cluster-management/cm-cli.git
cd cm-cli
make build && mv $GOPATH/bin/cm $GOPATH/bin/kubectl-cm
```
### oc Plugin only

```bash
git clone https://github.com/open-cluster-management/cm-cli.git
cd cm-cli
make build && mv $GOPATH/bin/cm $GOPATH/bin/oc-cm
```

## Dislaimer

This CLI (and plugin) is still in development, but aims to expose OCM/ACM's functional through a useful and lightweight CLI and kubectl/oc CLI plugin.  Some features may not be present, fully implemented, and it might be buggy!  

## Contributing

See our [Contributing Document](CONTRIBUTING.md) for more information.  

## Commands

The commands are composed of a verb and a noum and then a number of parameters.


