package main

import (
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/json"
	"ocm.software/ocm/api/ocm"
	"ocm.software/ocm/api/ocm/extensions/repositories/ocireg"
	"ocm.software/ocm/api/ocm/ocmutils/check"
	"ocm.software/ocm/api/tech/oci/identity"
	common "ocm.software/ocm/api/utils/misc"
)

func main() {
	ctx := ocm.DefaultContext()
	//credctx := ctx.CredentialsContext()
	creds := identity.SimpleCredentials("d060274", "<some-token>")

	//result, err := oci.GetConsumerIdForRef("konfidence.common.repositories.cloud.sap/ocm-test/ocm/vector")
	//if err != nil {
	//	panic(errors.Wrapf(err, "invalid consumer"))
	//}
	//credctx.SetCredentialsForConsumer(result, creds)

	spec := ocireg.NewRepositorySpec("konfidence.common.repositories.cloud.sap/ocm-test/ocm/vector")

	repo, err := ctx.RepositoryForSpec(spec, creds)
	if err != nil {
		panic(errors.Wrapf(err, "cannot setup repository"))
	}
	defer repo.Close()

	c, err := repo.LookupComponent("dwc.tools.sap/dwc-project/vector/dev-eu10")
	if err != nil {
		panic(errors.Wrapf(err, "cannot lookup component"))
	}
	defer c.Close()

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

	version, err := repo.LookupComponentVersion("dwc.tools.sap/dwc-project/vector/dev-eu10", "0.1.0")
	err = describeVersion(version)
	if err != nil {
		panic(errors.Wrapf(err, "cannot describe version"))
	}

	//
	//versions, err := c.ListVersions()
	//if err != nil {
	//	panic(errors.Wrapf(err, "cannot query version names"))
	//}
	//for _, i := range versions {
	//	println(i)
	//}
}

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
