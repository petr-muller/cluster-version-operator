package updatestatus

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"

	updatestatus "github.com/openshift/api/update/v1alpha1"
	updateclient "github.com/openshift/client-go/update/clientset/versioned"
	updatev1alpha1 "github.com/openshift/client-go/update/clientset/versioned/typed/update/v1alpha1"
)

const (
	unknownInsightGracePeriod = 60 * time.Minute
)

// High-level description of the informers -> USC communication protocol:
// ----------------------------------------------------------------------
// Informers send insights to the USC via messages. Communication is performed via a channel (but that is just the
// current implementation detail) and the data sent by informers is encapsulated in the informerMsg structure. The
// communication is unidirectional, from informers to the USC. The USC does not send any messages back to the informers.
//
// The informers send individual insights they want to propagate to the Status API, insights are identified by a UID.
// Insights with the same UID are considered the same insight in the context of the informer that sent it. The received
// insights are stored in the Status API by the USC if they are new, and updated with the new data if they are already
// present.
//
// Informers keep track of active insights, and include a list of all known insights (just the UIDs) in each message.
// On each message, USC compares the insights by the informer it has in the Status API with the list of known insights
// in the message, and when an insight is first not reported as known by the informer, it is marked for expiration. If
// the informer reports the insight as known again before it expires, the expiration is cancelled. If the insight is not
// reported as known again within a grace period, it is dropped from the Status API. This allows informers to restart
// and "learn" about conditions in the cluster without dropping insights that it have not yet learned about while
// still eventually dropping insights that are no longer detected.
//
// Informers can also report insights they want to explicitly drop. This works similarly to the expiration mechanism,
// but there is no grace period.
//
// TL;DR:
// --------
// Whenever an informer has an insight to report, it sends a message containing:
// - The informer's name
// - The insight itself, identified by a UID
// - The list of all insights it knows about (just the UIDs)
// - The list of all insights it wants to explicitly drop (just the UIDs)
//
// For each message received, the USC:
// - Updates the Status API with the insight
// - Marks insights by the informer already in Status API for expiration if informer does not know them
// - Drops insights marked for expiration it grace period is over and informer does not still know them
// - Unmarks the expiration for each insight the informer knows
// - Drops insights explicitly requested by the informer
//
// Implementation status:
// ---------------------
// - [x] USC-side known insight tracking
// - [x] USC-side insight expiration
// - [ ] Informer-side known insight tracking
// - [ ] Informer-side populating known insights in messages
// - [ ] USC-side insight explicit dropping
// - [ ] Informer-side explicit insight drop tracking
// - [ ] Informer-side populating explicit drop insights in messages

// informerMsg is the communication structure between informers and the update status controller. It contains the UID of
// the insight and the insight itself, serialized as YAML. Passing serialized avoids shared data access problems. Until
// we have the Status API we need to serialize ourselves anyway.
type informerMsg struct {
	informer string
	// knownInsights contains the UIDs of insights known by the informer, so the controller can remove insights formerly
	// reported by the informer but no longer known to it (e.g. because the informer was restarted and the culprit
	// condition ceased to exist in the meantime). The `uid` of the insight in the message payload is always assumed
	// to be known, and is not required to be included in `knownInsights` by the informers (but informers can do so).
	knownInsights []string

	uid string

	cvInsight     *updatestatus.ClusterVersionProgressInsightStatus
	coInsight     *updatestatus.ClusterOperatorProgressInsightStatus
	mcpInsight    *updatestatus.MachineConfigPoolProgressInsightStatus
	nodeInsight   *updatestatus.NodeProgressInsightStatus
	healthInsight *updatestatus.HealthInsightStatus
}

func makeClusterVersionProgressInsightMsg(insight *updatestatus.ClusterVersionProgressInsightStatus, uid, informer string) (informerMsg, error) {
	msg := informerMsg{
		informer:  informer,
		uid:       uid,
		cvInsight: insight.DeepCopy(),
	}
	return msg, msg.validate()
}

func makeClusterOperatorProgressInsightMsg(insight *updatestatus.ClusterOperatorProgressInsightStatus, uid, informer string) (informerMsg, error) {
	msg := informerMsg{
		informer:  informer,
		uid:       uid,
		coInsight: insight.DeepCopy(),
	}
	return msg, msg.validate()
}

func makeMachineConfigPoolProgressInsightMsg(insight *updatestatus.MachineConfigPoolProgressInsightStatus, uid, informer string) (informerMsg, error) {
	msg := informerMsg{
		informer:   informer,
		uid:        uid,
		mcpInsight: insight.DeepCopy(),
	}
	return msg, msg.validate()
}

func makeNodeProgressInsightMsg(insight *updatestatus.NodeProgressInsightStatus, uid, informer string) (informerMsg, error) {
	msg := informerMsg{
		informer:    informer,
		uid:         uid,
		nodeInsight: insight.DeepCopy(),
	}
	return msg, msg.validate()
}

func makeHealthInsightMsg(insight *updatestatus.HealthInsightStatus, uid, informer string) (informerMsg, error) {
	msg := informerMsg{
		informer:      informer,
		uid:           uid,
		healthInsight: insight.DeepCopy(),
	}
	return msg, msg.validate()
}

type sendInsightFn func(insight informerMsg)

// updateStatusController is a controller that collects insights from informers and maintains the UpdateStatus API.
// The controller maintains an internal desired content of the UpdateStatus instance (even if it does not exist in the
// cluster) and updates it in the cluster when new insights are received, or when the UpdateStatus changes
// in the cluster. The controller only maintains the UpdateStatus in the cluster if it exists, it does not create it
// itself (this serves as a simple opt-in mechanism).
//
// The communication between informers (insight producers) and this controller is performed via a channel. The controller
// constructor returns a sendInsightFn function to be used by other controllers to send insights to this controller. The
// informerMsg structure is the data transfer object.
//
// updateStatusController is set up to spawn the insight receiver after it is started. The receiver reads messages from
// the channel, updates the internal state of the controller, and queues the UpdateStatus to be updated in the cluster.
// The sendInsightFn function can be used to send insights to the controller even before the insight receiver starts,
// but the buffered channel has limited capacity so senders can block eventually.
//
// NOTE: The communication mechanism was added in the initial scaffolding PR and does not aspire to be the final
// and 100% efficient solution. Feel free to improve or even replace it if turns out to be unsuitable in practice.
type updateStatusController struct {
	cvInsights     updatev1alpha1.ClusterVersionProgressInsightInterface
	coInsights     updatev1alpha1.ClusterOperatorProgressInsightInterface
	mcpInsights    updatev1alpha1.MachineConfigPoolProgressInsightInterface
	nodeInsights   updatev1alpha1.NodeProgressInsightInterface
	healthInsights updatev1alpha1.HealthInsightInterface

	state updateStatusApi

	recorder events.Recorder
}

// newUpdateStatusController creates a new update status controller and returns it. The second return value is a function
// the other controllers should use to send insights to this controller.
func newUpdateStatusController(
	updateClient updateclient.Interface,
	recorder events.Recorder,
) (factory.Controller, sendInsightFn) {
	uscRecorder := recorder.WithComponentSuffix("update-status-controller")

	c := &updateStatusController{
		cvInsights:     updateClient.UpdateV1alpha1().ClusterVersionProgressInsights(),
		coInsights:     updateClient.UpdateV1alpha1().ClusterOperatorProgressInsights(),
		mcpInsights:    updateClient.UpdateV1alpha1().MachineConfigPoolProgressInsights(),
		nodeInsights:   updateClient.UpdateV1alpha1().NodeProgressInsights(),
		healthInsights: updateClient.UpdateV1alpha1().HealthInsights(),

		recorder: uscRecorder,
		state:    updateStatusApi{now: time.Now},
	}

	startInsightReceiver, sendInsight := c.setupInsightReceiver()

	controller := factory.New().
		WithSync(c.sync).ResyncEvery(time.Minute).
		WithPostStartHooks(startInsightReceiver).
		ToController("UpdateStatusController", c.recorder)

	return controller, sendInsight
}

func (m informerMsg) validate() error {
	switch {
	case m.informer == "":
		return fmt.Errorf("empty informer")
	case m.uid == "":
		return fmt.Errorf("empty UID")
	case m.cvInsight == nil && m.coInsight == nil && m.mcpInsight == nil && m.nodeInsight == nil && m.healthInsight == nil:
		return fmt.Errorf("empty insight")

	// Stupid but works for now
	case m.cvInsight != nil && (m.coInsight != nil || m.mcpInsight != nil || m.nodeInsight != nil || m.healthInsight != nil):
		return fmt.Errorf("multiple insights in a single message")
	case m.coInsight != nil && (m.cvInsight != nil || m.mcpInsight != nil || m.nodeInsight != nil || m.healthInsight != nil):
		return fmt.Errorf("multiple insights in a single message")
	case m.mcpInsight != nil && (m.cvInsight != nil || m.coInsight != nil || m.nodeInsight != nil || m.healthInsight != nil):
		return fmt.Errorf("multiple insights in a single message")
	case m.nodeInsight != nil && (m.cvInsight != nil || m.coInsight != nil || m.mcpInsight != nil || m.healthInsight != nil):
		return fmt.Errorf("multiple insights in a single message")
	case m.healthInsight != nil && (m.cvInsight != nil || m.coInsight != nil || m.mcpInsight != nil || m.nodeInsight != nil):
		return fmt.Errorf("multiple insights in a single message")
	}

	return nil
}

// setupInsightReceiver creates a communication channel between informers and the update status controller, and returns
// two methods: one to start the insight receiver (to be used as a post start hook so it called after the controller is
// started), and one to be passed to informers to send insights to the controller.
func (c *updateStatusController) setupInsightReceiver() (factory.PostStartHook, sendInsightFn) {
	fromInformers := make(chan informerMsg, 100)

	startInsightReceiver := func(ctx context.Context, syncCtx factory.SyncContext) error {
		klog.V(2).Info("USC :: Collector :: Starting insight collector")
		for {
			select {
			case message := <-fromInformers:
				for item := range c.state.processInsightMsg(message) {
					syncCtx.Queue().Add(item)
				}
			case <-ctx.Done():
				klog.Info("USC :: Collector :: Stopping insight collector")
				return nil
			}
		}
	}

	sendInsight := func(msg informerMsg) {
		fromInformers <- msg
	}

	return startInsightReceiver, sendInsight
}

func (c *updateStatusController) sync(ctx context.Context, syncCtx factory.SyncContext) error {
	key := syncCtx.QueueKey()
	items := strings.Split(key, "/")
	if len(items) != 3 {
		return fmt.Errorf("unexpected queue key: %s", key)
	}

	informer := items[0]
	uid := items[2]

	switch items[1] {
	case "cv":
		return c.syncClusterVersionProgressInsight(ctx, informer, uid)
	case "co":
		return c.syncClusterOperatorProgressInsight(ctx, informer, uid)
	case "mcp":
		return c.syncMachineConfigPoolProgressInsight(ctx, informer, uid)
	case "node":
		return c.syncNodeProgressInsight(ctx, informer, uid)
	case "health":
		return c.syncHealthInsight(ctx, informer, uid)
	}

	return nil
}

func (c *updateStatusController) syncClusterVersionProgressInsight(ctx context.Context, informer, uid string) error {
	insightStatus := c.state.getClusterVersionProgressInsight(informer, uid)
	name := fmt.Sprintf("%s-%s", informer, uid)

	current, err := c.cvInsights.Get(ctx, name, v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error getting cluster version insight: %w", err)
	}

	if insightStatus == nil {
		return c.deleteClusterVersionProgressInsight(ctx, current, err != nil && errors.IsNotFound(err))
	}

	return c.upsertClusterVersionProgressInsight(ctx, current, insightStatus, informer, uid)
}

func (c *updateStatusController) deleteClusterVersionProgressInsight(ctx context.Context, current *updatestatus.ClusterVersionProgressInsight, notFound bool) error {
	if notFound || current == nil || current.DeletionTimestamp != nil {
		return nil
	}

	err := c.cvInsights.Delete(ctx, current.Name, v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error deleting cluster version insight: %w", err)
	}
	return nil
}

func (c *updateStatusController) upsertClusterVersionProgressInsight(ctx context.Context, current *updatestatus.ClusterVersionProgressInsight, insightStatus *updatestatus.ClusterVersionProgressInsightStatus, informer, uid string) error {
	if current == nil {
		return c.createClusterVersionProgressInsight(ctx, insightStatus, informer, uid)
	}
	return c.updateClusterVersionProgressInsight(ctx, current, insightStatus)
}

func (c *updateStatusController) createClusterVersionProgressInsight(ctx context.Context, insightStatus *updatestatus.ClusterVersionProgressInsightStatus, informer, uid string) error {
	insight := &updatestatus.ClusterVersionProgressInsight{
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", informer, uid),
		},
		Status: *insightStatus.DeepCopy(),
	}

	_, err := c.cvInsights.Create(ctx, insight, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating cluster version insight: %w", err)
	}

	return nil
}

func (c *updateStatusController) updateClusterVersionProgressInsight(ctx context.Context, current *updatestatus.ClusterVersionProgressInsight, insightStatus *updatestatus.ClusterVersionProgressInsightStatus) error {
	insight := current.DeepCopy()
	insight.Status = *insightStatus.DeepCopy()

	_, err := c.cvInsights.UpdateStatus(ctx, insight, v1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating cluster version insight: %w", err)
	}

	return nil
}

func (c *updateStatusController) syncClusterOperatorProgressInsight(ctx context.Context, informer, uid string) error {
	insightStatus := c.state.getClusterOperatorProgressInsight(informer, uid)
	name := fmt.Sprintf("%s-%s", informer, uid)

	current, err := c.coInsights.Get(ctx, name, v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error getting cluster operator insight: %w", err)
	}

	if insightStatus == nil {
		return c.deleteClusterOperatorProgressInsight(ctx, current, err != nil && errors.IsNotFound(err))
	}

	return c.upsertClusterOperatorProgressInsight(ctx, current, insightStatus, informer, uid)
}

func (c *updateStatusController) deleteClusterOperatorProgressInsight(ctx context.Context, current *updatestatus.ClusterOperatorProgressInsight, notFound bool) error {
	if notFound || current == nil || current.DeletionTimestamp != nil {
		return nil
	}

	err := c.coInsights.Delete(ctx, current.Name, v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error deleting cluster operator insight: %w", err)
	}
	return nil
}

func (c *updateStatusController) upsertClusterOperatorProgressInsight(ctx context.Context, current *updatestatus.ClusterOperatorProgressInsight, insightStatus *updatestatus.ClusterOperatorProgressInsightStatus, informer, uid string) error {
	if current == nil {
		return c.createClusterOperatorProgressInsight(ctx, insightStatus, informer, uid)
	}
	return c.updateClusterOperatorProgressInsight(ctx, current, insightStatus)
}

func (c *updateStatusController) createClusterOperatorProgressInsight(ctx context.Context, insightStatus *updatestatus.ClusterOperatorProgressInsightStatus, informer, uid string) error {
	insight := &updatestatus.ClusterOperatorProgressInsight{
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", informer, uid),
		},
		Status: *insightStatus.DeepCopy(),
	}

	_, err := c.coInsights.Create(ctx, insight, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating cluster operator insight: %w", err)
	}

	return nil
}

func (c *updateStatusController) updateClusterOperatorProgressInsight(ctx context.Context, current *updatestatus.ClusterOperatorProgressInsight, insightStatus *updatestatus.ClusterOperatorProgressInsightStatus) error {
	insight := current.DeepCopy()
	insight.Status = *insightStatus.DeepCopy()

	_, err := c.coInsights.UpdateStatus(ctx, insight, v1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating cluster operator insight: %w", err)
	}

	return nil
}

func (c *updateStatusController) syncMachineConfigPoolProgressInsight(ctx context.Context, informer, uid string) error {
	insightStatus := c.state.getMachineConfigPoolProgressInsight(informer, uid)
	name := fmt.Sprintf("%s-%s", informer, uid)

	current, err := c.mcpInsights.Get(ctx, name, v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error getting machine config pool insight: %w", err)
	}

	if insightStatus == nil {
		return c.deleteMachineConfigPoolProgressInsight(ctx, current, err != nil && errors.IsNotFound(err))
	}

	return c.upsertMachineConfigPoolProgressInsight(ctx, current, insightStatus, informer, uid)
}

func (c *updateStatusController) deleteMachineConfigPoolProgressInsight(ctx context.Context, current *updatestatus.MachineConfigPoolProgressInsight, notFound bool) error {
	if notFound || current == nil || current.DeletionTimestamp != nil {
		return nil
	}

	err := c.mcpInsights.Delete(ctx, current.Name, v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error deleting machine config pool insight: %w", err)
	}
	return nil
}

func (c *updateStatusController) upsertMachineConfigPoolProgressInsight(ctx context.Context, current *updatestatus.MachineConfigPoolProgressInsight, insightStatus *updatestatus.MachineConfigPoolProgressInsightStatus, informer, uid string) error {
	if current == nil {
		return c.createMachineConfigPoolProgressInsight(ctx, insightStatus, informer, uid)
	}
	return c.updateMachineConfigPoolProgressInsight(ctx, current, insightStatus)
}

func (c *updateStatusController) createMachineConfigPoolProgressInsight(ctx context.Context, insightStatus *updatestatus.MachineConfigPoolProgressInsightStatus, informer, uid string) error {
	insight := &updatestatus.MachineConfigPoolProgressInsight{
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", informer, uid),
		},
		Status: *insightStatus.DeepCopy(),
	}

	_, err := c.mcpInsights.Create(ctx, insight, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating machine config pool insight: %w", err)
	}

	return nil
}

func (c *updateStatusController) updateMachineConfigPoolProgressInsight(ctx context.Context, current *updatestatus.MachineConfigPoolProgressInsight, insightStatus *updatestatus.MachineConfigPoolProgressInsightStatus) error {
	insight := current.DeepCopy()
	insight.Status = *insightStatus.DeepCopy()

	_, err := c.mcpInsights.UpdateStatus(ctx, insight, v1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating machine config pool insight: %w", err)
	}

	return nil
}

func (c *updateStatusController) syncNodeProgressInsight(ctx context.Context, informer, uid string) error {
	insightStatus := c.state.getNodeProgressInsight(informer, uid)
	name := fmt.Sprintf("%s-%s", informer, uid)

	current, err := c.nodeInsights.Get(ctx, name, v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error getting node insight: %w", err)
	}

	if insightStatus == nil {
		return c.deleteNodeProgressInsight(ctx, current, err != nil && errors.IsNotFound(err))
	}

	return c.upsertNodeProgressInsight(ctx, current, insightStatus, informer, uid)
}

func (c *updateStatusController) deleteNodeProgressInsight(ctx context.Context, current *updatestatus.NodeProgressInsight, notFound bool) error {
	if notFound || current == nil || current.DeletionTimestamp != nil {
		return nil
	}

	err := c.nodeInsights.Delete(ctx, current.Name, v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error deleting node insight: %w", err)
	}
	return nil
}

func (c *updateStatusController) upsertNodeProgressInsight(ctx context.Context, current *updatestatus.NodeProgressInsight, insightStatus *updatestatus.NodeProgressInsightStatus, informer, uid string) error {
	if current == nil {
		return c.createNodeProgressInsight(ctx, insightStatus, informer, uid)
	}
	return c.updateNodeProgressInsight(ctx, current, insightStatus)
}

func (c *updateStatusController) createNodeProgressInsight(ctx context.Context, insightStatus *updatestatus.NodeProgressInsightStatus, informer, uid string) error {
	insight := &updatestatus.NodeProgressInsight{
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", informer, uid),
		},
		Status: *insightStatus.DeepCopy(),
	}

	_, err := c.nodeInsights.Create(ctx, insight, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating node insight: %w", err)
	}

	return nil
}

func (c *updateStatusController) updateNodeProgressInsight(ctx context.Context, current *updatestatus.NodeProgressInsight, insightStatus *updatestatus.NodeProgressInsightStatus) error {
	insight := current.DeepCopy()
	insight.Status = *insightStatus.DeepCopy()

	_, err := c.nodeInsights.UpdateStatus(ctx, insight, v1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating node insight: %w", err)
	}

	return nil
}

func (c *updateStatusController) syncHealthInsight(ctx context.Context, informer, uid string) error {
	insightStatus := c.state.getHealthInsight(informer, uid)
	name := fmt.Sprintf("%s-%s", informer, uid)

	current, err := c.healthInsights.Get(ctx, name, v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error getting health insight: %w", err)
	}

	if insightStatus == nil {
		return c.deleteHealthInsight(ctx, current, err != nil && errors.IsNotFound(err))
	}

	return c.upsertHealthInsight(ctx, current, insightStatus, informer, uid)
}

func (c *updateStatusController) deleteHealthInsight(ctx context.Context, current *updatestatus.HealthInsight, notFound bool) error {
	if notFound || current == nil || current.DeletionTimestamp != nil {
		return nil
	}

	err := c.healthInsights.Delete(ctx, current.Name, v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error deleting health insight: %w", err)
	}
	return nil
}

func (c *updateStatusController) upsertHealthInsight(ctx context.Context, current *updatestatus.HealthInsight, insightStatus *updatestatus.HealthInsightStatus, informer, uid string) error {
	if current == nil {
		return c.createHealthInsight(ctx, insightStatus, informer, uid)
	}
	return c.updateHealthInsight(ctx, current, insightStatus)
}

func (c *updateStatusController) createHealthInsight(ctx context.Context, insightStatus *updatestatus.HealthInsightStatus, informer, uid string) error {
	insight := &updatestatus.HealthInsight{
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", informer, uid),
		},
		Status: *insightStatus.DeepCopy(),
	}

	_, err := c.healthInsights.Create(ctx, insight, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error creating health insight: %w", err)
	}

	return nil
}

func (c *updateStatusController) updateHealthInsight(ctx context.Context, current *updatestatus.HealthInsight, insightStatus *updatestatus.HealthInsightStatus) error {
	insight := current.DeepCopy()
	insight.Status = *insightStatus.DeepCopy()

	_, err := c.healthInsights.UpdateStatus(ctx, insight, v1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating health insight: %w", err)
	}

	return nil
}
