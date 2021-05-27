[comment]: # ( Copyright Contributors to the Open Cluster Management project )
# clusteradm CLI & CLI Plugin

A CLI and kubernetes CLI plugin that allows you to interact with open-cluster-management to manage your Hybrid Cloud presence from the command-line.

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
## Contributing

See our [Contributing Document](CONTRIBUTING.md) for more information.  

## Commands

The commands are composed of a verb and a noun and then a number of parameters.

### version

`clusteradm version`
