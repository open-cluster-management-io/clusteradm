[comment]: # ( Copyright Contributors to the Open Cluster Management project )
# clusteradm CLI & CLI Plugin

A CLI and kubernetes CLI plugin that allows you to interact with open-cluster-management to manage your Hybrid Cloud presence from the command-line.

## Requirements

Go 1.16 is required in order to build or contribute on this project as it leverage the `go:embed` tip.

## Installation

The binary will be installed in `$GOPATH/bin`
### CLI

```bash
git clone https://github.com/open-cluster-management-io/clusteradm.git
cd clusteradm
make build
clusteradm
```
## Contributing

See our [Contributing Document](CONTRIBUTING.md) for more information.  

## Commands

The commands are composed of a verb and a noun and then a number of parameters.

### version

Display the clusteradm version and the kubeversion

`clusteradm version`

### init

Initialize the hub by deploying the hub side resources to manage clusters.

`clusteradm init`

it returns the command line to launch on the spoke to join the hub.

### join

Install the agent on the spoke.

`clusteradm join --hub-token <token> --hub-apiserver <hub_apiserver_url> --cluster_name c1`

it returns the command line to launch on the hub the accept the spoke onboarding.

### accept

Accept the CSRs on the hub to approve the spoke clusters to join the hub.

`clustardm accept --clusters <cluster1>, <cluster2>,....`
