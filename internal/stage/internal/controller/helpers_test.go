//nolint:staticcheck // ST1001: allow dot-import for test utils using Gomega
package controller

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func CreateStage(ctx context.Context, k8sClient client.Client, name string, namespace string, vectorName string) {
	stage := &konfidence.Stage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "konfidence.cloud/v1alpha1",
			Kind:       "Stage",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: konfidence.StageSpec{
			Vector: vectorName,
		},
	}

	Expect(k8sClient.Create(ctx, stage)).To(Succeed())
}

func GetStage(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *konfidence.Stage {
	stage := &konfidence.Stage{}
	stageLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, stageLookupKey, stage)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch stage: %s", name)
	return stage
}

func DeleteStage(ctx context.Context, k8sClient client.Client, stage *konfidence.Stage) {
	err := k8sClient.Delete(ctx, stage)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete stage: %s", stage.Name)
}

func GetStages(ctx context.Context, k8sClient client.Client) *konfidence.StageList {
	stages := &konfidence.StageList{}
	err := k8sClient.List(ctx, stages)

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch stages")
	return stages
}

func CleanupStages(k8sClient client.Client) {
	ctx := context.Background()
	stages := GetStages(ctx, k8sClient)

	for _, stage := range stages.Items {
		DeleteStage(ctx, k8sClient, &stage)
	}
}

func CreateStageVersion(ctx context.Context, k8sClient client.Client, stageName, name string, namespace string, vectorName string, adaptedVectorName string) {
	stageVersion := &konfidence.StageVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "konfidence.cloud/v1alpha1",
			Kind:       "StageVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				pkgctrl.StageNameLabel:       stageName,
				pkgctrl.VectorReferenceLabel: adaptedVectorName,
			},
		},
		Spec: konfidence.StageVersionSpec{
			Vector:          vectorName,
			StageGeneration: 1,
			StageRef: &konfidence.StageReference{
				Name: stageName,
			},
		},
	}

	Expect(k8sClient.Create(ctx, stageVersion)).To(Succeed())
}

func CreateStageVersionWithLabels(
	ctx context.Context,
	k8sClient client.Client,
	name string,
	namespace string,
	vectorName string,
	stageName string,
	adaptedVectorName string,
) {
	stageVersion := &konfidence.StageVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "konfidence.cloud/v1alpha1",
			Kind:       "StageVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				pkgctrl.StageNameLabel:       stageName,
				pkgctrl.VectorReferenceLabel: adaptedVectorName,
			},
		},
		Spec: konfidence.StageVersionSpec{
			Vector:          vectorName,
			StageGeneration: 1,
			StageRef: &konfidence.StageReference{
				Name: stageName,
			},
		},
	}

	Expect(k8sClient.Create(ctx, stageVersion)).To(Succeed())
}

func GetStageVersion(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *konfidence.StageVersion {
	stageVersion := &konfidence.StageVersion{}
	stageVersionLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch stageVersion: %s", name)
	return stageVersion
}

func DeleteStageVersion(ctx context.Context, k8sClient client.Client, stageVersion *konfidence.StageVersion) {
	err := k8sClient.Delete(ctx, stageVersion)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete stageVersion: %s", stageVersion.Name)
}

func GetStageVersions(ctx context.Context, k8sClient client.Client) *konfidence.StageVersionList {
	stageVersions := &konfidence.StageVersionList{}
	err := k8sClient.List(ctx, stageVersions)

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch stageVersions")
	return stageVersions
}
func CleanupStageVersions(k8sClient client.Client) {
	ctx := context.Background()
	stageVersions := GetStageVersions(ctx, k8sClient)

	for _, stageVersion := range stageVersions.Items {
		DeleteStageVersion(ctx, k8sClient, &stageVersion)
	}
}

func CleanupStageVersion(k8sClient client.Client, stageVersionName string, namespace string) {
	ctx := context.Background()
	stageVersion := GetStageVersion(ctx, k8sClient, stageVersionName, namespace, true)

	if stageVersion != nil {
		DeleteStageVersion(ctx, k8sClient, stageVersion)
	}
}

func CreateStageVersionUsage(ctx context.Context, k8sClient client.Client, name string, namespace string, stageVersionName string) {
	usage := &konfidence.StageVersionUsage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "konfidence.cloud/v1alpha1",
			Kind:       "StageVersionUsage",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: konfidence.StageVersionUsageSpec{
			StageVersionRef: &konfidence.StageVersionReference{
				Name: stageVersionName,
			},
		},
	}

	Expect(k8sClient.Create(ctx, usage)).To(Succeed())
}

// CreateActiveStageVersionUsage creates the active StageVersionUsage of a stage the same way
// the vectoractivation domain does: deterministic name, active label and a controller
// reference to the stage.
func CreateActiveStageVersionUsage(ctx context.Context, k8sClient client.Client, stage *konfidence.Stage, stageVersionName string) {
	createActiveStageVersionUsage(ctx, k8sClient, stage, &konfidence.StageVersionReference{Name: stageVersionName})
}

// CreateActiveStageVersionUsageWithSelector creates the active StageVersionUsage of a stage
// without a stageVersion reference: it resolves its stageVersion by selector instead. The CRD
// requires exactly one of both fields, so this is the shape a reference-less usage has.
func CreateActiveStageVersionUsageWithSelector(ctx context.Context, k8sClient client.Client, stage *konfidence.Stage) {
	createActiveStageVersionUsage(ctx, k8sClient, stage, nil)
}

func createActiveStageVersionUsage(ctx context.Context, k8sClient client.Client, stage *konfidence.Stage,
	stageVersionRef *konfidence.StageVersionReference,
) {
	usage := &konfidence.StageVersionUsage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "konfidence.cloud/v1alpha1",
			Kind:       "StageVersionUsage",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      konfidence.ActiveStageVersionUsageName(stage.Name),
			Namespace: stage.Namespace,
			Labels: map[string]string{
				konfidence.ActiveStageVersionLabel: stage.Name,
			},
		},
		Spec: konfidence.StageVersionUsageSpec{StageVersionRef: stageVersionRef},
	}
	if stageVersionRef == nil {
		usage.Spec.StageVersionSelector = activeStageVersionSelector(stage)
	}

	Expect(controllerutil.SetControllerReference(stage, usage, k8sClient.Scheme())).To(Succeed())
	Expect(k8sClient.Create(ctx, usage)).To(Succeed())
}

// UpdateActiveStageVersionUsageRef points the active StageVersionUsage of a stage at another
// stageVersion. The spec change bumps the usage generation, which triggers a stage reconcile.
func UpdateActiveStageVersionUsageRef(ctx context.Context, k8sClient client.Client, stage *konfidence.Stage, stageVersionName string) {
	usage := GetStageVersionUsage(ctx, k8sClient, konfidence.ActiveStageVersionUsageName(stage.Name), stage.Namespace, false)
	usage.Spec.StageVersionRef = &konfidence.StageVersionReference{Name: stageVersionName}
	usage.Spec.StageVersionSelector = nil

	Expect(k8sClient.Update(ctx, usage)).To(Succeed())
}

// ClearActiveStageVersionUsageRef drops the stageVersion reference of a stage's active
// StageVersionUsage in favour of a selector. The spec change bumps the usage generation,
// which triggers a stage reconcile.
func ClearActiveStageVersionUsageRef(ctx context.Context, k8sClient client.Client, stage *konfidence.Stage) {
	usage := GetStageVersionUsage(ctx, k8sClient, konfidence.ActiveStageVersionUsageName(stage.Name), stage.Namespace, false)
	usage.Spec.StageVersionRef = nil
	usage.Spec.StageVersionSelector = activeStageVersionSelector(stage)

	Expect(k8sClient.Update(ctx, usage)).To(Succeed())
}

func activeStageVersionSelector(stage *konfidence.Stage) *metav1.LabelSelector {
	return &metav1.LabelSelector{MatchLabels: map[string]string{pkgctrl.StageNameLabel: stage.Name}}
}

func CreateStageVersionUsageWithSelector(
	ctx context.Context,
	k8sClient client.Client,
	name string,
	namespace string,
	stageName string,
	vectorRef string,
	isTarget bool,
) {
	usage := &konfidence.StageVersionUsage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "konfidence.cloud/v1alpha1",
			Kind:       "StageVersionUsage",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: konfidence.StageVersionUsageSpec{
			StageVersionSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					pkgctrl.StageNameLabel:       stageName,
					pkgctrl.VectorReferenceLabel: vectorRef,
				},
			},
		},
	}

	if isTarget {
		usage.SetLabels(map[string]string{
			pkgctrl.StageVersionUsageTarget: stageName,
		})
	}

	Expect(k8sClient.Create(ctx, usage)).To(Succeed())
}

func GetStageVersionUsage(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *konfidence.StageVersionUsage {
	stageVersionUsage := &konfidence.StageVersionUsage{}
	stageVersionUsageLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch stageVersionUsage: %s", name)
	return stageVersionUsage
}

func DeleteStageVersionUsage(ctx context.Context, k8sClient client.Client, stageVersionUsage *konfidence.StageVersionUsage) {
	err := k8sClient.Delete(ctx, stageVersionUsage)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete stageVersionUsage: %s", stageVersionUsage.Name)
}

func CleanupStageVersionUsage(k8sClient client.Client, stageVersionUsageName string, namespace string) {
	ctx := context.Background()
	stageVersionUsage := GetStageVersionUsage(ctx, k8sClient, stageVersionUsageName, namespace, true)

	if stageVersionUsage != nil {
		DeleteStageVersionUsage(ctx, k8sClient, stageVersionUsage)
	}
}

func GetStageVersionUsages(ctx context.Context, k8sClient client.Client) *konfidence.StageVersionUsageList {
	stageVersionUsages := &konfidence.StageVersionUsageList{}
	err := k8sClient.List(ctx, stageVersionUsages)

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch stageVersionUsages")
	return stageVersionUsages
}

func CleanupStageVersionUsages(k8sClient client.Client) {
	ctx := context.Background()
	stageVersionUsages := GetStageVersionUsages(ctx, k8sClient)

	for _, stageVersionUsage := range stageVersionUsages.Items {
		DeleteStageVersionUsage(ctx, k8sClient, &stageVersionUsage)
	}
}

func GetVectorDeployment(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *konfidence.VectorDeployment {
	vectorDeployment := &konfidence.VectorDeployment{}
	vectorDeploymentLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch vectorDeployment: %s", name)
	return vectorDeployment
}

func DeleteVectorDeployment(ctx context.Context, k8sClient client.Client, vectorDeployment *konfidence.VectorDeployment) {
	err := k8sClient.Delete(ctx, vectorDeployment)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete vectorDeployment: %s", vectorDeployment.Name)
}

func CleanupVectorDeployment(k8sClient client.Client, vectorDeploymentName string, namespace string) {
	ctx := context.Background()
	vectorDeployment := GetVectorDeployment(ctx, k8sClient, vectorDeploymentName, namespace, true)

	if vectorDeployment != nil {
		DeleteVectorDeployment(ctx, k8sClient, vectorDeployment)
	}
}

func GetVectorDeployments(ctx context.Context, k8sClient client.Client) *konfidence.VectorDeploymentList {
	vectorDeployments := &konfidence.VectorDeploymentList{}
	err := k8sClient.List(ctx, vectorDeployments)

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch vectorDeployments")
	return vectorDeployments
}

func CleanupVectorDeployments(k8sClient client.Client) {
	ctx := context.Background()
	vectorDeployments := GetVectorDeployments(ctx, k8sClient)

	for _, vectorDeployment := range vectorDeployments.Items {
		DeleteVectorDeployment(ctx, k8sClient, &vectorDeployment)
	}
}

func GetVectorMigration(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *konfidence.VectorMigration {
	vectorMigration := &konfidence.VectorMigration{}
	vectorMigrationLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch vectorMigration: %s", name)
	return vectorMigration
}

func DeleteVectorMigration(ctx context.Context, k8sClient client.Client, vectorMigration *konfidence.VectorMigration) {
	err := k8sClient.Delete(ctx, vectorMigration)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete vectorMigration: %s", vectorMigration.Name)
}

func CleanupVectorMigration(k8sClient client.Client, vectorMigrationName string, namespace string) {
	ctx := context.Background()
	vectorMigration := GetVectorMigration(ctx, k8sClient, vectorMigrationName, namespace, true)

	if vectorMigration != nil {
		DeleteVectorMigration(ctx, k8sClient, vectorMigration)
	}
}

func GetVectorMigrations(ctx context.Context, k8sClient client.Client) *konfidence.VectorMigrationList {
	vectorMigrations := &konfidence.VectorMigrationList{}
	err := k8sClient.List(ctx, vectorMigrations)

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch vectorMigrations")
	return vectorMigrations
}

func CleanupVectorMigrations(k8sClient client.Client) {
	ctx := context.Background()
	vectorMigrations := GetVectorMigrations(ctx, k8sClient)

	for _, vectorMigration := range vectorMigrations.Items {
		DeleteVectorMigration(ctx, k8sClient, &vectorMigration)
	}
}

func GetVectorActivation(ctx context.Context, k8sClient client.Client, name string, namespace string, opt bool) *konfidence.VectorActivation {
	vectorActivation := &konfidence.VectorActivation{}
	vectorActivationLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	err := k8sClient.Get(ctx, vectorActivationLookupKey, vectorActivation)

	if opt && err != nil && errors.IsNotFound(err) {
		return nil
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch vectorActivation: %s", name)
	return vectorActivation
}

func DeleteVectorActivation(ctx context.Context, k8sClient client.Client, vectorActivation *konfidence.VectorActivation) {
	err := k8sClient.Delete(ctx, vectorActivation)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete vectorActivation: %s", vectorActivation.Name)
}

func CleanupVectorActivation(k8sClient client.Client, vectorActivationName string, namespace string) {
	ctx := context.Background()
	vectorActivation := GetVectorActivation(ctx, k8sClient, vectorActivationName, namespace, true)

	if vectorActivation != nil {
		DeleteVectorActivation(ctx, k8sClient, vectorActivation)
	}
}

func GetVectorActivations(ctx context.Context, k8sClient client.Client) *konfidence.VectorActivationList {
	vectorActivations := &konfidence.VectorActivationList{}
	err := k8sClient.List(ctx, vectorActivations)

	Expect(err).ToNot(HaveOccurred(), "Failed to fetch vectorActivations")
	return vectorActivations
}

func CleanupVectorActivations(k8sClient client.Client) {
	ctx := context.Background()
	vectorActivations := GetVectorActivations(ctx, k8sClient)

	for _, vectorActivation := range vectorActivations.Items {
		DeleteVectorActivation(ctx, k8sClient, &vectorActivation)
	}
}

func CleanupResources(k8sClient client.Client) {
	CleanupStageVersionUsages(k8sClient)
	CleanupStages(k8sClient)
	CleanupStageVersions(k8sClient)
	CleanupVectorDeployments(k8sClient)
	CleanupVectorMigrations(k8sClient)
	CleanupVectorActivations(k8sClient)
}

func HasOwnerReference(ownerReferences []metav1.OwnerReference, ref metav1.OwnerReference) bool {
	for _, ownerReference := range ownerReferences {
		if ownerReference.Kind == ref.Kind && ownerReference.Name == ref.Name {
			return true
		}
	}

	return false
}
