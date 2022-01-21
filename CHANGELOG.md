[comment]: # ( Copyright Contributors to the Open Cluster Management project )
# Release Content
## Additions
- [Feat: Allows override OCM images upon initialization](https://github.com/open-cluster-management-io/clusteradm/pull/80) @yue9944882
- [Clusteradm proxy health](https://github.com/open-cluster-management-io/clusteradm/pull/90) @yue9944882
- [adding a '--wait' flag for optional foreground joining](https://github.com/open-cluster-management-io/clusteradm/pull/95) @yue9944882
- [feat: clusteradm init supports saving commands to local file](https://github.com/open-cluster-management-io/clusteradm/pull/93) @yue9944882
- [chore: reflecting detailed container status on cluster joining](https://github.com/open-cluster-management-io/clusteradm/pull/99) @yue9944882
- [addon enable: allow certain addon to be deployed](https://github.com/open-cluster-management-io/clusteradm/pull/101) @ycyaoxdu
- [Feat: clusteradm get addon command for list enabled addon](https://github.com/open-cluster-management-io/clusteradm/pull/103) @ycyaoxdu

## Breaking Changes
N/A

## Changes
- [Enhancement to clean up the control plane.](https://github.com/open-cluster-management-io/clusteradm/pull/75) @xauthulei
- [Added timeout flag to fix issue](https://github.com/open-cluster-management-io/clusteradm/pull/77) @ilan-pinto
- [Adding an install scripts for one line clusteradm installation](https://github.com/open-cluster-management-io/clusteradm/pull/81) @yue9944882
- [Fix issue #52 - added image rending to the join command](https://github.com/open-cluster-management-io/clusteradm/pull/83) @ilan-pinto
- [Specify install ns when enable an addon on a cluster](https://github.com/open-cluster-management-io/clusteradm/pull/89) @xauthulei
- [use the endpoint from --hub-apiserver for joining cluster](https://github.com/open-cluster-management-io/clusteradm/pull/107) @yue9944882

## Bug Fixes
- [Refactor/Bugfix: clusteradm join endpoint and spinner loading effect](https://github.com/open-cluster-management-io/clusteradm/pull/91) @yue9944882
- [fixes go install due to replace directive](https://github.com/open-cluster-management-io/clusteradm/pull/92) @yue9944882
- [modify integration test for command addon enable](https://github.com/open-cluster-management-io/clusteradm/pull/105) @ycyaoxdu

