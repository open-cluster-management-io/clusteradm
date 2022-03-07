// Copyright Contributors to the Open Cluster Management project
package clusteradme2e

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("test clusteradm version", func() {
	var err error

	ginkgo.It("write your tests here", func() {
		err = e2e.Clusteradm().Version()
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "clusteradm version check error")
	})
})
