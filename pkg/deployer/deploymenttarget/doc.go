// Package deploymenttarget provides a controller for deployers to reconcile DeploymentTarget resources following
// section 2 of the Konfidence deployer specification: https://github.com/konfidence-project/konfidence/blob/main/conformance/specs/deployer.md
//
// The intended use of this package looks like this:
//
//	deploymenttarget.NewControllerManagedBy(mgr).
//	    For(konfidencev1alpha1.ArtifactTypeSet{
//	        MyArtifactType: {},
//	    }).
//	    WithValidation(func(ctx context.Context, c client.Reader, target *konfidencev1alpha1.DeploymentTarget) error {
//	        // Validate target.Spec.Parameters and any referenced resources.
//	        return nil
//	    }).
//	    Watches(&mydeployer.DeployerCustomResource{}, deploymenttarget.EnqueueReferencedTargets()).
//	    Named("my-deployer:v1.0.0").
//	    Complete()
//
// A deployer must declare at least one supported artifact type with For. The controller then watches DeploymentTargets
// and reconciles only those targets whose artifact type it supports. A target is accepted when its parameters pass
// validation. Without a validation function, all parameters are accepted without additional checks. Deployers must
// validate their parameters before using them according to the deployer specification.
//
// Referenced resources should be read during validation and be part of the validation process. Use Watches to trigger
// DeploymentTarget reconciliation when those resources change. EnqueueReferencedTargets is a convenience handler that
// enqueues targets whose parameters reference the watched resource. Other event handlers may be used when appropriate.
package deploymenttarget
