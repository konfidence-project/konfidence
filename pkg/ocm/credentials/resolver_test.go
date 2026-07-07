package credentials_test

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	rsav1alpha1 "ocm.software/open-component-model/bindings/go/rsa/signing/v1alpha1"
	rsaidentityv1 "ocm.software/open-component-model/bindings/go/rsa/spec/identity/v1"
	ocmruntime "ocm.software/open-component-model/bindings/go/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"github.com/konfidence-project/konfidence/pkg/ocm/credentials"
	testocm "github.com/konfidence-project/konfidence/pkg/testutil/ocm"
	"github.com/konfidence-project/konfidence/pkg/testutil/pki"
)

var _ = Describe("ResolverFromCredentials", func() {
	It("returns nil resolver when creds is nil", func() {
		r, err := credentials.ResolverFromCredentials(context.Background(), nil, "default", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(r).To(BeNil())
	})

	It("returns nil resolver when creds.OCM is nil", func() {
		r, err := credentials.ResolverFromCredentials(context.Background(), nil, "default", &galaxy.Credentials{OCM: nil})
		Expect(err).NotTo(HaveOccurred())
		Expect(r).To(BeNil())
	})

	It("propagates not-found error when referenced Secret does not exist", func() {
		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		creds := &galaxy.Credentials{
			OCM: &galaxy.OCMCredentials{
				Refs: []galaxy.CredentialRef{{Name: "does-not-exist"}},
			},
		}
		_, err := credentials.ResolverFromCredentials(context.Background(), fakeClient, "default", creds)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("does-not-exist"))
	})
})

var _ = Describe("ResolverFromRefs", func() {
	It("returns nil resolver when refs is empty", func() {
		r, err := credentials.ResolverFromRefs(context.Background(), nil, "default", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(r).To(BeNil())
	})

	It("returns nil resolver when refs is an empty slice", func() {
		r, err := credentials.ResolverFromRefs(context.Background(), nil, "default", []credentials.Ref{})
		Expect(err).NotTo(HaveOccurred())
		Expect(r).To(BeNil())
	})

	It("returns error when Secret has no recognized credential key", func() {
		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "bad-secret", Namespace: "default"},
			Data:       map[string][]byte{"unrecognized-key": []byte("garbage")},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

		_, err := credentials.ResolverFromRefs(context.Background(), fakeClient, "default", []credentials.Ref{{Name: "bad-secret"}})
		Expect(err).To(HaveOccurred())
	})

	It("propagates not-found error when referenced Secret does not exist", func() {
		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		_, err := credentials.ResolverFromRefs(context.Background(), fakeClient, "default", []credentials.Ref{{Name: "missing"}})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing"))
	})

	It("flat-merges .ocmconfig and .dockerconfigjson Secrets into one resolver that answers RSA identity", func() {
		ctx := context.Background()
		pair := pki.GenerateRSAKeyPair("test-merger-cn")
		const sigName = "v-sig-merge"

		ocmSecret := testocm.OCMConfigSecret("ocm-creds", "default", testocm.Bind(sigName, pair))
		dockerSecret := testocm.DockerConfigSecret("docker-creds", "default", "user", "pass", "registry.example.com")

		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ocmSecret, dockerSecret).Build()

		resolver, err := credentials.ResolverFromRefs(ctx, fakeClient, "default", []credentials.Ref{
			{Name: "ocm-creds"},
			{Name: "docker-creds"},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resolver).NotTo(BeNil())

		rsaID := ocmruntime.Identity{
			rsaidentityv1.IdentityAttributeSignature: sigName,
			rsaidentityv1.IdentityAttributeAlgorithm: string(rsav1alpha1.AlgorithmRSASSAPSS),
		}
		rsaID.SetType(rsaidentityv1.V1Alpha1Type)

		resolved, err := resolver.Resolve(ctx, rsaID)
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved).NotTo(BeNil())

		raw, err := json.Marshal(resolved)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(raw)).To(ContainSubstring("privateKeyPEM"))
	})
})
