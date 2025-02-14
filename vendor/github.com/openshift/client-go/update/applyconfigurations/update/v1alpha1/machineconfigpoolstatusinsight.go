// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	updatev1alpha1 "github.com/openshift/api/update/v1alpha1"
	v1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

// MachineConfigPoolStatusInsightApplyConfiguration represents a declarative configuration of the MachineConfigPoolStatusInsight type for use
// with apply.
type MachineConfigPoolStatusInsightApplyConfiguration struct {
	Conditions []v1.ConditionApplyConfiguration   `json:"conditions,omitempty"`
	Name       *string                            `json:"name,omitempty"`
	Resource   *PoolResourceRefApplyConfiguration `json:"resource,omitempty"`
	Scope      *updatev1alpha1.ScopeType          `json:"scopeType,omitempty"`
	Assessment *updatev1alpha1.PoolAssessment     `json:"assessment,omitempty"`
	Completion *int32                             `json:"completion,omitempty"`
	Summaries  []NodeSummaryApplyConfiguration    `json:"summaries,omitempty"`
}

// MachineConfigPoolStatusInsightApplyConfiguration constructs a declarative configuration of the MachineConfigPoolStatusInsight type for use with
// apply.
func MachineConfigPoolStatusInsight() *MachineConfigPoolStatusInsightApplyConfiguration {
	return &MachineConfigPoolStatusInsightApplyConfiguration{}
}

// WithConditions adds the given value to the Conditions field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Conditions field.
func (b *MachineConfigPoolStatusInsightApplyConfiguration) WithConditions(values ...*v1.ConditionApplyConfiguration) *MachineConfigPoolStatusInsightApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithConditions")
		}
		b.Conditions = append(b.Conditions, *values[i])
	}
	return b
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *MachineConfigPoolStatusInsightApplyConfiguration) WithName(value string) *MachineConfigPoolStatusInsightApplyConfiguration {
	b.Name = &value
	return b
}

// WithResource sets the Resource field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Resource field is set to the value of the last call.
func (b *MachineConfigPoolStatusInsightApplyConfiguration) WithResource(value *PoolResourceRefApplyConfiguration) *MachineConfigPoolStatusInsightApplyConfiguration {
	b.Resource = value
	return b
}

// WithScope sets the Scope field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Scope field is set to the value of the last call.
func (b *MachineConfigPoolStatusInsightApplyConfiguration) WithScope(value updatev1alpha1.ScopeType) *MachineConfigPoolStatusInsightApplyConfiguration {
	b.Scope = &value
	return b
}

// WithAssessment sets the Assessment field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Assessment field is set to the value of the last call.
func (b *MachineConfigPoolStatusInsightApplyConfiguration) WithAssessment(value updatev1alpha1.PoolAssessment) *MachineConfigPoolStatusInsightApplyConfiguration {
	b.Assessment = &value
	return b
}

// WithCompletion sets the Completion field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Completion field is set to the value of the last call.
func (b *MachineConfigPoolStatusInsightApplyConfiguration) WithCompletion(value int32) *MachineConfigPoolStatusInsightApplyConfiguration {
	b.Completion = &value
	return b
}

// WithSummaries adds the given value to the Summaries field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Summaries field.
func (b *MachineConfigPoolStatusInsightApplyConfiguration) WithSummaries(values ...*NodeSummaryApplyConfiguration) *MachineConfigPoolStatusInsightApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithSummaries")
		}
		b.Summaries = append(b.Summaries, *values[i])
	}
	return b
}
