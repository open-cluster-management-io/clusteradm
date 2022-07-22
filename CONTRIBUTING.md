[comment]: # ( Copyright Contributors to the Open Cluster Management project )**Table of Contents**

- [Contributing guidelines](#contributing-guidelines)
  - [Terms](#terms)
  - [Certificate of Origin](#certificate-of-origin)
  - [Contributing a patch](#contributing-a-patch)
  - [Issue and pull request management](#issue-and-pull-request-management)
- [Requirements](#requirements)
- [Develop new commands](#develop-new-commands)
  - [Resources](#resources)
  - [Client](#client)
  - [Unit tests](#unit-tests)
  - [E2E tests](#e2e-tests)

# Contributing guidelines

## Terms

All contributions to the repository must be submitted under the terms of the [Apache Public License 2.0](https://www.apache.org/licenses/LICENSE-2.0).

## Certificate of Origin

By contributing to this project, you agree to the Developer Certificate of Origin (DCO). This document was created by the Linux Kernel community and is a simple statement that you, as a contributor, have the legal right to make the contribution. See the [DCO](DCO) file for details.

## Contributing a patch

1. Submit an issue describing your proposed change to the repository in question. The repository owners will respond to your issue promptly.
2. Fork the desired repository, then develop and test your code changes.
3. Submit a pull request.

## Issue and pull request management

Anyone can comment on issues and submit reviews for pull requests. In order to be assigned an issue or pull request, you can leave a `/assign <your Github ID>` comment on the issue or pull request.

# Requirements

- Go 1.17

# Develop new commands

- The project tries to follow the following grammar for the commands:

```bash
clusteradm <cmd> [subcmd] [flags]
```

- Each cmd/subcmd are in a package, the code is split in 3 files: The [cmd.go](pkg/cmd/version/cmd.go) which creates the cobra command, the [options.go](pkg/cmd/version/options.go) which defines the different option parameters for the command and the the [exec.go](pkg/cmd/version/exec.go) which contains the code to execute the command.
- Each command must support the flag `--dry-run`.
- The command uses [klog V2](https://github.com/kubernetes/klog) as logging package. All messages must be using `klog.V(x)`, in rare exception `klog.Error` and `klog.Warning` can be used.

## Resources

- Some commands needs resources files, in the project uses the `Go 1.17` `go:embed` functionality to store the resources files.
- Each command package contains its own resources in the scenario package. The scenario package contains one go file which provides the `go:embed` `embed.FS` files.

## Client

- The [main](cmd/clusteradm.go) provides a cmdutil.Factory which can be leveraged to get different clients and also the *rest.Config. The factory can be passed to the cobra.Command and then save in the Options.

```Go
kubeClient, err := o.factory.KubernetesClientSet()
```

```Go
config, err := f.ToRESTConfig()
```

## Unit tests

- If the unit test needs files to be executed, these files are stored under the pair `<verb>/<noun>/test/unit`.
A total coverage is shown when running `make test`. For the time being, the `cmd.go` and `client.go` are excluded from the total coverage.
- The `make test` is part of the PR acceptance and it is launched by PROW.

## E2E tests

- The project use `make test-e2e` to run e2e tests, this will deploy kind cluster and run a set of tests for clusteradm commands. A prerequisite is that Docker is already running.
- We have a [README](/test/e2e/README.md) indicating the way to write e2e tests.
- The `make test-e2e` is part of the PR acceptance and it is launched using git-actions.
