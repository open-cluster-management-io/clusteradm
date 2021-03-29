[comment]: # ( Copyright Contributors to the Open Cluster Management project )
# Contributing guidelines

## Contributions

All contributions to the repository must be submitted under the terms of the [Apache Public License 2.0](https://www.apache.org/licenses/LICENSE-2.0).

## Certificate of Origin

By contributing to this project you agree to the Developer Certificate of
Origin (DCO). This document was created by the Linux Kernel community and is a
simple statement that you, as a contributor, have the legal right to make the
contribution. See the [DCO](DCO) file for details.

## Contributing A Patch

1. Submit an issue describing your proposed change to the repo in question.
2. The [repo owners](OWNERS) will respond to your issue promptly.
3. Fork the desired repo, develop and test your code changes.
4. Submit a pull request.

## Issue and Pull Request Management

Anyone may comment on issues and submit reviews for pull requests. However, in
order to be assigned an issue or pull request, you must be a member of the
[open-cluster-management](https://github.com/open-cluster-management) GitHub organization.

Repo maintainers can assign you an issue or pull request by leaving a
`/assign <your Github ID>` comment on the issue or pull request.

# Requirements

- Go 1.16

# Develop new commands

- The project tries to follow the following grammar for the commands:

```bash
cm <verb> <noun>
```

- A number of verbs are already defined in [verbs](pkg/cmd/verbs/verbs.go), if you would like to add a new verb or noun, please contact the [OWNERS](OWNERS).

- The noun represents the object on which the verb applies.

- Each pair (verb/noum) has its own package. For example [create/cluster](pkg/cmd/delete/cluster/cmd.go) package contains the code to create a cluster.

- Inside the package, the code is split in 3 files: The [cmd.go](pkg/cmd/create/cluster/cmd.go) which creates the cobra command, the [options.go](pkg/cmd/create/cluster/options.go) which defines the different option parameters for the command and the the [exec.go](pkg/cmd/create/cluster/exec.go) which contains the code to execute the command.


## Resources

- Some commands needs resources files, in the project uses the `Go 1.16` `go:embed` functionality to store the resources files.
- Each command package contains its own resources in the scenario package. The scenario package contains one go file which provides the `go:embed` `embed.FS` files. For example [resources.go](pkg/cmd/create/cluster/scenario/resources.go).
- All resources must be accessed using unstrusctured, the project must not have api dependencies.

## Applier

- This project relies on the [applier](https://github.com/open-cluster-management/applier) to create, update or delete kubernetes resources. The applier allows you to create templated resources and these templates are parsed with a provided values files and then applied in the kubernetes cluster. For example, the [scenario](pkg/cmd/create/cluster/scenario) contains the templated resources to create a managed cluster on a hub.

## Client

- The [helpers](pkg/helpers/client.go) package contains methods to get a client. For the time being only a `sigs.k8s.io/controller-runtime/pkg/client` is used as it is the one needed for the applier, but if you would like to use another client for other goals, please add the method to create client in that package. The most important to to get the config from:

```Go
config, err := configFlags.ToRESTConfig()
```

as it uses also the parameters like `--server` or `--kubeconfig` to generate the client.

## Unit tests

- If the unit test needs files to be executed, these files are stored under the pair `<verb>/<noun>/test/unit` like [values-fake/yaml](pkg/cmd/detach/cluster/test/unit/values-fake.yaml).
A total coverage is shown when running `make test`. For the time being, the `cmd.go` and `client.go` are excluded from the total coverage.
- The `make test` is part of the PR acceptance and it is launched by PROW.

## Functional tests

- The project runs functional-tests `make functional-test-full`, this test deploys a [KiND](https://kind.sigs.k8s.io/) cluster, install some resource using the applier and then runs a set of tests against that cluster [run-functional-tests.sh](build/run-functional-tests.sh). 
- The `make functional-tests-full` is part of the PR acceptance and it is launched using git-actions.