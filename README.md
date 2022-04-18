[comment]: # ( Copyright Contributors to the Open Cluster Management project )
# clusteradm CLI & CLI Plugin

A CLI and kubernetes CLI plugin that allows you to interact with open-cluster-management to manage your Hybrid Cloud presence from the command-line.

## Quick start

### Install the clusteradm command-line:
#### From binaries:

The binaries for several platforms are available [here](https://github.com/open-cluster-management-io/clusteradm/releases).
- Download the compressed file from [here](https://github.com/open-cluster-management-io/clusteradm/releases)
- Uncompress the file and place the output in a directory of your $PATH

#### From source:

Go 1.17 is required in order to build or contribute on this project as it leverage the `go:embed` tip.
The binary will be installed in `$GOPATH/bin`

```bash
git clone https://github.com/open-cluster-management-io/clusteradm.git
cd clusteradm
make build
clusteradm
```
### Initialize a hub and join a cluster

```bash
# Initialize the hub
kubectl config use-context <hub cluster context> # kubectl config use-context kind-hub
clusteradm init

# Request a managed cluster to join the hub
kubectl config use-context <managed cluster context> # kubectl config use-context kind-managed-cluster
clusteradm join --hub-token <token> --hub-apiserver <api server url> --cluster-name <cluster name>

# Accept the managed cluster request on the hub
kubectl config use-context <hub cluster context> # kubectl config use-context kind-hub
clusteradm accept --clusters <list of clusters> # clusteradm accept --clusters c1,c2,...
```

After each above clusteradm command, the clusteradm will print out the next clusteradm command to execute which can be copy/paste.

## Contributing

See our [Contributing Document](CONTRIBUTING.md) for more information.

## Commands

The commands are composed of a verb and a noun and then a number of parameters.
Logs can be gather by setting the klog flag `-v`.
To get the logs in a separate file:
```
clusteradm <subcommand> -v 2 > <your_logfile>
```
or
```
clusteradm <subcommand> -v 99 --logtostderr=false --log-file=<your_log_file>
```

### version

Display the clusteradm version and the kubeversion

`clusteradm version`

### init

Initialize the hub by deploying the hub side resources to manage clusters.

`clusteradm init [--use-bootstrap-token]`

it returns the command line to launch on the spoke to join the hub.

### join

Install the agent on the spoke.

`clusteradm join --hub-token <token> --hub-apiserver <hub_apiserver_url> --cluster-name c1`

it returns the command line to launch on the hub the accept the spoke onboarding.

### accept

Accept the CSRs on the hub to approve the spoke clusters to join the hub.

`clusteradm accept --clusters <cluster1>, <cluster2>,....`

### install hub-addon

Install specific built-in add-on(s) to the hub cluster.

`clusteradm install hub-addon --names application-manager`

`clusteradm install hub-addon --names policy-framework`

### enable addons

Enable specific add-on(s) agent deployment to the given managed clusters of the specify namespace

`clusteradm addon enable --name application-manager --namespace <namespace> --cluster <cluster1>,<cluster2>,....`
