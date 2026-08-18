//nolint:staticcheck // ST1001: allow dot-import for test utils using Gomega
package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utility functions", func() {
	const (
		ArtifactName                    = "github.com/konfidence/frontend/kdenv-ui"
		ArtifactNameWithLongName        = "github.com/konfidence/frontend/kdenv-angular-user-interface-development-alpha1-ui"
		ArtifactNameWithoutSlash        = "kdenv-ui"
		ArtifactNameWithSlashTerminator = "kdenv-ui/"
		ArtifactVersion                 = "1.0.0"
		UID                             = "5683asd56bf00"
	)

	// dnsLabel is the RFC 1123 label contract every returned name must satisfy:
	// lowercase alphanumerics and dashes, starting and ending alphanumeric.
	const dnsLabel = `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`

	Context("Construct Artifact Deployment Name", func() {
		It("should fail if artifact name is empty", func() {
			_, _, err := ConstructArtifactDeploymentName("   ", ArtifactVersion, nil, 0)
			Expect(err).To(MatchError("artifact name or version is empty"))
		})

		It("should fail if artifact version is empty", func() {
			_, _, err := ConstructArtifactDeploymentName(ArtifactName, "   ", nil, 0)
			Expect(err).To(MatchError("artifact name or version is empty"))
		})

		// Properties that must hold for ANY valid input — asserted structurally
		// rather than against hardcoded golden strings, so they survive any
		// implementation change that preserves the contract.
		DescribeTable("should honor the naming contract",
			func(artifactName, artifactVersion string, uid *string, collisionCount int32) {
				name, deploymentHash, err := ConstructArtifactDeploymentName(artifactName, artifactVersion, uid, collisionCount)

				Expect(err).ToNot(HaveOccurred())

				By("producing a hash of fixed width and base36 charset")
				Expect(deploymentHash).To(HaveLen(10))
				Expect(deploymentHash).To(MatchRegexp("^[0-9a-z]+$"))

				By("producing a name within the label size budget")
				Expect(len(name)).To(BeNumerically("<=", MaxLabelSize))

				By("producing a valid RFC 1123 DNS label")
				Expect(name).To(MatchRegexp(dnsLabel))

				By("embedding the hash in the name")
				Expect(name).To(ContainSubstring(deploymentHash))
			},
			Entry("short name with slash", ArtifactName, ArtifactVersion, nil, int32(0)),
			Entry("name without slash", ArtifactNameWithoutSlash, ArtifactVersion, nil, int32(0)),
			Entry("name ending in slash", ArtifactNameWithSlashTerminator, ArtifactVersion, nil, int32(0)),
			Entry("long name requiring truncation", ArtifactNameWithLongName, ArtifactVersion, nil, int32(0)),
			Entry("version so long only the hash fits", ArtifactName, "1gamma.0beta.0alpha-test-1234567890-delta-theta-3456999", nil, int32(0)),
			Entry("with uid", ArtifactName, ArtifactVersion, new(UID), int32(0)),
			Entry("with collision salt", ArtifactName, ArtifactVersion, nil, int32(1)),
			Entry("with uid and collision salt", ArtifactName, ArtifactVersion, new(UID), int32(3)),
		)

		It("should be deterministic for identical input", func() {
			name1, hash1, err1 := ConstructArtifactDeploymentName(ArtifactName, ArtifactVersion, nil, 0)
			name2, hash2, err2 := ConstructArtifactDeploymentName(ArtifactName, ArtifactVersion, nil, 0)
			Expect(err1).ToNot(HaveOccurred())
			Expect(err2).ToNot(HaveOccurred())
			Expect(name1).To(Equal(name2))
			Expect(hash1).To(Equal(hash2))
		})

		It("should produce a different name when a uid is supplied (reuse disabled)", func() {
			reusable, _, err := ConstructArtifactDeploymentName(ArtifactName, ArtifactVersion, nil, 0)
			Expect(err).ToNot(HaveOccurred())
			unique, _, err := ConstructArtifactDeploymentName(ArtifactName, ArtifactVersion, new(UID), 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(unique).ToNot(Equal(reusable))
		})

		It("should produce different names for different artifacts", func() {
			byName, _, err := ConstructArtifactDeploymentName(ArtifactName, ArtifactVersion, nil, 0)
			Expect(err).ToNot(HaveOccurred())
			byVersion, _, err := ConstructArtifactDeploymentName(ArtifactName, "2.0.0", nil, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(byName).ToNot(Equal(byVersion))
		})

		// Collision recovery contract: collisionCount == 0 must reproduce the unsalted name,
		// while any bump must produce a distinct name so the recovered
		// ArtifactDeployment no longer collides with the foreign one occupying the original name.
		It("should reproduce the unsalted name when collisionCount is zero", func() {
			unsalted, unsaltedHash, err := ConstructArtifactDeploymentName(ArtifactName, ArtifactVersion, nil, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(unsalted).To(Equal("kdenv-ui-1-0-0-zn9cvk5rd2"))
			Expect(unsaltedHash).To(Equal("zn9cvk5rd2"))
		})

		It("should produce distinct names for distinct collision salts", func() {
			seen := map[string]struct{}{}
			for count := int32(0); count <= 3; count++ {
				name, _, err := ConstructArtifactDeploymentName(ArtifactName, ArtifactVersion, nil, count)
				Expect(err).ToNot(HaveOccurred())
				Expect(seen).ToNot(HaveKey(name), "collision salt %d produced a duplicate name %q", count, name)
				seen[name] = struct{}{}
			}
		})

		It("should be deterministic for a given collision salt", func() {
			name1, hash1, err1 := ConstructArtifactDeploymentName(ArtifactName, ArtifactVersion, nil, 2)
			name2, hash2, err2 := ConstructArtifactDeploymentName(ArtifactName, ArtifactVersion, nil, 2)
			Expect(err1).ToNot(HaveOccurred())
			Expect(err2).ToNot(HaveOccurred())
			Expect(name1).To(Equal(name2))
			Expect(hash1).To(Equal(hash2))
		})

		// Stability guard — NOT a mere characterization test.
		//
		// These names are persisted as Kubernetes resource names in live clusters.
		// If the hashing/naming algorithm changes, previously-created
		// ArtifactDeployments become orphaned (a new name no longer resolves to the
		// existing object). Treat a break here as a migration concern, not a value
		// to blindly re-baseline. Do NOT update these strings without understanding
		// the upgrade impact on existing deployments.
		It("should remain stable across releases (orphaning guard)", func() {
			name, deploymentHash, err := ConstructArtifactDeploymentName(ArtifactName, ArtifactVersion, nil, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(name).To(Equal("kdenv-ui-1-0-0-zn9cvk5rd2"))
			Expect(deploymentHash).To(Equal("zn9cvk5rd2"))

			nameWithUID, hashWithUID, err := ConstructArtifactDeploymentName(ArtifactName, ArtifactVersion, new(UID), 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(nameWithUID).To(Equal("kdenv-ui-1-0-0-978crfm844"))
			Expect(hashWithUID).To(Equal("978crfm844"))
		})
	})
})
