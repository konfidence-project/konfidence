package pipeline

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type (

	// Client wraps the controller-runtime client interface, to make it easier to mock in tests.
	Client interface {
		client.Client
	}

	// SubResourceWriter wraps the controller-runtime client.SubResourceWriter interface, to make it easier to mock in tests.
	SubResourceWriter interface {
		client.SubResourceWriter
	}

	// Object is a wrapper around the controller-runtime client.Object interface.
	Object interface {
		client.Object
	}
)
