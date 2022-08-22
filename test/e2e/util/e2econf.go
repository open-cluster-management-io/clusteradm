// Copyright Contributors to the Open Cluster Management project
package util

type TestE2eConfig struct {
	values     *values
	clusteradm *clusteradm

	Kubeconfigpath string

	ClearEnv func() error
}

func (tec *TestE2eConfig) CommandResult() *HandledOutput {
	return &tec.clusteradm.h
}

func (tec *TestE2eConfig) Cluster() *clusterValues {
	return tec.values.cv
}

func (tec *TestE2eConfig) Clusteradm() clusteradmInterface {
	return tec.clusteradm
}

func NewTestE2eConfig(
	kubeconfigpath string,
	hub string,
	hubctx string,
	mcl1 string,
	mcl1ctx string,
	mcl2 string,
	mcl2ctx string,
) *TestE2eConfig {

	ctx := clusterValues{
		hub: &clusterConfig{
			name:    hub,
			context: hubctx,
		},
		mcl1: &clusterConfig{
			name:    mcl1,
			context: mcl1ctx,
		},
		mcl2: &clusterConfig{
			name:    mcl2,
			context: mcl2ctx,
		},
	}

	cfgval := values{
		cv: &ctx,
	}

	return &TestE2eConfig{
		values:         &cfgval,
		clusteradm:     &clusteradm{},
		Kubeconfigpath: kubeconfigpath,
	}
}
