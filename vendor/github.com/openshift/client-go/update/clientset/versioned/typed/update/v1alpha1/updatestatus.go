// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	context "context"

	updatev1alpha1 "github.com/openshift/api/update/v1alpha1"
	applyconfigurationsupdatev1alpha1 "github.com/openshift/client-go/update/applyconfigurations/update/v1alpha1"
	scheme "github.com/openshift/client-go/update/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// UpdateStatusesGetter has a method to return a UpdateStatusInterface.
// A group's client should implement this interface.
type UpdateStatusesGetter interface {
	UpdateStatuses() UpdateStatusInterface
}

// UpdateStatusInterface has methods to work with UpdateStatus resources.
type UpdateStatusInterface interface {
	Create(ctx context.Context, updateStatus *updatev1alpha1.UpdateStatus, opts v1.CreateOptions) (*updatev1alpha1.UpdateStatus, error)
	Update(ctx context.Context, updateStatus *updatev1alpha1.UpdateStatus, opts v1.UpdateOptions) (*updatev1alpha1.UpdateStatus, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, updateStatus *updatev1alpha1.UpdateStatus, opts v1.UpdateOptions) (*updatev1alpha1.UpdateStatus, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*updatev1alpha1.UpdateStatus, error)
	List(ctx context.Context, opts v1.ListOptions) (*updatev1alpha1.UpdateStatusList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *updatev1alpha1.UpdateStatus, err error)
	Apply(ctx context.Context, updateStatus *applyconfigurationsupdatev1alpha1.UpdateStatusApplyConfiguration, opts v1.ApplyOptions) (result *updatev1alpha1.UpdateStatus, err error)
	// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
	ApplyStatus(ctx context.Context, updateStatus *applyconfigurationsupdatev1alpha1.UpdateStatusApplyConfiguration, opts v1.ApplyOptions) (result *updatev1alpha1.UpdateStatus, err error)
	UpdateStatusExpansion
}

// updateStatuses implements UpdateStatusInterface
type updateStatuses struct {
	*gentype.ClientWithListAndApply[*updatev1alpha1.UpdateStatus, *updatev1alpha1.UpdateStatusList, *applyconfigurationsupdatev1alpha1.UpdateStatusApplyConfiguration]
}

// newUpdateStatuses returns a UpdateStatuses
func newUpdateStatuses(c *UpdateV1alpha1Client) *updateStatuses {
	return &updateStatuses{
		gentype.NewClientWithListAndApply[*updatev1alpha1.UpdateStatus, *updatev1alpha1.UpdateStatusList, *applyconfigurationsupdatev1alpha1.UpdateStatusApplyConfiguration](
			"updatestatuses",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *updatev1alpha1.UpdateStatus { return &updatev1alpha1.UpdateStatus{} },
			func() *updatev1alpha1.UpdateStatusList { return &updatev1alpha1.UpdateStatusList{} },
		),
	}
}
