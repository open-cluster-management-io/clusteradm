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

## Help

```bash
cm <verbs> cluster -h
```

## Verbs

### Attach Cluster

The `attach` verb provides the capability to attach a cluster to a hub.

### Detach Cluster

The `detach` verb will detach from the hub an already managed cluster.

### Create Cluster

The `create` will create a new managed cluster and attach it to the hub. Cloud provider credentials must be given in the values.yaml.

### Delete Cluster

The `delete` detaches and deletes an existing managed cluster.