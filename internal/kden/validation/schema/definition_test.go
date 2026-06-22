package schema_test

import (
	"os"

	"github.com/konfidence-project/konfidence/internal/kden/validation/schema"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArtifactConstructor", func() {
	It("generates a schema that matches the current artifact validation JSON schema", func() {
		want, err := os.ReadFile("../resources/konfidence-artifact-schema.json")
		Expect(err).NotTo(HaveOccurred())

		got, err := schema.MarshalJSONSchema()
		Expect(err).NotTo(HaveOccurred())

		Expect(string(got)).To(Equal(string(want)), "schema is stale or invalid")
	})
})
