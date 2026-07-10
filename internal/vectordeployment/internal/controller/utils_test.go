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

	Context("Construct Artifact Deployment Name", func() {
		It("should successfully construct name without uid", func() {
			name, err := ConstructArtifactDeploymentName(ArtifactName, ArtifactVersion, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(name).To(Equal("kdenv-ui-1-0-0-cniur9k0oumsey7kp3jovtf8a"))
		})
		It("should successfully construct name with uid", func() {
			name, err := ConstructArtifactDeploymentName(ArtifactName, ArtifactVersion, new(UID))
			Expect(err).ToNot(HaveOccurred())
			Expect(name).To(Equal("kdenv-ui-1-0-0-2iiib8ffrzz9sttaa5wmayf17"))
		})
		It("should fail if artifact name is empty", func() {
			_, err := ConstructArtifactDeploymentName("   ", ArtifactVersion, nil)
			Expect(err).To(MatchError("artifact name or version is empty"))
		})
		It("should only use hash value as name if version with hash is too long", func() {
			name, err := ConstructArtifactDeploymentName(ArtifactName, "1gamma.0beta.0alpha-test-1234567890-delta-theta-3456999", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(name).To(Equal("4rdbzqzh9f2zddqw9zia1n78y"))
		})
		It("should use artifact name as is with hash value if no slash separator exists in artifact name", func() {
			name, err := ConstructArtifactDeploymentName(ArtifactNameWithoutSlash, ArtifactVersion, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(name).To(Equal("kdenv-ui-1-0-0-6ee9kgrr2f4tfk4xoxqti818p"))
		})
		It("should use artifact name plus dash with hash value if name ends with slash separator", func() {
			name, err := ConstructArtifactDeploymentName(ArtifactNameWithSlashTerminator, ArtifactVersion, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(name).To(Equal("kdenv-ui--1-0-0-3rbh3e6szd9rqu7xdtwmjfpya"))
		})
		It("should truncate artifact name if max length is exceeded", func() {
			name, err := ConstructArtifactDeploymentName(ArtifactNameWithLongName, ArtifactVersion, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(name).To(Equal("kdenv-angular-user-interface-de-1-0-0-8z8hxz9nafidtjpfak2ea3a9f"))
		})
	})
})
