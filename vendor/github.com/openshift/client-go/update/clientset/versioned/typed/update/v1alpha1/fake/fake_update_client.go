// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/openshift/client-go/update/clientset/versioned/typed/update/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeUpdateV1alpha1 struct {
	*testing.Fake
}

func (c *FakeUpdateV1alpha1) UpdateStatuses() v1alpha1.UpdateStatusInterface {
	return newFakeUpdateStatuses(c)
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeUpdateV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
