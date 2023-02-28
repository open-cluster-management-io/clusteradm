[comment]: # ( Copyright Contributors to the Open Cluster Management project )
# clusteradm CLI & CLI Plugin

A CLI and kubernetes CLI plugin that allows you to interact with open-cluster-management to manage your Hybrid Cloud presence from the command-line.

## Quick start

### Install the clusteradm command-line:

#### From install script:

```shell
curl -L https://raw.githubusercontent.com/open-cluster-management-io/clusteradm/main/install.sh | bash
```

#### From go install:

```shell
GO111MODULE=off go get -u open-cluster-management.io/clusteradm/...
```

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
> NOTE: Do not run init command against a [multicluster-controlplane](https://github.com/open-cluster-management-io/multicluster-controlplane) instance. It is already an initialized hub on start. Instead, use `clusteradm get token --use-bootstrap-token` to get the join command.

### get token

Get the latest token to import a new managed cluster.

`clusteradm get token --context ${CTX_HUB_CLUSTER}`
### join

Install the agent on the spoke.

`clusteradm join --hub-token <token> --hub-apiserver <hub_apiserver_url> --cluster-name c1 [--ca-file <path-to-ca-file>] [--force-internal-endpoint-lookup]`

it returns the command line to launch on the hub the accept the spoke onboarding.

> NOTE: The `--ca-file` flag is used to provide a valid CA for hub. The ca data is fetched from cluster-info configmap in kube-public namespace of the hub cluster, then from kube-root-ca.crt configmap in kube-public namespace if the cluster-info configmap does not exist.

> NOTE: If you're trying to join a hub cluster which is initialized from a kind cluster, please set the `--force-internal-endpoint-lookup` flag.

### accept

Accept the CSRs on the hub to approve the spoke clusters to join the hub.

`clusteradm accept --clusters <cluster1>, <cluster2>,....`

### unjoin

Uninstall the agent on the spoke

`clusteradm unjoin --cluster-name c1`
> Note: the applied resources on managed cluster would be checked and prompt a warning if still exist any.

### clean

Clean up the multicluster hub control plane and other initialized resources on the hub cluster

`clusteradm clean --context ${CTX_HUB_CLUSTER}`

### install hub-addon

Install specific built-in add-on(s) to the hub cluster.

`clusteradm install hub-addon --names application-manager`

`clusteradm install hub-addon --names governance-policy-framework`

### enable addons

Enable specific add-on(s) agent deployment to the given managed clusters of the specified namespace

`clusteradm addon enable --names application-manager --namespace <namespace> --clusters <cluster1>,<cluster2>,....`

`clusteradm addon enable --names governance-policy-framework --namespace <namespace> --clusters <cluster1>,<cluster2>,....`

`clusteradm addon enable --names config-policy-controller --namespace <namespace> --clusters <cluster1>,<cluster2>,....`

### create sample application

Create and Deploy a Sample Subscription Application

`clusteradm create sampleapp sampleapp1`
