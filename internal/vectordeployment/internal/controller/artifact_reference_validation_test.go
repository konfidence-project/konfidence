package controller

import (
	"context"
	"errors"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"ocm.software/open-component-model/bindings/go/oci/compref"
)

type vectorDescriptorStub struct {
	descriptor VectorDescriptor
	calls      int
}

func (s *vectorDescriptorStub) GetVectorDescriptor(_ context.Context, _ compref.Ref) (VectorDescriptor, error) {
	s.calls++
	return s.descriptor, nil
}

func (s *vectorDescriptorStub) GetArtifactManifestByReference(_ context.Context, _ compref.Ref) (ArtifactManifest, error) {
	return ArtifactManifest{}, errors.New("unexpected manifest request")
}

var _ = Describe("artifact reference validation", func() {
	It("accepts distinct artifact component names", func() {
		err := validateArtifactReferences([]compref.Ref{
			{Component: "example.org/service", Version: "1.0.0"},
			{Component: "example.org/database", Version: "1.0.0"},
		})

		Expect(err).NotTo(HaveOccurred())
	})

	It("rejects different versions of the same artifact", func() {
		err := validateArtifactReferences([]compref.Ref{
			{Component: "example.org/service", Version: "1.0.0"},
			{Component: "example.org/service", Version: "2.0.0"},
		})

		var permanentError *PermanentError
		Expect(errors.As(err, &permanentError)).To(BeTrue())
		Expect(permanentError.Reason).To(Equal(konfidence.VectorDeploymentStalledReasonDuplicateArtifact))
		Expect(err.Error()).To(ContainSubstring(`artifact "example.org/service" more than once`))
		Expect(err.Error()).To(ContainSubstring(`"1.0.0"`))
		Expect(err.Error()).To(ContainSubstring(`"2.0.0"`))
	})

	It("rejects the same artifact version more than once", func() {
		err := validateArtifactReferences([]compref.Ref{
			{Component: "example.org/service", Version: "1.0.0"},
			{Component: "example.org/service", Version: "1.0.0"},
		})

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`artifact "example.org/service" more than once`))
	})

	It("validates a freshly downloaded descriptor", func() {
		stub := &vectorDescriptorStub{
			descriptor: VectorDescriptor{
				References: []compref.Ref{
					{Component: "example.org/service", Version: "1.0.0"},
					{Component: "example.org/service", Version: "2.0.0"},
				},
				DescriptorJSON: []byte(`{"meta":{"schemaVersion":"v2"}}`),
			},
		}
		reconciler := &VectorDeploymentReconciler{OcmAdapter: stub}

		_, err := reconciler.resolveVector(context.Background(), &konfidence.VectorDeployment{}, compref.Ref{})

		Expect(err).To(HaveOccurred())
		Expect(stub.calls).To(Equal(1))

		var permanentError *PermanentError
		Expect(errors.As(err, &permanentError)).To(BeTrue())
		Expect(permanentError.Reason).To(Equal(konfidence.VectorDeploymentStalledReasonDuplicateArtifact))
	})

	It("validates references reconstructed from cached status", func() {
		stub := &vectorDescriptorStub{}
		reconciler := &VectorDeploymentReconciler{OcmAdapter: stub}
		vectorDeployment := &konfidence.VectorDeployment{
			Status: konfidence.VectorDeploymentStatus{
				ResolvedVectorOcm: `{
					"meta":{"schemaVersion":"v2"},
					"component":{
						"name":"example.org/vector",
						"version":"1.0.0",
						"provider":"example",
						"resources":[],
						"sources":[],
						"componentReferences":[
							{
								"name":"service-v1",
								"componentName":"example.org/service",
								"version":"1.0.0"
							},
							{
								"name":"service-v2",
								"componentName":"example.org/service",
								"version":"2.0.0"
							}
						]
					}
				}`,
			},
		}

		_, err := reconciler.resolveVector(context.Background(), vectorDeployment, compref.Ref{})

		Expect(err).To(HaveOccurred())
		Expect(stub.calls).To(BeZero())

		var permanentError *PermanentError
		Expect(errors.As(err, &permanentError)).To(BeTrue())
		Expect(permanentError.Reason).To(Equal(konfidence.VectorDeploymentStalledReasonDuplicateArtifact))
	})
})
