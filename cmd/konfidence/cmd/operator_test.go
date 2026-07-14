package cmd

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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
	starControllers := []string{
		stage.OperatorFlagName,
		taskorchestration.OperatorFlagName,
		vectoractivation.OperatorFlagName,
		vectordeployment.OperatorFlagName,
	}
	galaxyControllers := []string{
		stageconfiguration.OperatorFlagName,
		vectorassembly.OperatorFlagName,
		vectorpromotion.OperatorFlagName,
	}

	// The setup closures are only invoked once a controller is enabled and
	// the manager runs, so a nil manager is safe for map-shape assertions.
	build := func(enableGalaxy bool) map[string]func() error {
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		return buildControllerSetups(ctx, cancel, nil, enableGalaxy)
	}

	Context("with galaxy enabled", func() {
		It("registers the star and galaxy controllers", func() {
			setups := build(true)
			Expect(setups).To(HaveLen(len(starControllers) + len(galaxyControllers)))
			for _, name := range append(starControllers, galaxyControllers...) {
				Expect(setups).To(HaveKey(name))
			}
		})

		It("accepts a galaxy controller in the --controllers filter", func() {
			enabled, err := pkgcmd.FilterEnabledControllers(stageconfiguration.OperatorFlagName, build(true))
			Expect(err).NotTo(HaveOccurred())
			Expect(enabled).To(Equal(map[string]bool{stageconfiguration.OperatorFlagName: true}))
		})
	})

	Context("with galaxy disabled", func() {
		It("registers only the star controllers", func() {
			setups := build(false)
			Expect(setups).To(HaveLen(len(starControllers)))
			for _, name := range starControllers {
				Expect(setups).To(HaveKey(name))
			}
		})

		It("rejects a galaxy controller in the --controllers filter", func() {
			_, err := pkgcmd.FilterEnabledControllers(stageconfiguration.OperatorFlagName, build(false))
			Expect(err).To(MatchError(ContainSubstring("matches no registered controller")))
		})
	})
})
