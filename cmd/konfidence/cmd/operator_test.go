package cmd

import (
	"github.com/konfidence-project/konfidence/internal/deploymenttarget"
	"github.com/konfidence-project/konfidence/internal/landscape"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/project"
	"github.com/konfidence-project/konfidence/internal/stage"
	"github.com/konfidence-project/konfidence/internal/taskorchestration"
	"github.com/konfidence-project/konfidence/internal/vectoractivation"
	"github.com/konfidence-project/konfidence/internal/vectorassembly"
	"github.com/konfidence-project/konfidence/internal/vectordeployment"
	"github.com/konfidence-project/konfidence/internal/vectorpromotion"
	pkgcmd "github.com/konfidence-project/konfidence/pkg/cmd"
	"github.com/konfidence-project/konfidence/pkg/operator"
)

var _ = Describe("controllerDomains", func() {
	allControllers := []string{
		deploymenttarget.OperatorFlagName,
		stage.OperatorFlagName,
		taskorchestration.OperatorFlagName,
		vectoractivation.OperatorFlagName,
		vectordeployment.OperatorFlagName,
		project.OperatorFlagName,
		landscape.OperatorFlagName,
		vectorassembly.OperatorFlagName,
		vectorpromotion.OperatorFlagName,
	}

	It("registers all controller groups", func() {
		Expect(operator.Names(controllerDomains())).To(ConsistOf(allControllers))
	})

	It("lists the controllers of every group in the flag help", func() {
		help := controllersHelp()
		for _, domain := range controllerDomains() {
			Expect(domain.Controllers).NotTo(BeEmpty())
			Expect(help).To(ContainSubstring(domain.Name))
			Expect(help).To(ContainSubstring(domain.Controllers))
		}
	})

	It("accepts any controller group in the --controllers filter", func() {
		enabled, err := pkgcmd.FilterEnabledControllers(vectorassembly.OperatorFlagName,
			operator.Names(controllerDomains()))
		Expect(err).NotTo(HaveOccurred())
		Expect(enabled).To(Equal(map[string]bool{vectorassembly.OperatorFlagName: true}))
	})
})
