// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

// ControlPlaneInformerApplyConfiguration represents a declarative configuration of the ControlPlaneInformer type for use
// with apply.
type ControlPlaneInformerApplyConfiguration struct {
	Name     *string                                 `json:"name,omitempty"`
	Insights []ControlPlaneInsightApplyConfiguration `json:"insights,omitempty"`
}

// ControlPlaneInformerApplyConfiguration constructs a declarative configuration of the ControlPlaneInformer type for use with
// apply.
func ControlPlaneInformer() *ControlPlaneInformerApplyConfiguration {
	return &ControlPlaneInformerApplyConfiguration{}
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *ControlPlaneInformerApplyConfiguration) WithName(value string) *ControlPlaneInformerApplyConfiguration {
	b.Name = &value
	return b
}

// WithInsights adds the given value to the Insights field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Insights field.
func (b *ControlPlaneInformerApplyConfiguration) WithInsights(values ...*ControlPlaneInsightApplyConfiguration) *ControlPlaneInformerApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithInsights")
		}
		b.Insights = append(b.Insights, *values[i])
	}
	return b
}
