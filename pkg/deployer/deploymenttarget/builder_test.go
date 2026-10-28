package deploymenttarget

import (
	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewControllerManagedBy", func() {
	It("validates required artifact types and watcher configuration", func() {
		valid := func() *builder {
			return NewControllerManagedBy(nil).For(konfidencev1alpha1.ArtifactTypeSet{"cloud.helm.example": {}})
		}
		Expect(valid().validateConfiguration()).To(Succeed())
		invalid := []*builder{
			NewControllerManagedBy(nil), NewControllerManagedBy(nil).For(nil),
			NewControllerManagedBy(nil).For(konfidencev1alpha1.ArtifactTypeSet{" ": {}}),
			valid().Watches(nil, referencedTargetsMarker), valid().Watches(&konfidencev1alpha1.DeploymentTarget{}, nil),
		}
		for _, b := range invalid {
			err := b.validateConfiguration()
			Expect(err).To(HaveOccurred())
		}
	})
})
