[comment]: # ( Copyright Contributors to the Open Cluster Management project )
# Release Content
## Additions
- [deploy work to placement selected clusters](https://github.com/open-cluster-management-io/clusteradm/pull/247) @haoqing0110
- [add test that check manual bootstrap token](https://github.com/open-cluster-management-io/clusteradm/pull/191) @mgold1234
- [clusteradm: provide darwin arm64 binary](https://github.com/open-cluster-management-io/clusteradm/pull/204) @SataQiu
- [Chore: Introducing golangci-lint - make verify](https://github.com/open-cluster-management-io/clusteradm/pull/213) @yue9944882
- [Add proxy kubectl command](https://github.com/open-cluster-management-io/clusteradm/pull/219) @xuezhaojun
- [Doc: Adding more install approaches](https://github.com/open-cluster-management-io/clusteradm/pull/225) @yue9944882
- [Add the governance-policy-addon-controller](https://github.com/open-cluster-management-io/clusteradm/pull/229) @dhaiducek

## Breaking Changes
N/A

## Changes
- [Add yue9944882 as approver and ycyaoxdu as reviewer](https://github.com/open-cluster-management-io/clusteradm/pull/193) @qiujian16
- [upgrade to latest openshift/library-go](https://github.com/open-cluster-management-io/clusteradm/pull/194) @itdove
- [Upgrade to ginkgo v2](https://github.com/open-cluster-management-io/clusteradm/pull/203) @itdove
- [Stop using the GitHub API in the install script](https://github.com/open-cluster-management-io/clusteradm/pull/224) @mprahl
- [Add context](https://github.com/open-cluster-management-io/clusteradm/pull/244) @itdove
- [Update app addon CRDs](https://github.com/open-cluster-management-io/clusteradm/pull/246) @mikeshng

## Bug Fixes
- [modify: help information](https://github.com/open-cluster-management-io/clusteradm/pull/189) @ycyaoxdu
- [add change line to make output clear](https://github.com/open-cluster-management-io/clusteradm/pull/192) @jichenjc
- [fix: use poll method to monitor namespace deletion](https://github.com/open-cluster-management-io/clusteradm/pull/195) @ycyaoxdu
- [Add not found into output when deleted work not found](https://github.com/open-cluster-management-io/clusteradm/pull/197) @jichenjc
- [add copyright check for .mk file](https://github.com/open-cluster-management-io/clusteradm/pull/199) @ycyaoxdu
- [Add WithCache and getCache](https://github.com/open-cluster-management-io/clusteradm/pull/200) @itdove
- [Use cache in ApplyCustomResources](https://github.com/open-cluster-management-io/clusteradm/pull/202) @itdove
- [Make some Variable and Method to be at public scope](https://github.com/open-cluster-management-io/clusteradm/pull/206) @mgold1234
- [change method and veriables to public scope in addon command.](https://github.com/open-cluster-management-io/clusteradm/pull/207) @mgold1234
- [cluster_name has been converted to cluster-name](https://github.com/open-cluster-management-io/clusteradm/pull/209) @panguicai008
- [fix clusteradm addon enable command unknown flag: --names](https://github.com/open-cluster-management-io/clusteradm/pull/211) @panguicai008
- [fix clusteradm join usage example](https://github.com/open-cluster-management-io/clusteradm/pull/212) @panguicai008
- [add bundler version into output](https://github.com/open-cluster-management-io/clusteradm/pull/215) @jichenjc
- [change method and veriables to public scope in clean command.](https://github.com/open-cluster-management-io/clusteradm/pull/216) @mgold1234
- [change method and veriables to public scope in clusterset_bind command.](https://github.com/open-cluster-management-io/clusteradm/pull/217) @mgold1234
- [fix: correct spelling of output table](https://github.com/open-cluster-management-io/clusteradm/pull/218) @vbelouso
- [change method and veriables to public scope in clusetset_set command.](https://github.com/open-cluster-management-io/clusteradm/pull/220) @mgold1234
- [change method and veriables to public scope in clusetset_unbind command.](https://github.com/open-cluster-management-io/clusteradm/pull/221) @mgold1234
- [Update PolicySet CRD to match propagator](https://github.com/open-cluster-management-io/clusteradm/pull/222) @JustinKuli
- [change method and veriables to public scope in create command.](https://github.com/open-cluster-management-io/clusteradm/pull/223) @mgold1234
- [inogre the vscode caches](https://github.com/open-cluster-management-io/clusteradm/pull/226) @xauthulei
- [Fix init messaging](https://github.com/open-cluster-management-io/clusteradm/pull/227) @dhaiducek
- [Update: proxy kubectl logs](https://github.com/open-cluster-management-io/clusteradm/pull/230) @xuezhaojun
- [Consistent flags/Fix spelling](https://github.com/open-cluster-management-io/clusteradm/pull/231) @dhaiducek 
- [update wording to make error clear](https://github.com/open-cluster-management-io/clusteradm/pull/232) @jichenjc
- [prevent delete of default clusterset](https://github.com/open-cluster-management-io/clusteradm/pull/233) @jichenjc
- [show controller image in ./clusteradm get klusterlet-info](https://github.com/open-cluster-management-io/clusteradm/pull/237) @jichenjc
- [Add dry run for delete clusterset](https://github.com/open-cluster-management-io/clusteradm/pull/239) @jichenjc 
- [Force developers to call Build()](https://github.com/open-cluster-management-io/clusteradm/pull/242) @itdove
- [Force developers to call Build()](https://github.com/open-cluster-management-io/clusteradm/pull/243) @itdove
