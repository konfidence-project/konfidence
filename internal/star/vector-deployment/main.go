package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/json"
	"ocm.software/ocm/api/oci"
	"ocm.software/ocm/api/ocm"
	"ocm.software/ocm/api/ocm/extensions/repositories/ocireg"
	"ocm.software/ocm/api/ocm/ocmutils/check"
	"ocm.software/ocm/api/tech/oci/identity"
	common "ocm.software/ocm/api/utils/misc"
)

func main() {
	ctx := ocm.DefaultContext()
	credctx := ctx.CredentialsContext()

	creds := identity.SimpleCredentials("d060274", os.Getenv("repo_token"))

	consumerId, err := oci.GetConsumerIdForRef("konfidence.common.repositories.cloud.sap")
	if err != nil {
		panic(errors.Wrapf(err, "invalid consumer"))
	}
	credctx.SetCredentialsForConsumer(consumerId, creds)

	spec := ocireg.NewRepositorySpec("konfidence.common.repositories.cloud.sap/ocm-test/ocm/vector")

	consumerId, err = oci.GetConsumerIdForRef("foo.bar")
	if err != nil {
		panic(errors.Wrapf(err, "invalid consumer"))
	}
	foo, err := credctx.GetCredentialsForConsumer(consumerId)
	if err != nil {
		panic(errors.Wrapf(err, "cannot get consumer"))
	}
	println(foo)

	repo, err := ctx.RepositoryForSpec(spec, creds)
	if err != nil {
		panic(errors.Wrapf(err, "cannot setup repository"))
	}
	defer repo.Close() //nolint:errcheck // TODO: fix errcheck - properly handle error return value

	c, err := repo.LookupComponent("dwc.tools.sap/dwc-project/vector/dev-eu10")
	if err != nil {
		panic(errors.Wrapf(err, "cannot lookup component"))
	}
	defer c.Close() //nolint:errcheck // TODO: fix errcheck - properly handle error return value

	options := check.Check()
	result, err := options.ForId(repo, common.NewNameVersion("dwc.tools.sap/dwc-project/vector/dev-eu10", "0.1.0"))
	if err != nil {
		panic(errors.Wrapf(err, "check failed"))
	}

	marshal, err := json.Marshal(result)
	if err != nil {
		panic(errors.Wrapf(err, "marshal failed"))
	}
	println(string(marshal))

	// TODO: fix ineffassign - handle or remove unused assignment
	//nolint:ineffassign,staticcheck
	version, err := repo.LookupComponentVersion("dwc.tools.sap/dwc-project/service1", "0.0.1")
	err = describeVersion(version)
	if err != nil {
		panic(errors.Wrapf(err, "cannot describe version"))
	}

	for _, access := range version.GetResources() {
		if access.Meta().Type != "cloud.konfidence.artifact.manifest" {
			continue
		}
		method, err := access.AccessMethod()
		if err != nil {
			panic(errors.Wrapf(err, "cannot get access method for resource %s", access.Meta().GetName()))
		}
		get, err := method.Get()
		if err != nil {
			panic(errors.Wrapf(err, "cannot get content for resource %s", access.Meta().GetName()))
		}
		fmt.Println(string(get))
	}
}

//nolint:unparam // TODO: fix unparam - either make function return meaningful errors or change return type
func describeVersion(cv ocm.ComponentVersionAccess) error {
	// many elements of the API keep trak of their context
	ctx := cv.GetContext()

	references := cv.GetReferences()
	for _, ref := range references {
		fmt.Printf("reference: %s\n", ref.GetName())
		fmt.Printf("  component: %s\n", ref.GetComponentName())
		fmt.Printf("  version: %s\n", ref.GetVersion())
	}

	// Have a look at the component descriptor
	cd := cv.GetDescriptor()
	fmt.Printf("resources of the latest version of %s:\n", cv.GetName())
	fmt.Printf("  version:  %s\n", cv.GetVersion())
	fmt.Printf("  provider: %s\n", cd.Provider.Name)

	// and list all the included resources.
	for i, r := range cv.GetResources() {
		fmt.Printf("  %2d: name:           %s\n", i+1, r.Meta().GetName())
		fmt.Printf("      extra identity: %s\n", r.Meta().GetExtraIdentity())
		fmt.Printf("      resource type:  %s\n", r.Meta().GetType())
		acc, err := r.Access()
		if err != nil {
			fmt.Printf("      access:         error: %s\n", err)
		} else {
			fmt.Printf("      access:         %s\n", acc.Describe(ctx))
		}
	}
	return nil
}
