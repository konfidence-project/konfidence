package cmd

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/project"
	"github.com/konfidence-project/konfidence/internal/stage"
	"github.com/konfidence-project/konfidence/internal/stageconfiguration"
	"github.com/konfidence-project/konfidence/internal/taskorchestration"
	"github.com/konfidence-project/konfidence/internal/vectoractivation"
	"github.com/konfidence-project/konfidence/internal/vectorassembly"
	"github.com/konfidence-project/konfidence/internal/vectordeployment"
	"github.com/konfidence-project/konfidence/internal/vectorpromotion"
	pkgcmd "github.com/konfidence-project/konfidence/pkg/cmd"
)

var _ = Describe("buildControllerSetups", func() {
	allControllers := []string{
		stage.OperatorFlagName,
		taskorchestration.OperatorFlagName,
		vectoractivation.OperatorFlagName,
		vectordeployment.OperatorFlagName,
		project.OperatorFlagName,
		stageconfiguration.OperatorFlagName,
		vectorassembly.OperatorFlagName,
		vectorpromotion.OperatorFlagName,
	}

	// The setup closures are only invoked once a controller is enabled and
	// the manager runs, so a nil manager is safe for map-shape assertions.
	build := func() map[string]func() error {
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		return buildControllerSetups(ctx, cancel, nil)
	}

	It("registers all controllers", func() {
		setups := build()
		Expect(setups).To(HaveLen(len(allControllers)))
		for _, name := range allControllers {
			Expect(setups).To(HaveKey(name))
		}
	})

	It("accepts any controller in the --controllers filter", func() {
		enabled, err := pkgcmd.FilterEnabledControllers(stageconfiguration.OperatorFlagName, build())
		Expect(err).NotTo(HaveOccurred())
		Expect(enabled).To(Equal(map[string]bool{stageconfiguration.OperatorFlagName: true}))
	})
})
