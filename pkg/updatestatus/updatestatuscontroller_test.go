package updatestatus

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	fakeupdateclient "github.com/openshift/client-go/update/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	clocktesting "k8s.io/utils/clock/testing"

	updatestatus "github.com/openshift/api/update/v1alpha1"
	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
)

func Test_updateStatusController(t *testing.T) {
	var now = time.Now()
	var minus90sec = now.Add(-90 * time.Second)
	var minus30sec = now.Add(-30 * time.Second)
	var plus30sec = now.Add(30 * time.Second)
	var plus60min = now.Add(1 * time.Hour)

	cvInsight := updatestatus.ClusterVersionProgressInsightStatus{Name: "version"}
	coInsight := updatestatus.ClusterOperatorProgressInsightStatus{Name: "cluster-operator"}
	mcpInsight := updatestatus.MachineConfigPoolProgressInsightStatus{Name: "workers"}
	nodeInsight := updatestatus.NodeProgressInsightStatus{Name: "node"}
	healthInsight := updatestatus.HealthInsightStatus{}

	cvResourceRef := updatestatus.ResourceRef{
		Group:    "config.openshift.io",
		Resource: "clusterversions",
		Name:     "version",
	}

	testCases := []struct {
		name string

		before *updateStatusApi

		informerMsg []informerMsg

		expected *updateStatusApi
	}{
		{
			name:        "no messages, no state -> no state",
			before:      &updateStatusApi{},
			informerMsg: []informerMsg{},
			expected:    &updateStatusApi{},
		},
		{
			name: "no messages, empty state -> empty state",
			before: &updateStatusApi{
				informers: map[string]*informer{},
			},
			expected: &updateStatusApi{
				informers: map[string]*informer{},
			},
		},
		{
			name: "no messages, state -> unchanged state",
			before: &updateStatusApi{
				informers: map[string]*informer{
					"cpi": {
						name: "cpi",
						cvInsights: map[string]*updatestatus.ClusterVersionProgressInsightStatus{
							cvInsight.Name: &cvInsight,
						},
					},
				},
			},
			expected: &updateStatusApi{
				informers: map[string]*informer{
					"cpi": {
						name: "cpi",
						cvInsights: map[string]*updatestatus.ClusterVersionProgressInsightStatus{
							cvInsight.Name: &cvInsight,
						},
					},
				},
			},
		},
		{
			name: "one message, no state -> initialize from message",
			before: &updateStatusApi{
				informers: nil,
			},
			informerMsg: []informerMsg{
				{
					informer:  "cpi",
					uid:       cvInsight.Name,
					cvInsight: &cvInsight,
				},
			},
			expected: &updateStatusApi{
				informers: map[string]*informer{
					"cpi": {
						name: "cpi",
						cvInsights: map[string]*updatestatus.ClusterVersionProgressInsightStatus{
							cvInsight.Name: &cvInsight,
						},
					},
				},
			},
		},
		{
			name: "messages over time build state over old state",
			before: &updateStatusApi{
				informers: map[string]*informer{
					"cpi": {
						cvInsights: map[string]*updatestatus.ClusterVersionProgressInsightStatus{cvInsight.Name: &cvInsight},
						coInsights: map[string]*updatestatus.ClusterOperatorProgressInsightStatus{
							"overwritten": {
								Name: "overwritten",
								Conditions: []metav1.Condition{
									{
										Type:    string(updatestatus.ClusterOperatorProgressInsightUpdating),
										Status:  metav1.ConditionFalse,
										Reason:  "Original",
										Message: "Original message",
									},
								},
							},
						},
					},
				},
			},
			informerMsg: []informerMsg{
				{
					informer: "cpi",
					uid:      "new-clusteroperator",
					coInsight: &updatestatus.ClusterOperatorProgressInsightStatus{
						Name: "new-clusteroperator",
						Conditions: []metav1.Condition{
							{
								Type:    string(updatestatus.ClusterOperatorProgressInsightUpdating),
								Status:  metav1.ConditionTrue,
								Reason:  "NewClusterOperator",
								Message: "Message about new ClusterOperator",
							},
						},
					},
					knownInsights: []string{cvInsight.Name, "overwritten"},
				},
				{
					informer: "cpi",
					uid:      "overwritten",
					coInsight: &updatestatus.ClusterOperatorProgressInsightStatus{
						Name: "overwritten",
						Conditions: []metav1.Condition{
							{
								Type:    string(updatestatus.ClusterOperatorProgressInsightUpdating),
								Status:  metav1.ConditionUnknown,
								Reason:  "FirstWrite",
								Message: "First update into overwritten CO",
							},
						},
					},
					knownInsights: []string{cvInsight.Name, "new-clusteroperator"},
				},
				{
					informer: "cpi",
					uid:      "another-clusteroperator",
					coInsight: &updatestatus.ClusterOperatorProgressInsightStatus{
						Conditions: []metav1.Condition{
							{
								Type:    string(updatestatus.ClusterOperatorProgressInsightUpdating),
								Status:  metav1.ConditionTrue,
								Reason:  "AnotherClusterOperator",
								Message: "Message about another ClusterOperator",
							},
						},
						Name: "another-clusteroperator",
					},
					knownInsights: []string{cvInsight.Name, "new-clusteroperator", "overwritten"},
				},
				{
					informer: "cpi",
					uid:      "overwritten",
					coInsight: &updatestatus.ClusterOperatorProgressInsightStatus{
						Conditions: []metav1.Condition{
							{
								Type:    string(updatestatus.ClusterOperatorProgressInsightUpdating),
								Status:  metav1.ConditionTrue,
								Reason:  "FinalWrite",
								Message: "Final update into overwritten CO",
							},
						},
						Name: "overwritten",
					},
					knownInsights: []string{cvInsight.Name, "new-clusteroperator", "another-clusteroperator"},
				},
			},
			expected: &updateStatusApi{
				informers: map[string]*informer{
					"cpi": {
						cvInsights: map[string]*updatestatus.ClusterVersionProgressInsightStatus{cvInsight.Name: &cvInsight},
						coInsights: map[string]*updatestatus.ClusterOperatorProgressInsightStatus{
							"overwritten": {
								Conditions: []metav1.Condition{
									{
										Type:    string(updatestatus.ClusterOperatorProgressInsightUpdating),
										Status:  metav1.ConditionTrue,
										Reason:  "FinalWrite",
										Message: "Final update into overwritten CO",
									},
								},
								Name: "overwritten",
							},
							"new-clusteroperator": {
								Conditions: []metav1.Condition{
									{
										Type:    string(updatestatus.ClusterOperatorProgressInsightUpdating),
										Status:  metav1.ConditionTrue,
										Reason:  "NewClusterOperator",
										Message: "Message about new ClusterOperator",
									},
								},
								Name: "new-clusteroperator",
							},
							"another-clusteroperator": {
								Conditions: []metav1.Condition{
									{
										Type:    string(updatestatus.ClusterOperatorProgressInsightUpdating),
										Status:  metav1.ConditionTrue,
										Reason:  "AnotherClusterOperator",
										Message: "Message about another ClusterOperator",
									},
								},
								Name: "another-clusteroperator",
							},
						},
					},
				},
			},
		},
		{
			name:   "messages can come from different informers",
			before: &updateStatusApi{},
			informerMsg: []informerMsg{
				{
					informer: "one",
					uid:      "item",
					healthInsight: &updatestatus.HealthInsightStatus{
						StartedAt: metav1.NewTime(minus30sec),
						Scope: updatestatus.InsightScope{
							Type:      updatestatus.ControlPlaneScope,
							Resources: []updatestatus.ResourceRef{cvResourceRef},
						},
						Impact: updatestatus.InsightImpact{
							Level:       updatestatus.InfoImpactLevel,
							Type:        updatestatus.UnknownImpactType,
							Summary:     "Item from informer one",
							Description: "Longer description about item from informer one",
						},
						Remediation: updatestatus.InsightRemediation{Reference: "https://example.com"},
					},
				},
				{
					informer: "two",
					uid:      "item",
					healthInsight: &updatestatus.HealthInsightStatus{
						StartedAt: metav1.NewTime(minus90sec),
						Scope: updatestatus.InsightScope{
							Type:      updatestatus.ControlPlaneScope,
							Resources: []updatestatus.ResourceRef{cvResourceRef},
						},
						Impact: updatestatus.InsightImpact{
							Level:       updatestatus.InfoImpactLevel,
							Type:        updatestatus.UnknownImpactType,
							Summary:     "Item from informer two",
							Description: "Longer description about item from informer two",
						},
						Remediation: updatestatus.InsightRemediation{Reference: "https://example.com"},
					},
				},
				{
					informer: "three",
					uid:      "item",
					healthInsight: &updatestatus.HealthInsightStatus{
						StartedAt: metav1.NewTime(minus90sec),
						Scope: updatestatus.InsightScope{
							Type:      updatestatus.ControlPlaneScope,
							Resources: []updatestatus.ResourceRef{cvResourceRef},
						},
						Impact: updatestatus.InsightImpact{
							Level:       updatestatus.InfoImpactLevel,
							Type:        updatestatus.UnknownImpactType,
							Summary:     "Item from informer three",
							Description: "Longer description about item from informer three",
						},
						Remediation: updatestatus.InsightRemediation{Reference: "https://example.com"},
					},
				},
			},
			expected: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						healthInsights: map[string]*updatestatus.HealthInsightStatus{
							"item": {
								StartedAt: metav1.NewTime(minus30sec),
								Scope: updatestatus.InsightScope{
									Type:      updatestatus.ControlPlaneScope,
									Resources: []updatestatus.ResourceRef{cvResourceRef},
								},
								Impact: updatestatus.InsightImpact{
									Level:       updatestatus.InfoImpactLevel,
									Type:        updatestatus.UnknownImpactType,
									Summary:     "Item from informer one",
									Description: "Longer description about item from informer one",
								},
								Remediation: updatestatus.InsightRemediation{Reference: "https://example.com"},
							},
						},
					},
					"two": {
						name: "two",
						healthInsights: map[string]*updatestatus.HealthInsightStatus{
							"item": {
								StartedAt: metav1.NewTime(minus90sec),
								Scope: updatestatus.InsightScope{
									Type:      updatestatus.ControlPlaneScope,
									Resources: []updatestatus.ResourceRef{cvResourceRef},
								},
								Impact: updatestatus.InsightImpact{
									Level:       updatestatus.InfoImpactLevel,
									Type:        updatestatus.UnknownImpactType,
									Summary:     "Item from informer two",
									Description: "Longer description about item from informer two",
								},
								Remediation: updatestatus.InsightRemediation{Reference: "https://example.com"},
							},
						},
					},
					"three": {
						name: "three",
						healthInsights: map[string]*updatestatus.HealthInsightStatus{
							"item": {
								StartedAt: metav1.NewTime(minus90sec),
								Scope: updatestatus.InsightScope{
									Type:      updatestatus.ControlPlaneScope,
									Resources: []updatestatus.ResourceRef{cvResourceRef},
								},
								Impact: updatestatus.InsightImpact{
									Level:       updatestatus.InfoImpactLevel,
									Type:        updatestatus.UnknownImpactType,
									Summary:     "Item from informer three",
									Description: "Longer description about item from informer three",
								},
								Remediation: updatestatus.InsightRemediation{Reference: "https://example.com"},
							},
						},
					},
				},
			},
		},
		{
			name:   "empty informer -> message gets dropped",
			before: &updateStatusApi{},
			informerMsg: []informerMsg{
				{
					informer:  "",
					uid:       "item",
					cvInsight: &cvInsight,
				},
			},
			expected: &updateStatusApi{},
		},
		{
			name:   "empty uid -> message gets dropped",
			before: &updateStatusApi{},
			informerMsg: []informerMsg{
				{
					informer:  "one",
					uid:       "",
					cvInsight: &cvInsight,
				},
			},
			expected: &updateStatusApi{},
		},
		{
			name:   "nil insight payload -> message gets dropped",
			before: &updateStatusApi{},
			informerMsg: []informerMsg{
				{
					informer: "one",
					uid:      "item",
				},
			},
			expected: &updateStatusApi{},
		},
		{
			name:   "multiple insight payload -> message gets dropped",
			before: &updateStatusApi{},
			informerMsg: []informerMsg{
				{
					informer:  "one",
					uid:       "item",
					cvInsight: &cvInsight,
					coInsight: &coInsight,
				},
			},
			expected: &updateStatusApi{},
		},
		{
			name: "unknown insight -> not removed from state immediately but set for expiration",
			before: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						coInsights: map[string]*updatestatus.ClusterOperatorProgressInsightStatus{
							coInsight.Name: &coInsight,
						},
					},
				},
			},
			informerMsg: []informerMsg{{
				informer:      "one",
				uid:           cvInsight.Name,
				cvInsight:     &cvInsight,
				knownInsights: nil,
			}},
			expected: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						cvInsights: map[string]*updatestatus.ClusterVersionProgressInsightStatus{
							cvInsight.Name: &cvInsight,
						},
						coInsights: map[string]*updatestatus.ClusterOperatorProgressInsightStatus{
							coInsight.Name: &coInsight,
						},
					},
				},
				unknownInsightExpirations: map[string]insightExpirations{
					"one": {coInsight.Name: plus60min},
				},
			},
		},
		{
			name: "unknown insight already set for expiration -> not removed from state while not expired yet",
			before: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						coInsights: map[string]*updatestatus.ClusterOperatorProgressInsightStatus{
							coInsight.Name: &coInsight,
						},
					},
				},
				unknownInsightExpirations: map[string]insightExpirations{
					"one": {coInsight.Name: plus30sec},
				},
			},
			informerMsg: []informerMsg{{
				informer:      "one",
				uid:           cvInsight.Name,
				cvInsight:     &cvInsight,
				knownInsights: nil,
			}},
			expected: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						cvInsights: map[string]*updatestatus.ClusterVersionProgressInsightStatus{
							cvInsight.Name: &cvInsight,
						},
						coInsights: map[string]*updatestatus.ClusterOperatorProgressInsightStatus{
							coInsight.Name: &coInsight,
						},
					},
				},
				unknownInsightExpirations: map[string]insightExpirations{
					"one": {coInsight.Name: plus30sec},
				},
			},
		},
		{
			name: "previously unknown insight set for expiration is known again -> kept in state and expire dropped",
			before: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						coInsights: map[string]*updatestatus.ClusterOperatorProgressInsightStatus{
							coInsight.Name: &coInsight,
						},
					},
				},
				unknownInsightExpirations: map[string]insightExpirations{
					"one": {coInsight.Name: minus30sec},
				},
			},
			informerMsg: []informerMsg{{
				informer:      "one",
				uid:           cvInsight.Name,
				cvInsight:     &cvInsight,
				knownInsights: []string{coInsight.Name},
			}},
			expected: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						coInsights: map[string]*updatestatus.ClusterOperatorProgressInsightStatus{
							coInsight.Name: &coInsight,
						},
						cvInsights: map[string]*updatestatus.ClusterVersionProgressInsightStatus{
							cvInsight.Name: &cvInsight,
						},
					},
				},
				unknownInsightExpirations: nil,
			},
		},
		{
			name: "previously unknown CV insight expired and never became known again -> dropped from state and expire dropped",
			before: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						cvInsights: map[string]*updatestatus.ClusterVersionProgressInsightStatus{
							cvInsight.Name: &cvInsight,
						},
					},
				},
				unknownInsightExpirations: map[string]insightExpirations{
					"one": {cvInsight.Name: minus90sec},
				},
			},
			informerMsg: []informerMsg{{
				informer:      "one",
				uid:           coInsight.Name,
				coInsight:     &coInsight,
				knownInsights: nil,
			}},
			expected: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						coInsights: map[string]*updatestatus.ClusterOperatorProgressInsightStatus{
							coInsight.Name: &coInsight,
						},
					},
				},
				unknownInsightExpirations: nil,
			},
		},
		{
			name: "previously unknown CO insight expired and never became known again -> dropped from state and expire dropped",
			before: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						coInsights: map[string]*updatestatus.ClusterOperatorProgressInsightStatus{
							coInsight.Name: &coInsight,
						},
					},
				},
				unknownInsightExpirations: map[string]insightExpirations{
					"one": {coInsight.Name: minus90sec},
				},
			},
			informerMsg: []informerMsg{{
				informer:      "one",
				uid:           cvInsight.Name,
				cvInsight:     &cvInsight,
				knownInsights: nil,
			}},
			expected: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						cvInsights: map[string]*updatestatus.ClusterVersionProgressInsightStatus{
							cvInsight.Name: &cvInsight,
						},
					},
				},
				unknownInsightExpirations: nil,
			},
		},
		{
			name: "previously unknown MCP insight expired and never became known again -> dropped from state and expire dropped",
			before: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						mcpInsights: map[string]*updatestatus.MachineConfigPoolProgressInsightStatus{
							mcpInsight.Name: &mcpInsight,
						},
					},
				},
				unknownInsightExpirations: map[string]insightExpirations{
					"one": {mcpInsight.Name: minus90sec},
				},
			},
			informerMsg: []informerMsg{{
				informer:      "one",
				uid:           cvInsight.Name,
				cvInsight:     &cvInsight,
				knownInsights: nil,
			}},
			expected: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						cvInsights: map[string]*updatestatus.ClusterVersionProgressInsightStatus{
							cvInsight.Name: &cvInsight,
						},
					},
				},
				unknownInsightExpirations: nil,
			},
		},
		{
			name: "previously unknown Node insight expired and never became known again -> dropped from state and expire dropped",
			before: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						nodeInsights: map[string]*updatestatus.NodeProgressInsightStatus{
							nodeInsight.Name: &nodeInsight,
						},
					},
				},
				unknownInsightExpirations: map[string]insightExpirations{
					"one": {nodeInsight.Name: minus90sec},
				},
			},
			informerMsg: []informerMsg{{
				informer:      "one",
				uid:           cvInsight.Name,
				cvInsight:     &cvInsight,
				knownInsights: nil,
			}},
			expected: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						cvInsights: map[string]*updatestatus.ClusterVersionProgressInsightStatus{
							cvInsight.Name: &cvInsight,
						},
					},
				},
				unknownInsightExpirations: nil,
			},
		},
		{
			name: "previously unknown Health insight expired and never became known again -> dropped from state and expire dropped",
			before: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						healthInsights: map[string]*updatestatus.HealthInsightStatus{
							"health-uid": &healthInsight,
						},
					},
				},
				unknownInsightExpirations: map[string]insightExpirations{
					"one": {"health-uid": minus90sec},
				},
			},
			informerMsg: []informerMsg{{
				informer:      "one",
				uid:           cvInsight.Name,
				cvInsight:     &cvInsight,
				knownInsights: nil,
			}},
			expected: &updateStatusApi{
				informers: map[string]*informer{
					"one": {
						name: "one",
						cvInsights: map[string]*updatestatus.ClusterVersionProgressInsightStatus{
							cvInsight.Name: &cvInsight,
						},
					},
				},
				unknownInsightExpirations: nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			updateClient := fakeupdateclient.NewClientset()

			controller := updateStatusController{
				cvInsights:     updateClient.UpdateV1alpha1().ClusterVersionProgressInsights(),
				coInsights:     updateClient.UpdateV1alpha1().ClusterOperatorProgressInsights(),
				mcpInsights:    updateClient.UpdateV1alpha1().MachineConfigPoolProgressInsights(),
				nodeInsights:   updateClient.UpdateV1alpha1().NodeProgressInsights(),
				healthInsights: updateClient.UpdateV1alpha1().HealthInsights(),

				state: updateStatusApi{
					informers:                 tc.before.informers,
					unknownInsightExpirations: tc.before.unknownInsightExpirations,
					now:                       func() time.Time { return now },
				},
			}

			startInsightReceiver, sendInsight := controller.setupInsightReceiver()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			go func() {
				_ = startInsightReceiver(ctx, newTestSyncContextWithQueue())
			}()

			for _, msg := range tc.informerMsg {
				sendInsight(msg)
			}

			expectedProcessed := len(tc.informerMsg)
			var sawProcessed int
			var diffInformers string
			var diffExpirations string

			backoff := wait.Backoff{Duration: 5 * time.Millisecond, Factor: 2, Steps: 10}
			if err := wait.ExponentialBackoff(backoff, func() (bool, error) {
				controller.state.Lock()
				defer controller.state.Unlock()

				sawProcessed = controller.state.processed
				diffInformers = cmp.Diff(tc.expected.informers, controller.state.informers, cmp.AllowUnexported(informer{}))
				diffExpirations = cmp.Diff(tc.expected.unknownInsightExpirations, controller.state.unknownInsightExpirations)

				return diffInformers == "" && diffExpirations == "" && sawProcessed == expectedProcessed, nil
			}); err != nil {
				if diffInformers != "" {
					t.Errorf("controller state differs from expected:\n%s", diffInformers)
				}
				if diffExpirations != "" {
					t.Errorf("expirations differ from expected:\n%s", diffExpirations)
				}
				if controller.state.processed != len(tc.informerMsg) {
					t.Errorf("controller processed %d messages, expected %d", controller.state.processed, len(tc.informerMsg))
				}
			}
		})
	}
}

func newTestSyncContextWithQueue() factory.SyncContext {
	return testSyncContext{
		eventRecorder: events.NewInMemoryRecorder("test", clocktesting.NewFakePassiveClock(time.Now())),
		queue:         workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[any]()),
	}
}
