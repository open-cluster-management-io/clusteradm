[comment]: # ( Copyright Contributors to the Open Cluster Management project )

# Cluster noun

The CLI has commands to manage clusters.
Behind the scene it uses the [applier](https://github.com/open-cluster-management/applier) with predefined templates. One template per verb.

```bash
cm <verb> cluster <options...>
```

## Brief common options

- `--outfile | -o`: Outputs the resources yaml in a file rather than to apply them in the cluster. Later, `oc|kubectl apply -f` can be used to effectively apply it.
- `--out-templates-dir`: Extracts the templates used for a given verb and the values.yaml sample.
- `--values`: Provides to the CLI the values that will used to parse the templates. The values can also be provided as a input pipe like 
```bash
cat values.yaml | cm <verb> cluster
```

The values.yaml have the same format and so if one is used for `create` it can be used for `attach`, `delete`, `detach`.

## Help

```bash
cm <verbs> cluster -h
```

## Verbs

### Attach Cluster

```
cm attach cluster --values <values_yaml_path>
```
or
```
cm attach cluster --name <cluster_name> [--cluster-kubeconfig <managed_cluster_kubeconfig_path>]
```

The `attach` verb provides the capability to attach a cluster to a hub.
The `attach` can be done on different ways. 
1. Manually, meaning once you ran the `attach` you still have to run an `cm apply` command (provided by the execution of the `attach` command) to install the agent on the managed cluster.

2. Automatically:
    a) By providing the kubeconfig in the [values.yaml](../pkg/cmd/attach/cluster/scenario/attach/values-template.yaml), then a secret will be created on the hub cluster and the system will use it to install the agent. The secret is deleled if the `attach` failed or succeed and so the credentials are not kept on the hub.
    b) By providing the pair server/token in the [values.yaml](../pkg/cmd/attach/cluster/scenario/attach/values-template.yaml) and again a secret will be created on the hub and the system will use it to install the agent. The secret is deleled if the `attach` failed or succeed and so the credentials are not kept on the hub. 
    c) When the cluster was provisionned with hive. If the cluster was provisionned with hive, a clusterdeployemnt custom resource exists which contain a secret to access the remote cluster and thus if you `attach` a hive cluster, you don't have to provide any credential to access the cluster. The system will find out the credentials and attach the cluster.

    The `attach` command also takes `--name` and `--cluster-kubeconfig` instead of the `--values`, in that case the default [values.yaml](../pkg/attach/cluster/scenario/attach/values-default.yaml) will be used.

5. Attaching the hub: by default the hub is attached but if you detached it and want to reattach it you just have to provide a [values.yaml](../pkg/cmd/attach/cluster/scenario/attach/values-template.yaml) with a cluster name `local-cluster`. The system will recognized that name and use the cluster credentials to do the attach.

### Detach Cluster

```
cm detach cluster --values <values_yaml_path>
```
or
```
cm detach cluster --name <cluster_name>
```

The `detach` verb will detach from the hub an already managed cluster.

### Create Cluster

```
cm create cluster --values <values_yaml_path>
```

The `create` will create a new managed cluster and attach it to the hub. Cloud provider credentials must be given in the values.yaml.

### Delete Cluster


```
cm delete cluster --values <values_yaml_path>
```
or
```
cm delete cluster --name <cluster_name>
```

The `delete` detaches and deletes an existing managed cluster.