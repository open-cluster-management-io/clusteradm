[comment]: # ( Copyright Contributors to the Open Cluster Management project )
# clusteradm CLI & CLI Plugin

A CLI and kubernetes CLI plugin that allows you to interact with open-cluster-management to provision and managed your Hybrid Cloud presence from the command-line.

## Requirements

Go 1.16 is required in order to build or contribute on this project as it leverage the `go:embed` tip.

## Installation

The binary will be installed in `$GOPATH/bin`
### CLI

```bash
git clone https://open-cluster-management-io/clusteradm.git
cd clusteradmin
make build
cm
```

### Plugin

This will create a binary `oc-clusteradm` and `kubectl-clusteradm` in the `$GOPATH/go/bin` allowing you to call `oc clusteradm` or `kubectl clusteradm`
```bash
git clone https://open-cluster-management-io/clusteradm.git
cd clusteradm
make plugin
kubectl clusteradm
oc clusteradm
```
## Disclaimer

This CLI (and plugin) is still in development, but aims to expose CNCF's functional through a useful and lightweight CLI and kubectl/oc CLI plugin.  Some features may not be present, fully implemented, and it might be buggy!  

## Contributing

See our [Contributing Document](CONTRIBUTING.md) for more information.  

## Commands

The commands are composed of a verb and a noun and then a number of parameters.

### version

`clusteradm version`
