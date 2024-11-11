// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"
	json "encoding/json"
	"fmt"

	v1 "github.com/openshift/api/config/v1"
	configv1 "github.com/openshift/client-go/config/applyconfigurations/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeInfrastructures implements InfrastructureInterface
type FakeInfrastructures struct {
	Fake *FakeConfigV1
}

var infrastructuresResource = v1.SchemeGroupVersion.WithResource("infrastructures")

var infrastructuresKind = v1.SchemeGroupVersion.WithKind("Infrastructure")

// Get takes name of the infrastructure, and returns the corresponding infrastructure object, and an error if there is any.
func (c *FakeInfrastructures) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Infrastructure, err error) {
	emptyResult := &v1.Infrastructure{}
	obj, err := c.Fake.
		Invokes(testing.NewRootGetActionWithOptions(infrastructuresResource, name, options), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Infrastructure), err
}

// List takes label and field selectors, and returns the list of Infrastructures that match those selectors.
func (c *FakeInfrastructures) List(ctx context.Context, opts metav1.ListOptions) (result *v1.InfrastructureList, err error) {
	emptyResult := &v1.InfrastructureList{}
	obj, err := c.Fake.
		Invokes(testing.NewRootListActionWithOptions(infrastructuresResource, infrastructuresKind, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.InfrastructureList{ListMeta: obj.(*v1.InfrastructureList).ListMeta}
	for _, item := range obj.(*v1.InfrastructureList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested infrastructures.
func (c *FakeInfrastructures) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchActionWithOptions(infrastructuresResource, opts))
}

// Create takes the representation of a infrastructure and creates it.  Returns the server's representation of the infrastructure, and an error, if there is any.
func (c *FakeInfrastructures) Create(ctx context.Context, infrastructure *v1.Infrastructure, opts metav1.CreateOptions) (result *v1.Infrastructure, err error) {
	emptyResult := &v1.Infrastructure{}
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateActionWithOptions(infrastructuresResource, infrastructure, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Infrastructure), err
}

// Update takes the representation of a infrastructure and updates it. Returns the server's representation of the infrastructure, and an error, if there is any.
func (c *FakeInfrastructures) Update(ctx context.Context, infrastructure *v1.Infrastructure, opts metav1.UpdateOptions) (result *v1.Infrastructure, err error) {
	emptyResult := &v1.Infrastructure{}
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateActionWithOptions(infrastructuresResource, infrastructure, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Infrastructure), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeInfrastructures) UpdateStatus(ctx context.Context, infrastructure *v1.Infrastructure, opts metav1.UpdateOptions) (result *v1.Infrastructure, err error) {
	emptyResult := &v1.Infrastructure{}
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceActionWithOptions(infrastructuresResource, "status", infrastructure, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Infrastructure), err
}

// Delete takes name of the infrastructure and deletes it. Returns an error if one occurs.
func (c *FakeInfrastructures) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(infrastructuresResource, name, opts), &v1.Infrastructure{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeInfrastructures) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	action := testing.NewRootDeleteCollectionActionWithOptions(infrastructuresResource, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v1.InfrastructureList{})
	return err
}

// Patch applies the patch and returns the patched infrastructure.
func (c *FakeInfrastructures) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Infrastructure, err error) {
	emptyResult := &v1.Infrastructure{}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceActionWithOptions(infrastructuresResource, name, pt, data, opts, subresources...), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Infrastructure), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied infrastructure.
func (c *FakeInfrastructures) Apply(ctx context.Context, infrastructure *configv1.InfrastructureApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Infrastructure, err error) {
	if infrastructure == nil {
		return nil, fmt.Errorf("infrastructure provided to Apply must not be nil")
	}
	data, err := json.Marshal(infrastructure)
	if err != nil {
		return nil, err
	}
	name := infrastructure.Name
	if name == nil {
		return nil, fmt.Errorf("infrastructure.Name must be provided to Apply")
	}
	emptyResult := &v1.Infrastructure{}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceActionWithOptions(infrastructuresResource, *name, types.ApplyPatchType, data, opts.ToPatchOptions()), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Infrastructure), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakeInfrastructures) ApplyStatus(ctx context.Context, infrastructure *configv1.InfrastructureApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Infrastructure, err error) {
	if infrastructure == nil {
		return nil, fmt.Errorf("infrastructure provided to Apply must not be nil")
	}
	data, err := json.Marshal(infrastructure)
	if err != nil {
		return nil, err
	}
	name := infrastructure.Name
	if name == nil {
		return nil, fmt.Errorf("infrastructure.Name must be provided to Apply")
	}
	emptyResult := &v1.Infrastructure{}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceActionWithOptions(infrastructuresResource, *name, types.ApplyPatchType, data, opts.ToPatchOptions(), "status"), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1.Infrastructure), err
}
