# Clusteradm
<img align="right" src="logo.png" alt="open-cluster-management"/>

Clusteradm is a tool built to provide `clusteradm init` and `clusteradm join` as best-practice "fast paths" for managing multiple Kubernetes clusters.

## Installation

1. Clone the repository. For example:
   ```
   git clone git@github.com:open-cluster-management-io/clusteradm.git
   ./clusteradm
   ```
1. Make sure you have all the [dependencies](#dependencies).

## Dependencies

- `bash` 
   - version 4 or newer
   - on macOS with [Homebrew](https://brew.sh/) installed, run `brew install bash`. This bash must be first in your path, but need not be `/bin/bash` or your default login shell.
- `gsed`
  - on macOS with [Homebrew](https://brew.sh/) installed, run `brew install gnu-sed`.

## Usage
Online help is available directly from the CLI using the global `-h` option.

[View Usage](./USAGE.md)
