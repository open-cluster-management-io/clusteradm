[comment]: # ( Copyright Contributors to the Open Cluster Management project )

# clusteradm

[![Go Report Card](https://goreportcard.com/badge/open-cluster-management.io/clusteradm)](https://goreportcard.com/report/open-cluster-management.io/clusteradm)
[![License](https://img.shields.io/github/license/open-cluster-management-io/clusteradm)](/LICENSE)
[![GitHub release](https://img.shields.io/github/release/open-cluster-management-io/clusteradm.svg)](https://github.com/open-cluster-management-io/clusteradm/releases/)

**clusteradm** is the command-line tool for [Open Cluster Management (OCM)](https://open-cluster-management.io/), providing a unified interface to manage multi-cluster Kubernetes environments from the command line.

## Overview

[Open Cluster Management (OCM)](https://open-cluster-management.io/) is a CNCF sandbox project that enables end-to-end visibility and control across your Kubernetes clusters using a powerful hub-agent architecture. OCM provides:

- **Cluster Lifecycle Management**: Register, manage, and monitor multiple Kubernetes clusters
- **Application Distribution**: Deploy and manage applications across multiple clusters
- **Policy & Governance**: Enforce security policies and compliance across your fleet
- **Add-on Extensibility**: Extend functionality with a rich ecosystem of add-ons

**clusteradm** serves as the primary CLI tool for interacting with OCM, enabling administrators to:
- Initialize hub clusters and register managed clusters
- Deploy and manage multi-cluster applications
- Configure policies and governance
- Manage cluster sets and placements
- Install and configure add-ons

## Quick Start

### Installation

#### From install script:

```shell
curl -L https://raw.githubusercontent.com/open-cluster-management-io/clusteradm/main/install.sh | bash
```

#### From Go:

```shell
go install open-cluster-management.io/clusteradm/cmd/clusteradm@latest
```

#### From releases:

Download the latest binary from [GitHub Releases](https://github.com/open-cluster-management-io/clusteradm/releases) and add it to your `$PATH`.

#### From source:

Go 1.24+ is required to build from source.

```bash
git clone https://github.com/open-cluster-management-io/clusteradm.git
cd clusteradm
make build
./bin/clusteradm
```

### Basic Usage

Set up a multi-cluster environment in three steps:

```bash
# 1. Initialize the hub cluster
kubectl config use-context <hub-cluster-context>
clusteradm init

# 2. Join a managed cluster to the hub
kubectl config use-context <managed-cluster-context>
clusteradm join --hub-token <token> --hub-apiserver <hub-api-url> --cluster-name <cluster-name>

# 3. Accept the managed cluster on the hub
kubectl config use-context <hub-cluster-context>
clusteradm accept --clusters <cluster-name>
```

After each command, clusteradm provides the next command to execute, making the process seamless.

## Commands

clusteradm organizes commands into logical groups for different aspects of cluster management.

### General Commands

| Command | Description |
|---------|-------------|
| `create` | Create OCM resources (placements, cluster sets, sample apps, work) |
| `delete` | Delete OCM resources (cluster sets, tokens, work) |
| `get` | Display OCM resources (clusters, hub info, tokens, placements, work, add-ons) |
| `install` | Install hub add-ons |
| `uninstall` | Uninstall hub add-ons |
| `upgrade` | Upgrade cluster manager or klusterlet |
| `version` | Display clusteradm and cluster version information |

### Registration Commands

| Command | Description |
|---------|-------------|
| `init` | Initialize a hub cluster |
| `join` | Join a cluster to the hub as a managed cluster |
| `accept` | Accept cluster join requests on the hub |
| `unjoin` | Remove a cluster from the hub |
| `clean` | Clean up OCM components from the hub cluster |

### Cluster Management Commands

| Command | Description |
|---------|-------------|
| `addon` | Manage add-ons (enable, disable, create) |
| `clusterset` | Manage cluster sets (bind, unbind, set) |
| `proxy` | Access managed clusters through the cluster proxy |

### Logging and Debugging

Get detailed logs by setting the klog flag:

```bash
# Basic logging
clusteradm <command> -v 2 > logfile.log

# Verbose logging to file
clusteradm <command> -v 99 --logtostderr=false --log-file=debug.log
```

## Detailed Command Reference

### Cluster Lifecycle

#### Initialize a Hub Cluster

```bash
clusteradm init [--use-bootstrap-token]
```

Deploys OCM hub components and returns the join command for managed clusters.

> **Note**: Do not run `init` against a [multicluster-controlplane](https://github.com/open-cluster-management-io/multicluster-controlplane) instance. Use `clusteradm get token --use-bootstrap-token` instead.

#### Get Join Token

```bash
clusteradm get token [--use-bootstrap-token]
```

Retrieves the latest token for joining managed clusters.

#### Join a Managed Cluster

```bash
clusteradm join --hub-token <token> --hub-apiserver <hub-url> --cluster-name <name> \
  [--ca-file <ca-file>] [--force-internal-endpoint-lookup]
```

Installs the klusterlet agent on a managed cluster.

**Options:**
- `--ca-file`: Provide a custom CA file for hub verification
- `--force-internal-endpoint-lookup`: Required for clusters behind NAT (e.g., kind clusters)

#### Accept Cluster Registration

```bash
clusteradm accept --clusters <cluster1>,<cluster2>,...
```

Approves cluster join requests on the hub.

#### Remove a Managed Cluster

```bash
clusteradm unjoin --cluster-name <cluster-name>
```

Removes klusterlet components from a managed cluster.

### Add-on Management

#### Install Hub Add-ons

```bash
# Install specific add-ons
clusteradm install hub-addon --names argocd
clusteradm install hub-addon --names governance-policy-framework
```

#### Enable Add-ons on Managed Clusters

```bash
clusteradm addon enable --names <addon-name> --namespace <namespace> --clusters <clusters>

# Examples
clusteradm addon enable --names argocd --namespace argocd --clusters cluster1,cluster2
clusteradm addon enable --names governance-policy-framework --namespace open-cluster-management-agent-addon --clusters cluster1
```

### Cluster Sets and Placement

#### Create Cluster Sets

```bash
clusteradm create clusterset <clusterset-name>
```

#### Bind Clusters to Sets

```bash
clusteradm clusterset bind <clusterset-name> --clusters <cluster1>,<cluster2>
```

#### Create Placements

```bash
clusteradm create placement <placement-name> --clusters <cluster1>,<cluster2>
```

### Application Deployment

#### Create Sample Applications

```bash
clusteradm create sampleapp <app-name>
```

Creates and deploys sample applications using Argo CD ApplicationSets.

### Cluster Proxy

Access managed clusters through the cluster proxy:

```bash
clusteradm proxy health --cluster-name <cluster-name>
clusteradm proxy kubectl --cluster-name <cluster-name> -- <kubectl-args>
```

## Version Bundles

clusteradm uses version bundles to ensure compatibility between OCM components. You can:

- Use the default bundle: `clusteradm init`
- Specify a version: `clusteradm init --bundle-version v0.16.0`
- Override component versions: `clusteradm init --bundle-version-overrides /path/to/overrides.json`

Example override file:

```json
{
  "ocm": "v0.16.1",
  "app_addon": "v0.16.0",
  "policy_addon": "v0.16.0",
  "multicluster_controlplane": "v0.7.0"
}
```

## Examples

### Multi-cluster Application Deployment

```bash
# 1. Create a cluster set
clusteradm create clusterset production

# 2. Bind clusters to the set
clusteradm clusterset bind production --clusters web-cluster,api-cluster

# 3. Create a placement for the application
clusteradm create placement web-app-placement --clusterset production

# 4. Deploy using ManifestWork or Argo CD
clusteradm create work my-web-app --clusters web-cluster,api-cluster
```

### Policy Enforcement

```bash
# 1. Install policy framework
clusteradm install hub-addon --names governance-policy-framework

# 2. Enable on managed clusters
clusteradm addon enable --names governance-policy-framework \
  --namespace open-cluster-management-agent-addon \
  --clusters cluster1,cluster2

# 3. Enable config policy controller
clusteradm addon enable --names config-policy-controller \
  --namespace open-cluster-management-agent-addon \
  --clusters cluster1,cluster2
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on:

- Code contribution process
- Development environment setup
- Testing guidelines
- Community guidelines

## Community & Support

### Get Connected

- **Website**: [open-cluster-management.io](https://open-cluster-management.io/)
- **Slack**: [#open-cluster-mgmt](https://kubernetes.slack.com/archives/C01GE7YSUUF) on Kubernetes Slack
- **Mailing List**: [open-cluster-management@googlegroups.com](https://groups.google.com/g/open-cluster-management)
- **YouTube**: [OCM Community](https://www.youtube.com/channel/UC7xxOh2jBM5Jfwt3fsBzOZw)
- **Community Meetings**: [Calendar](https://calendar.google.com/calendar/u/0/embed?src=openclustermanagement@gmail.com)

## License

This project is licensed under the [Apache License 2.0](LICENSE).

---

Built by the [Open Cluster Management community](https://github.com/open-cluster-management-io)
