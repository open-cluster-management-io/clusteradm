// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("test clusteradm version", func() {
	ginkgo.It("write your tests here", func() {
		err := e2e.Clusteradm().Version().Run()
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm version check error")
	})
})
