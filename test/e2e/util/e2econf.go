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
) (*TestE2eConfig, error) {

	hubConfig, err := buildConfigFromFlags(hubctx, kubeconfigpath)
	if err != nil {
		return nil, err
	}
	mcl1Config, err := buildConfigFromFlags(mcl1ctx, kubeconfigpath)
	if err != nil {
		return nil, err
	}
	ctx := clusterValues{
		hub: &clusterConfig{
			name:       hub,
			context:    hubctx,
			kubeConfig: hubConfig,
		},
		mcl1: &clusterConfig{
			name:       mcl1,
			context:    mcl1ctx,
			kubeConfig: mcl1Config,
		},
	}

	cfgval := values{
		cv: &ctx,
	}

	return &TestE2eConfig{
		values:         &cfgval,
		clusteradm:     &clusteradm{},
		Kubeconfigpath: kubeconfigpath,
	}, nil
}
