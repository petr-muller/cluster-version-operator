package updatestatus

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	updatestatus "github.com/openshift/api/update/v1alpha1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	clocktesting "k8s.io/utils/clock/testing"

	fakeupdateclient "github.com/openshift/client-go/update/clientset/versioned/fake"
	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
)

func Test_updateStatusController(t *testing.T) {
	testCases := []struct {
		name string

		initialState  *updatestatus.UpdateStatus
		informerMsg   []informerMsg
		expectedState *updatestatus.UpdateStatus
	}{
		{
			name:          "no messages, no state -> no state",
			initialState:  nil,
			informerMsg:   []informerMsg{},
			expectedState: nil,
		},
		{
			name:          "no messages, empty state -> empty state",
			initialState:  &updatestatus.UpdateStatus{},
			expectedState: &updatestatus.UpdateStatus{},
		},
		{
			name:         "no messages, state -> unchanged state",
			initialState: &updatestatus.UpdateStatus{
				// TODO: FIXME
				// Data: map[string]string{
				// 	"usc.cpi.cv-version": "value",
				// },
			},
			expectedState: &updatestatus.UpdateStatus{
				// TODO: FIXME
				// Data: map[string]string{
				// 	"usc.cpi.cv-version": "value",
				// },
			},
		},
		{
			name:         "one message, no state -> initialize from message",
			initialState: nil,
			informerMsg: []informerMsg{
				{
					informer:  "cpi",
					uid:       "cv-version",
					cpInsight: &updatestatus.ControlPlaneInsight{UID: "cv-version"},
				},
			},
			expectedState: &updatestatus.UpdateStatus{
				// TODO: FIXME
				// Data: map[string]string{
				// 	"usc.cpi.cv-version": "cv-version from cpi",
				// },
			},
		},
		{
			name:         "messages over time build state over old state",
			initialState: &updatestatus.UpdateStatus{
				// TODO: FIXME
				// Data: map[string]string{
				// 	"usc.cpi.kept":        "kept",
				// 	"usc.cpi.overwritten": "old",
				// },
			},
			informerMsg: []informerMsg{
				{
					informer:      "cpi",
					uid:           "new-item",
					cpInsight:     &updatestatus.ControlPlaneInsight{UID: "new-item"},
					knownInsights: []string{"kept", "overwritten"},
				},
				{
					informer:      "cpi",
					uid:           "overwritten",
					cpInsight:     &updatestatus.ControlPlaneInsight{UID: "overwritten"},
					knownInsights: []string{"kept", "new-item"},
				},
				{
					informer:      "cpi",
					uid:           "another",
					cpInsight:     &updatestatus.ControlPlaneInsight{UID: "another"},
					knownInsights: []string{"kept", "overwritten", "new-item"},
				},
				{
					informer:      "cpi",
					uid:           "overwritten",
					cpInsight:     &updatestatus.ControlPlaneInsight{UID: "overwritten"},
					knownInsights: []string{"kept", "new-item", "another"},
				},
			},
			expectedState: &updatestatus.UpdateStatus{
				// TODO: FIXME
				// Data: map[string]string{
				// 	"usc.cpi.kept":        "kept",
				// 	"usc.cpi.new-item":    "new-item from cpi",
				// 	"usc.cpi.another":     "another from cpi",
				// 	"usc.cpi.overwritten": "overwritten from cpi",
				// },
			},
		},
		{
			name: "messages can come from different informers",
			informerMsg: []informerMsg{
				{
					informer:  "one",
					uid:       "item",
					cpInsight: &updatestatus.ControlPlaneInsight{UID: "item"},
				},
				{
					informer:  "two",
					uid:       "item",
					cpInsight: &updatestatus.ControlPlaneInsight{UID: "item"},
				},
				{
					informer:  "three",
					uid:       "item",
					wpInsight: &updatestatus.WorkerPoolInsight{UID: "item"},
				},
			},
			expectedState: &updatestatus.UpdateStatus{
				// TODO: FIXME
				// Data: map[string]string{
				// 	"usc.one.item":   "item from one",
				// 	"usc.two.item":   "item from two",
				// 	"usc.three.item": "item from three",
				// },
			},
		},
		{
			name:         "empty informer -> message gets dropped",
			initialState: nil,
			informerMsg: []informerMsg{
				{
					informer:  "",
					uid:       "item",
					cpInsight: &updatestatus.ControlPlaneInsight{UID: "item"},
				},
			},
			expectedState: nil,
		},
		{
			name:         "empty uid -> message gets dropped",
			initialState: nil,
			informerMsg: []informerMsg{
				{
					informer:  "one",
					uid:       "",
					cpInsight: &updatestatus.ControlPlaneInsight{UID: ""},
				},
			},
			expectedState: nil,
		},
		{
			name:         "nil insight payload -> message gets dropped",
			initialState: nil,
			informerMsg: []informerMsg{
				{
					informer:  "one",
					uid:       "item",
					cpInsight: nil,
					wpInsight: nil,
				},
			},
			expectedState: nil,
		},
		{
			name:         "both cp & wp insights payload -> message gets dropped",
			initialState: nil,
			informerMsg: []informerMsg{
				{
					informer:  "one",
					uid:       "item",
					cpInsight: &updatestatus.ControlPlaneInsight{UID: "item"},
					wpInsight: &updatestatus.WorkerPoolInsight{UID: "item"},
				},
			},
			expectedState: nil,
		},
		{
			name:         "unknown message gets removed from state",
			initialState: &updatestatus.UpdateStatus{
				// TODO: FIXME
				// Data: map[string]string{
				// 	"usc.one.old": "payload",
				// },
			},
			informerMsg: []informerMsg{{
				informer:      "one",
				uid:           "new",
				cpInsight:     &updatestatus.ControlPlaneInsight{UID: "new"},
				knownInsights: nil,
			}},
			expectedState: &updatestatus.UpdateStatus{
				// Data: map[string]string{
				// 	"usc.one.new": "new from one",
				// },
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			updateStatusClient := fakeupdateclient.NewClientset()

			controller := updateStatusController{
				updateStatuses: updateStatusClient.UpdateV1alpha1().UpdateStatuses(),
			}
			controller.statusApi.Lock()
			controller.statusApi.us = tc.initialState
			controller.statusApi.Unlock()

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
			var diff string
			backoff := wait.Backoff{Duration: 5 * time.Millisecond, Factor: 2, Steps: 10}
			if err := wait.ExponentialBackoff(backoff, func() (bool, error) {
				controller.statusApi.Lock()
				defer controller.statusApi.Unlock()

				sawProcessed = controller.statusApi.processed
				diff = cmp.Diff(tc.expectedState, controller.statusApi.us)

				return diff == "" && sawProcessed == expectedProcessed, nil
			}); err != nil {
				if diff != "" {
					t.Errorf("controller config map differs from expectedState:\n%s", diff)
				}
				if controller.statusApi.processed != len(tc.informerMsg) {
					t.Errorf("controller processed %d messages, expected %d", controller.statusApi.processed, len(tc.informerMsg))
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
