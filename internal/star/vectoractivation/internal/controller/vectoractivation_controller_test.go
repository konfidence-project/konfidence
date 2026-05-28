package controller

import (
	"context"
	"fmt"
	"time"

	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/star/vectoractivation/internal/usage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/coordination/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	StageName            = "stage-dev"
	StageVersionName     = "stage-version-12345"
	VectorActivationName = "activation-12345"
	Vector001            = "https://registry.kdenv.lab/ocm/vector//star.konfidence.cloud/example/vector:0.0.1"
	RegistrationName     = "registration-1"
	RegistrationType     = "registration-type-1"
	Namespace            = "default"
	VectorDeploymentName = "vector-deployment-1"
	timeout              = time.Second * 10
	interval             = time.Millisecond * 150
)

var _ = Describe("VectorActivation Controller", func() {
	BeforeEach(func() {
		ctx = context.Background()
	})

	AfterEach(func() {
		Delete[*star.Stage](ctx, k8sClient, StageName, Namespace)
		Delete[*star.StageVersion](ctx, k8sClient, StageVersionName, Namespace)
		Delete[*star.VectorActivation](ctx, k8sClient, VectorActivationName, Namespace)
		DeleteAll[*v1.Lease, *v1.LeaseList](ctx, k8sClient, client.InNamespace(Namespace))
		DeleteAll[*star.StageVersionUsage, *star.StageVersionUsageList](ctx, k8sClient, client.InNamespace(Namespace))
		Delete[*star.ActivationTaskRegistration](ctx, k8sClient, RegistrationName, Namespace)
	})

	Context("When reconciling a vector activation", func() {
		It("should successfully reconcile the vector activation", func() {

			SetupResources()

			vectorActivation := CreateVectorActivation()

			// assert lease object
			Eventually(func(g Gomega) {
				leaseName := fmt.Sprintf("vectoractivation-%s-lock", StageName)
				lease := &v1.Lease{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: leaseName, Namespace: Namespace}, lease)).To(Succeed())
				g.Expect(lease).ToNot(BeNil(), "expected Lease to be created for VectorActivationName")
				g.Expect(lease.OwnerReferences).ToNot(BeEmpty(), "expected lease to have owner references")
				g.Expect(lease.OwnerReferences[0].Name).To(Equal(StageName))
				g.Expect(lease.Spec.HolderIdentity).ToNot(BeNil(), "expected lease holder identity to be set")
				g.Expect(*lease.Spec.HolderIdentity).To(Equal(fmt.Sprintf("%s-%s", ActivationControllerName, vectorActivation.UID)))
				g.Expect(lease.Namespace).To(Equal(Namespace))
			}, timeout, interval).Should(Succeed())

			// assert activation usage
			Eventually(func(g Gomega) {
				activationUsageLabels := client.MatchingLabels{usage.ActivationStageVersionUsage: StageName}
				usageList := List[*star.StageVersionUsageList](
					ctx, k8sClient, &star.StageVersionUsageList{}, client.InNamespace(Namespace), activationUsageLabels,
				)

				g.Expect(usageList.Items).To(HaveLen(1))
				activationUsage := &usageList.Items[0]

				g.Expect(activationUsage).ToNot(BeNil(), "expected ActivationUsage to be created for VectorActivationName")
				g.Expect(activationUsage.Name).To(ContainSubstring("activation-"))
				g.Expect(activationUsage.OwnerReferences[0].Kind).To(Equal(vectorActivation.Kind))
				g.Expect(activationUsage.OwnerReferences[0].UID).To(Equal(vectorActivation.UID))
				g.Expect(activationUsage.Namespace).To(Equal(Namespace))
			}, timeout, interval).Should(Succeed())

			// assert status is InProgress
			Eventually(func(g Gomega) {
				vectorActivation = Get(ctx, k8sClient, VectorActivationName, Namespace, &star.VectorActivation{}, false)
				g.Expect(vectorActivation.Status.Conditions).ToNot(BeEmpty())
				latestCondition := vectorActivation.Status.Conditions[len(vectorActivation.Status.Conditions)-1]
				g.Expect(latestCondition.Type).To(Equal(star.ActivationInProgress))
			}, timeout, interval).Should(Succeed())

			// assert activationExecutions are created
			Eventually(func(g Gomega) {
				executionList := &star.ActivationTaskExecutionList{}
				g.Expect(k8sClient.List(ctx, executionList, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(executionList.Items).ToNot(BeEmpty(), "expected ActivationTaskExecutions to be created for VectorActivationName")
				for _, execution := range executionList.Items {
					g.Expect(execution.Name).To(HavePrefix(RegistrationName + "-"))
					g.Expect(execution.Status.Conditions).To(BeEmpty())
					g.Expect(execution.OwnerReferences).ToNot(BeEmpty())
					g.Expect(execution.OwnerReferences[0].UID).To(Equal(vectorActivation.UID))
				}
			}, timeout, interval).Should(Succeed())

			// ACT: update status of executions to Succeeded
			Eventually(func(g Gomega) {
				executionList := &star.ActivationTaskExecutionList{}
				g.Expect(k8sClient.List(ctx, executionList, client.InNamespace(Namespace))).To(Succeed())
				g.Expect(executionList.Items).ToNot(BeEmpty())
				for _, execution := range executionList.Items {
					updated := execution.DeepCopy()
					meta.SetStatusCondition(&updated.Status.Conditions, metav1.Condition{
						Type:               star.ActivationTaskExecutionSucceeded,
						Status:             metav1.ConditionTrue,
						Reason:             star.ActivationTaskExecutionSucceeded,
						Message:            "marked succeeded by test",
						ObservedGeneration: updated.Generation,
						LastTransitionTime: metav1.Now(),
					})
					g.Expect(k8sClient.Status().Patch(ctx, updated, client.MergeFrom(&execution))).To(Succeed())
				}
			}, timeout, interval).Should(Succeed())

			// assert active usage is created (or updated)
			Eventually(func(g Gomega) {
				activeUsageLabels := client.MatchingLabels{usage.ActiveStageVersion: StageName}
				usageList := &star.StageVersionUsageList{}
				g.Expect(k8sClient.List(ctx, usageList, client.InNamespace(Namespace), activeUsageLabels)).To(Succeed())
				g.Expect(usageList.Items).To(HaveLen(1))
				activeUsage := usageList.Items[0]
				g.Expect(activeUsage.Name).To(Equal(fmt.Sprintf("%s-active-usage", StageName)))
				g.Expect(activeUsage.Spec.StageVersionRef.Name).To(Equal(StageVersionName))
				g.Expect(activeUsage.Spec.Reason).To(Equal(usage.StageVersionUsageActiveType))
				g.Expect(activeUsage.OwnerReferences).ToNot(BeEmpty(), "expected active usage to have owner reference")
				g.Expect(activeUsage.OwnerReferences[0].Name).To(Equal(StageName))
			}, timeout, interval).Should(Succeed())

			// assert activation usage is deleted
			Eventually(func(g Gomega) {
				activationUsageLabels := client.MatchingLabels{usage.ActivationStageVersionUsage: StageName}
				usageList := &star.StageVersionUsageList{}
				g.Expect(k8sClient.List(ctx, usageList, client.InNamespace(Namespace), activationUsageLabels)).To(Succeed())
				g.Expect(usageList.Items).To(BeEmpty(), "expected activation usage to be deleted after success")
			}, timeout, interval).Should(Succeed())

			// assert status is Succeeded
			Eventually(func(g Gomega) {
				vectorActivation := &star.VectorActivation{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: VectorActivationName, Namespace: Namespace}, vectorActivation)).To(Succeed())
				g.Expect(vectorActivation).ToNot(BeNil())
				g.Expect(vectorActivation.Status.Conditions).ToNot(BeEmpty())
				g.Expect(meta.IsStatusConditionTrue(vectorActivation.Status.Conditions, star.ActivationSucceeded)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// assert lock is released
			Eventually(func(g Gomega) {
				leaseName := fmt.Sprintf("vectoractivation-%s-lock", StageName)
				lease := &v1.Lease{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: leaseName, Namespace: Namespace}, lease)).To(Succeed())
				g.Expect(lease.Spec.HolderIdentity).To(BeNil())
				g.Expect(lease.Spec.RenewTime).To(BeNil())
				g.Expect(lease.Spec.AcquireTime).To(BeNil())
			}, timeout, interval).Should(Succeed())
		})
	})
})

func CreateVectorActivation() *star.VectorActivation {
	vectorActivation := &star.VectorActivation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "star.konfidence.cloud/v1alpha1",
			Kind:       "VectorActivation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      VectorActivationName,
			Namespace: Namespace,
		},
		Spec: star.VectorActivationSpec{
			Vector:           Vector001,
			StageVersion:     StageVersionName,
			Stage:            StageName,
			VectorDeployment: VectorDeploymentName,
		},
	}
	Eventually(func(g Gomega) {
		Create(ctx, k8sClient, vectorActivation)
	}, timeout, interval).Should(Succeed())

	Eventually(func(g Gomega) {
		vectorActivation = &star.VectorActivation{}
		g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: VectorActivationName, Namespace: Namespace}, vectorActivation)).To(Succeed())
	}, timeout, interval).Should(Succeed())
	return vectorActivation
}

// SetupResources creates Resources that can be expected to be present when an activation gets reconciled (e.g.  Stage and StageVersion)
func SetupResources() {
	Eventually(func(g Gomega) {
		stage := &star.Stage{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "star.konfidence.cloud/v1alpha1",
				Kind:       "Stage",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      StageName,
				Namespace: Namespace,
			},
			Spec: star.StageSpec{
				Vector: Vector001,
			},
		}
		Create(ctx, k8sClient, stage)
		Get(ctx, k8sClient, StageName, Namespace, stage, false)

		stageVersion := &star.StageVersion{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "star.konfidence.cloud/v1alpha1",
				Kind:       "StageVersion",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      StageVersionName,
				Namespace: Namespace,
			},
			Spec: star.StageVersionSpec{
				Vector:          Vector001,
				StageGeneration: 1,
				StageRef: &star.StageReference{
					Name: StageName,
				},
			},
		}
		Create(ctx, k8sClient, stageVersion)
		Get(ctx, k8sClient, StageVersionName, Namespace, stageVersion, false)
		g.Expect(controllerutil.SetOwnerReference(stage, stageVersion, k8sClient.Scheme())).To(Succeed())
		Update(ctx, k8sClient, stageVersion)

		activationTaskRegistration := &star.ActivationTaskRegistration{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "star.konfidence.cloud/v1alpha1",
				Kind:       "ActivationTaskRegistration",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      RegistrationName,
				Namespace: Namespace,
			},
			Spec: star.ActivationTaskRegistrationSpec{
				Type: RegistrationType,
				Spec: runtime.RawExtension{Raw: []byte("{}")},
			},
		}
		Create(ctx, k8sClient, activationTaskRegistration)
		Get(ctx, k8sClient, RegistrationName, Namespace, activationTaskRegistration, false)
		registrationList := &star.ActivationTaskRegistrationList{}
		List(ctx, k8sClient, registrationList)
		g.Expect(registrationList.Items).ToNot(BeEmpty())
	}, timeout, interval).Should(Succeed())
}
