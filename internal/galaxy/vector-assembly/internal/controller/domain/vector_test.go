package domain

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ociv1 "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/oci"
)

func TestDomainVector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "vector domain Suite")
}

var _ = Describe("HasDrift", func() {

	It("has drift when lengths differ", func() {
		desiredArtifacts := make([]Artifact, 3)
		actualArtifacts := make([]Artifact, 2)
		Expect(HasDrift(desiredArtifacts, actualArtifacts)).To(BeTrue())
	})

	It("no drift when lengths are same and artifacts match", func() {
		desiredArtifacts := []Artifact{
			{
				Version: "1.2.3",
				Name:    "component-a",
			},
			{
				Version: "1.52.0",
				Name:    "component-b",
			},
		}

		actualArtifacts := []Artifact{
			{
				Version: "1.2.3",
				Name:    "component-a",
			},
			{
				Version: "1.52.0",
				Name:    "component-b",
			},
		}
		Expect(HasDrift(desiredArtifacts, actualArtifacts)).To(BeFalse())
	})

	It("has drift when lengths are same but artifacts differ", func() {
		desiredArtifacts := []Artifact{
			{
				Version: "1.2.3",
				Name:    "component-a",
			},
			{
				Version: "1.52.0",
				Name:    "component-b",
			},
		}
		actualArtifacts := []Artifact{
			{
				Version: "1.2.3",
				Name:    "component-a",
			},
			{
				Version: "1.60.5",
				Name:    "component-b",
			},
		}
		Expect(HasDrift(desiredArtifacts, actualArtifacts)).To(BeTrue())
	})

	It("has drift when an artifact is missing in actual", func() {
		desiredArtifacts := []Artifact{
			{
				Version: "1.2.3",
				Name:    "component-a",
			},
			{
				Version: "1.52.0",
				Name:    "component-b",
			},
		}

		actualArtifacts := []Artifact{
			{
				Version: "1.2.3",
				Name:    "component-a",
			},
			{
				Version: "2.0.0",
				Name:    "component-c",
			},
		}
		Expect(HasDrift(desiredArtifacts, actualArtifacts)).To(BeTrue())
	})

	It("no drift when SourceRepo differs but Name and Version match", func() {
		desiredArtifacts := []Artifact{
			{
				Version:    "1.2.3",
				Name:       "component-a",
				SourceRepo: &ociv1.Repository{BaseUrl: "http://localhost:5100"},
			},
			{
				Version:    "1.52.0",
				Name:       "component-b",
				SourceRepo: &ociv1.Repository{BaseUrl: "http://localhost:5200"},
			},
		}

		actualArtifacts := []Artifact{
			{
				Version: "1.2.3",
				Name:    "component-a",
			},
			{
				Version: "1.52.0",
				Name:    "component-b",
			},
		}
		Expect(HasDrift(desiredArtifacts, actualArtifacts)).To(BeFalse())
	})
})
