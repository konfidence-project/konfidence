package vectordeployment

import (
	"os"

	"github.com/konfidence-project/konfidence/internal/vectordeployment/internal/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artifact manifest concurrency configuration", Serial, func() {
	const envName = "KONFIDENCE_TEST_MANIFEST_LIMIT"

	BeforeEach(func() {
		original, existed := os.LookupEnv(envName)
		Expect(os.Unsetenv(envName)).To(Succeed())

		DeferCleanup(func() {
			if existed {
				Expect(os.Setenv(envName, original)).To(Succeed())
			} else {
				Expect(os.Unsetenv(envName)).To(Succeed())
			}
		})
	})

	It("uses the fallback when neither option nor environment is set", func() {
		value, err := parseManifestDownloadLimit(0, envName, 8)
		Expect(err).NotTo(HaveOccurred())
		Expect(value).To(Equal(8))
	})

	It("reads and trims the environment value", func() {
		Expect(os.Setenv(envName, " 12 ")).To(Succeed())

		value, err := parseManifestDownloadLimit(0, envName, 8)
		Expect(err).NotTo(HaveOccurred())
		Expect(value).To(Equal(12))
	})

	It("prefers an explicit option over the environment", func() {
		Expect(os.Setenv(envName, "not-an-integer")).To(Succeed())

		value, err := parseManifestDownloadLimit(16, envName, 8)
		Expect(err).NotTo(HaveOccurred())
		Expect(value).To(Equal(16))
	})

	DescribeTable("validates configured values",
		func(option int, environment string, expected int, wantError bool) {
			if environment != "" {
				Expect(os.Setenv(envName, environment)).To(Succeed())
			}

			value, err := parseManifestDownloadLimit(option, envName, 8)
			if wantError {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(envName))
				return
			}

			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal(expected))
		},

		Entry("minimum option", 1, "", 1, false),
		Entry("maximum option", 128, "", 128, false),
		Entry("negative option", -1, "", 0, true),
		Entry("option above maximum", 129, "", 0, true),
		Entry("zero environment", 0, "0", 0, true),
		Entry("negative environment", 0, "-1", 0, true),
		Entry("environment above maximum", 0, "129", 0, true),
		Entry("malformed environment", 0, "invalid", 0, true),
	)

	It("uses the production defaults", func() {
		perDeployment, err := parseManifestDownloadLimit(
			0,
			MaxArtifactManifestDownloadsPerVectorDeploymentEnv,
			controller.DefaultArtifactManifestDownloadsPerVectorDeployment,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(perDeployment).To(Equal(8))

		global, err := parseManifestDownloadLimit(
			0,
			MaxGlobalArtifactManifestDownloadsEnv,
			controller.DefaultGlobalArtifactManifestDownloads,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(global).To(Equal(32))
	})
})
