package deploymenttarget

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeploymentTarget(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DeploymentTarget Suite")
}
