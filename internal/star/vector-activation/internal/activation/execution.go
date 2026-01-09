package activation

import (
	"context"
	"fmt"

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// ListExecutionsForRegistration gets all executions for the given registration and activation
func ListExecutionsForRegistration(ctx context.Context, c client.Client, namespace string, registration landscape.ActivationTaskRegistration, vectorActivation *landscape.VectorActivation) (*landscape.ActivationTaskExecutionList, error) {
	executionLabels := client.MatchingLabels{
		"registration": registration.Name,
		"activation":   vectorActivation.Name,
	}
	executionList := &landscape.ActivationTaskExecutionList{}
	if err := c.List(ctx, executionList, client.InNamespace(namespace), executionLabels); err != nil {
		return nil, fmt.Errorf("failed to list ActivationExecutions for labelSelector %s: %w", executionLabels, err)
	}
	return executionList, nil
}

// CreateExecution creates an ActivationTaskExecution for the given registration
func CreateExecution(ctx context.Context, c client.Client, namespace string, vectorActivation *landscape.VectorActivation, registration landscape.ActivationTaskRegistration) (*landscape.ActivationTaskExecution, error) {
	log := logf.FromContext(ctx)
	activationExecution := &landscape.ActivationTaskExecution{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "activation-execution-",
			Namespace:    namespace,
			Labels: map[string]string{
				"registration": registration.Name,
				"activation":   vectorActivation.Name,
			},
		},
		Spec: landscape.ActivationTaskExecutionSpec{
			Type:             registration.Spec.Type,
			VectorActivation: vectorActivation.Name,
			Spec:             registration.Spec.Spec,
		},
	}
	if err := controllerutil.SetControllerReference(vectorActivation, activationExecution, c.Scheme()); err != nil {
		return nil, fmt.Errorf("failed to set owner reference on activationExecution: %w", err)
	}
	if err := c.Create(ctx, activationExecution); err != nil {
		return nil, fmt.Errorf("failed to create ActivationExecution for registration %s: %w", registration.Name, err)
	}
	log.Info("created ActivationExecution for registration", "registration", registration.Name, "activation", activationExecution.Name)
	return activationExecution, nil
}

// EnsureExecutionsForRegistrations ensures that for each registration in the list there is an execution created for the given activation
func EnsureExecutionsForRegistrations(ctx context.Context, c client.Client, namespace string, registrationList *landscape.ActivationTaskRegistrationList, vectorActivation *landscape.VectorActivation) (*landscape.ActivationTaskExecutionList, error) {
	allExecutions := &landscape.ActivationTaskExecutionList{}
	for _, registration := range registrationList.Items {
		executionList, err := ListExecutionsForRegistration(ctx, c, namespace, registration, vectorActivation)
		if err != nil {
			return nil, err
		}
		allExecutions.Items = append(allExecutions.Items, executionList.Items...)
		if len(executionList.Items) == 0 {
			newExecution, err := CreateExecution(ctx, c, namespace, vectorActivation, registration)
			if err != nil {
				return nil, err
			}
			allExecutions.Items = append(allExecutions.Items, *newExecution)
		}
	}
	return allExecutions, nil
}
