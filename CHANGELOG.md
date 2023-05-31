[comment]: # ( Copyright Contributors to the Open Cluster Management project )
# Release Content
## Additions
- [feature: enhance unjoin command to support hosted klusterlet](https://github.com/open-cluster-management-io/clusteradm/pull/317) @ycyaoxdu
- [Add feature-gate flag for init and join](https://github.com/open-cluster-management-io/clusteradm/pull/325) @qiujian16
- [Add interactiveMode in proxy.](https://github.com/open-cluster-management-io/clusteradm/pull/328) @xuezhaojun
- [Add cmd to create placement](https://github.com/open-cluster-management-io/clusteradm/pull/331) @qiujian16
- [Apply manifestowkr replicaset when use --replicaset label](https://github.com/open-cluster-management-io/clusteradm/pull/337) @qiujian16
- [Set autoApproveUsers with ManagedClusterAutoApproval feature gates](https://github.com/open-cluster-management-io/clusteradm/pull/341) @qiujian16

## Breaking Changes

## Changes
- [Refactor get addon cmd](https://github.com/open-cluster-management-io/clusteradm/pull/316) @qiujian16
- [Refactor cluster and clusters flags](https://github.com/open-cluster-management-io/clusteradm/pull/327) @qiujian16
- [Rm applier](https://github.com/open-cluster-management-io/clusteradm/pull/332) @qiujian16
- [Remove applier deps from tests](https://github.com/open-cluster-management-io/clusteradm/pull/333) @qiujian16
- [List clusters when getting clusterset](https://github.com/open-cluster-management-io/clusteradm/pull/335) @qiujian16

## Bug Fixes
- [Bug Fix: remove bootstratp secret during clean](https://github.com/open-cluster-management-io/clusteradm/pull/319) @USER0308
- [add preflight check for cluster name](https://github.com/open-cluster-management-io/clusteradm/pull/321) @ycyaoxdu
- [Fix get work filter and return more details](https://github.com/open-cluster-management-io/clusteradm/pull/326) @qiujian16
- [Fix: flaky e2e failed because of "Error: cluster manager still exists"](https://github.com/open-cluster-management-io/clusteradm/pull/329) @xuezhaojun
- [fix: version break line](https://github.com/open-cluster-management-io/clusteradm/pull/330) @huiwq1990
- [fix 3 CVEs: CVE-2022-27664, CVE-2022-32149, CVE-2022-28948](https://github.com/open-cluster-management-io/clusteradm/pull/345) @ycyangxdu
