package vector

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

var _ = Describe("Vector.Clone", func() {
	It("mutations to cloned Artifacts do not affect the original", func() {
		original := Vector{
			Artifacts: []Artifact{
				{Name: "comp-a", Version: "1.0.0"},
				{Name: "comp-b", Version: "2.0.0"},
			},
		}

		cloned := original.Clone()
		cloned.Artifacts[0] = Artifact{Name: "comp-a", Version: "MUTATED"}
		cloned.Artifacts = append(cloned.Artifacts, Artifact{Name: "comp-c", Version: "3.0.0"})

		Expect(original.Artifacts).To(HaveLen(2))
		Expect(original.Artifacts[0].Version).To(Equal("1.0.0"))
	})

	It("mutations to cloned VectorConfig.Content do not affect the original", func() {
		original := Vector{
			VectorConfig: &VectorConfiguration{Content: []byte(`{"feature":"on"}`)},
		}

		cloned := original.Clone()
		cloned.VectorConfig.Content[0] = 'X'

		Expect(original.VectorConfig.Content[0]).To(Equal(byte('{')))
	})

	It("nil VectorConfig is preserved", func() {
		original := Vector{Artifacts: []Artifact{{Name: "a", Version: "1.0.0"}}}
		cloned := original.Clone()
		Expect(cloned.VectorConfig).To(BeNil())
	})
})

var _ = Describe("HasDrift", func() {

	It("has drift when artifacts lengths differ", func() {
		currentVector := Vector{
			Artifacts: make([]Artifact, 2),
		}
		desiredVector := Vector{
			Artifacts: make([]Artifact, 3),
		}
		Expect(HasDrift(currentVector, desiredVector)).To(BeTrue())
	})

	It("no drift when artifacts lengths are same and artifacts match", func() {
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

		currentArtifacts := []Artifact{
			{
				Version: "1.2.3",
				Name:    "component-a",
			},
			{
				Version: "1.52.0",
				Name:    "component-b",
			},
		}
		currentVector := Vector{
			Artifacts: currentArtifacts,
		}
		desiredVector := Vector{
			Artifacts: desiredArtifacts,
		}
		Expect(HasDrift(currentVector, desiredVector)).To(BeFalse())
	})

	It("has drift when artifacts lengths are same but artifacts differ", func() {
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
		currentArtifacts := []Artifact{
			{
				Version: "1.2.3",
				Name:    "component-a",
			},
			{
				Version: "1.60.5",
				Name:    "component-b",
			},
		}
		currentVector := Vector{
			Artifacts: currentArtifacts,
		}
		desiredVector := Vector{
			Artifacts: desiredArtifacts,
		}
		Expect(HasDrift(currentVector, desiredVector)).To(BeTrue())
	})

	It("has drift when an artifact is missing in current artifacts", func() {
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

		currentArtifacts := []Artifact{
			{
				Version: "1.2.3",
				Name:    "component-a",
			},
			{
				Version: "2.0.0",
				Name:    "component-c",
			},
		}
		currentVector := Vector{
			Artifacts: currentArtifacts,
		}
		desiredVector := Vector{
			Artifacts: desiredArtifacts,
		}
		Expect(HasDrift(currentVector, desiredVector)).To(BeTrue())
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

		currentArtifacts := []Artifact{
			{
				Version: "1.2.3",
				Name:    "component-a",
			},
			{
				Version: "1.52.0",
				Name:    "component-b",
			},
		}
		currentVector := Vector{
			Artifacts: currentArtifacts,
		}
		desiredVector := Vector{
			Artifacts: desiredArtifacts,
		}
		Expect(HasDrift(currentVector, desiredVector)).To(BeFalse())
	})

	It("has no drift when vector config does not exist and no new config should be created", func() {
		currentVector := Vector{
			Artifacts:    nil,
			VectorConfig: nil,
		}
		desiredVector := Vector{}
		Expect(HasDrift(currentVector, desiredVector)).To(BeFalse())
	})
	It("has drift when current vector config exists and no new config should be created", func() {
		currentVector := Vector{
			Artifacts:    nil,
			VectorConfig: &VectorConfiguration{Content: []byte("test")},
		}
		desiredVector := Vector{}
		Expect(HasDrift(currentVector, desiredVector)).To(BeTrue())
	})
	It("has drift when no current vector config exists and a new config should be created", func() {
		currentVector := Vector{
			Artifacts: nil,
		}
		desiredVector := Vector{
			VectorConfig: &VectorConfiguration{Content: []byte("test")},
		}
		Expect(HasDrift(currentVector, desiredVector)).To(BeTrue())
	})
	It("has drift when current vector config exists and content of new config differs", func() {
		currentVector := Vector{
			Artifacts:    nil,
			VectorConfig: &VectorConfiguration{Content: []byte("test2")},
		}
		desiredVector := Vector{
			VectorConfig: &VectorConfiguration{Content: []byte("test")},
		}
		Expect(HasDrift(currentVector, desiredVector)).To(BeTrue())
	})
	It("has no drift when current vector config equals new config", func() {
		currentVector := Vector{
			Artifacts:    nil,
			VectorConfig: &VectorConfiguration{Content: []byte("test")},
		}
		desiredVector := Vector{
			VectorConfig: &VectorConfiguration{Content: []byte("test")},
		}
		Expect(HasDrift(currentVector, desiredVector)).To(BeFalse())
	})
})
