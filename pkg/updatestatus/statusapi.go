package updatestatus

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	updatev1alpha1 "github.com/openshift/api/update/v1alpha1"
)

// insightExpirations is UID -> expiration time map
type insightExpirations map[string]time.Time

type informer struct {
	// name is the name of the informer
	name string

	cvInsights     map[string]*updatev1alpha1.ClusterVersionProgressInsightStatus
	coInsights     map[string]*updatev1alpha1.ClusterOperatorProgressInsightStatus
	mcpInsights    map[string]*updatev1alpha1.MachineConfigPoolProgressInsightStatus
	nodeInsights   map[string]*updatev1alpha1.NodeProgressInsightStatus
	healthInsights map[string]*updatev1alpha1.HealthInsightStatus
}

// statusApi is the desired state of the status API ConfigMap. It is updated when new insights are received.
// Any access to the struct should be done with the lock held.
type updateStatusApi struct {
	sync.Mutex

	// informers tracks insights contributed by individual informers
	informers map[string]*informer

	// unknownInsightExpirations is a map of informer -> map of UID -> expiration time. It is used to track insights
	// that were reported by informers but are no longer known to them. The API keeps unknown insights until they
	// expire. If an insight is reported as known again before it expires, it is removed from the map.
	// TODO (muller): Needs to periodically rebuilt to avoid leaking memory
	unknownInsightExpirations map[string]insightExpirations

	// processed is the number of insights processed, used for testing
	processed int

	// TODO: Get rid of this and use `clock.Clock` in all controllers, passed from start.go main function's
	// controllercmd.ControllerContext
	now func() time.Time
}

// processInsightMsg validates the message and if valid, updates the status API with the included
// insight. Returns the queue keys to sync.
func (c *updateStatusApi) processInsightMsg(message informerMsg) []string {
	c.Lock()
	defer c.Unlock()

	c.processed++

	if err := message.validate(); err != nil {
		klog.Warningf("USC :: Collector :: Invalid message: %v", err)
		return nil
	}

	klog.Infof("USC :: Collector :: Received insight from informer %q", message.informer)
	syncs := c.updateInsightInStatusApi(message)
	c.removeUnknownInsights(message)

	return syncs
}

// updateInsightInStatusApi updates the status API using the message.
// Assumes the statusApi field is locked.
func (c *updateStatusApi) updateInsightInStatusApi(msg informerMsg) []string {
	if c.informers == nil {
		c.informers = map[string]*informer{msg.informer: {name: msg.informer}}
	} else if _, ok := c.informers[msg.informer]; !ok {
		c.informers[msg.informer] = &informer{name: msg.informer}
	}

	switch {
	case msg.cvInsight != nil:
		return c.informers[msg.informer].ingestClusterVersionProgressInsight(msg.uid, msg.cvInsight)
	case msg.coInsight != nil:
		return c.informers[msg.informer].ingestClusterOperatorProgressInsight(msg.uid, msg.coInsight)
	case msg.mcpInsight != nil:
		return c.informers[msg.informer].ingestMachineConfigPoolProgressInsight(msg.uid, msg.mcpInsight)
	case msg.nodeInsight != nil:
		return c.informers[msg.informer].ingestNodeProgressInsight(msg.uid, msg.nodeInsight)
	case msg.healthInsight != nil:
		return c.informers[msg.informer].ingestHealthInsight(msg.uid, msg.healthInsight)
	}

	panic(fmt.Sprintf("unknown insight type in message: %v", msg))
}

func (c *updateStatusApi) getClusterVersionProgressInsight(informer string, uid string) *updatev1alpha1.ClusterVersionProgressInsightStatus {
	c.Lock()
	defer c.Unlock()

	if i, ok := c.informers[informer]; ok {
		if insight, ok := i.cvInsights[uid]; ok {
			return insight.DeepCopy()
		}
	}

	return nil
}

func (c *updateStatusApi) getClusterOperatorProgressInsight(informer string, uid string) *updatev1alpha1.ClusterOperatorProgressInsightStatus {
	c.Lock()
	defer c.Unlock()

	if i, ok := c.informers[informer]; ok {
		if insight, ok := i.coInsights[uid]; ok {
			return insight.DeepCopy()
		}
	}

	return nil
}

func (c *updateStatusApi) getMachineConfigPoolProgressInsight(informer string, uid string) *updatev1alpha1.MachineConfigPoolProgressInsightStatus {
	c.Lock()
	defer c.Unlock()

	if i, ok := c.informers[informer]; ok {
		if insight, ok := i.mcpInsights[uid]; ok {
			return insight.DeepCopy()
		}
	}

	return nil
}

func (c *updateStatusApi) getNodeProgressInsight(informer string, uid string) *updatev1alpha1.NodeProgressInsightStatus {
	c.Lock()
	defer c.Unlock()

	if i, ok := c.informers[informer]; ok {
		if insight, ok := i.nodeInsights[uid]; ok {
			return insight.DeepCopy()
		}
	}

	return nil
}

func (c *updateStatusApi) getHealthInsight(informer string, uid string) *updatev1alpha1.HealthInsightStatus {
	c.Lock()
	defer c.Unlock()

	if i, ok := c.informers[informer]; ok {
		if insight, ok := i.healthInsights[uid]; ok {
			return insight.DeepCopy()
		}
	}

	return nil
}

// ingestClusterVersionProgressInsight updates the status API with the ClusterVersionProgressInsight.
// Assumes the statusApi field is locked.
func (i *informer) ingestClusterVersionProgressInsight(uid string, insight *updatev1alpha1.ClusterVersionProgressInsightStatus) []string {
	if i.cvInsights == nil {
		i.cvInsights = map[string]*updatev1alpha1.ClusterVersionProgressInsightStatus{}
	}
	i.cvInsights[uid] = insight
	return []string{fmt.Sprintf("%s/cv/%s", i.name, uid)}
}

// ingestClusterOperatorProgressInsight updates the status API with the ClusterOperatorProgressInsight.
// Assumes the statusApi field is locked.
func (i *informer) ingestClusterOperatorProgressInsight(uid string, insight *updatev1alpha1.ClusterOperatorProgressInsightStatus) []string {
	if i.coInsights == nil {
		i.coInsights = map[string]*updatev1alpha1.ClusterOperatorProgressInsightStatus{}
	}
	i.coInsights[uid] = insight
	return []string{fmt.Sprintf("%s/co/%s", i.name, uid)}
}

// ingestMachineConfigPoolProgressInsight updates the status API with the MachineConfigPoolProgressInsight.
// Assumes the statusApi field is locked.
func (i *informer) ingestMachineConfigPoolProgressInsight(uid string, insight *updatev1alpha1.MachineConfigPoolProgressInsightStatus) []string {
	if i.mcpInsights == nil {
		i.mcpInsights = map[string]*updatev1alpha1.MachineConfigPoolProgressInsightStatus{}
	}
	i.mcpInsights[uid] = insight
	return []string{fmt.Sprintf("%s/mcp/%s", i.name, uid)}
}

// ingestNodeProgressInsight updates the status API with the NodeProgressInsight.
// Assumes the statusApi field is locked.
func (i *informer) ingestNodeProgressInsight(uid string, insight *updatev1alpha1.NodeProgressInsightStatus) []string {
	if i.nodeInsights == nil {
		i.nodeInsights = map[string]*updatev1alpha1.NodeProgressInsightStatus{}
	}
	i.nodeInsights[uid] = insight
	return []string{fmt.Sprintf("%s/node/%s", i.name, uid)}
}

// ingestHealthInsight updates the status API with the HealthInsight.
// Assumes the statusApi field is locked.
func (i *informer) ingestHealthInsight(uid string, insight *updatev1alpha1.HealthInsightStatus) []string {
	if i.healthInsights == nil {
		i.healthInsights = map[string]*updatev1alpha1.HealthInsightStatus{}
	}
	i.healthInsights[uid] = insight
	return []string{fmt.Sprintf("%s/health/%s", i.name, uid)}
}

// removeUnknownInsights removes insights from the status API that are no longer reported as known to the informer
// that originally reported them.
// Assumes the statusApi field is locked.
func (c *updateStatusApi) removeUnknownInsights(message informerMsg) {
	known := sets.New(message.knownInsights...)
	known.Insert(message.uid)

	c.handleUnknownInsightsByInformer(message.informer, known)
}

func (c *updateStatusApi) handleUnknownInsightsByInformer(informer string, known sets.Set[string]) {
	cvFilter := c.makeClusterVersionInsightFilter(informer, known)
	coFilter := c.makeClusterOperatorInsightFilter(informer, known)
	mcpFilter := c.makeMachineConfigPoolInsightFilter(informer, known)
	nodeFilter := c.makeNodeInsightFilter(informer, known)
	healthFilter := c.makeHealthInsightFilter(informer, known)

	for i := range c.informers {
		if c.informers[i].name != informer {
			continue
		}
		c.informers[i].cvInsights = cvFilter(c.informers[i].cvInsights)
		c.informers[i].coInsights = coFilter(c.informers[i].coInsights)
		c.informers[i].mcpInsights = mcpFilter(c.informers[i].mcpInsights)
		c.informers[i].nodeInsights = nodeFilter(c.informers[i].nodeInsights)
		c.informers[i].healthInsights = healthFilter(c.informers[i].healthInsights)
	}

	if len(c.unknownInsightExpirations[informer]) == 0 {
		delete(c.unknownInsightExpirations, informer)
	}
	if len(c.unknownInsightExpirations) == 0 {
		c.unknownInsightExpirations = nil
	}
}

type keepInsightFunc func(uid string) bool
type cvInsightFilter func(insights map[string]*updatev1alpha1.ClusterVersionProgressInsightStatus) map[string]*updatev1alpha1.ClusterVersionProgressInsightStatus
type coInsightFilter func(insights map[string]*updatev1alpha1.ClusterOperatorProgressInsightStatus) map[string]*updatev1alpha1.ClusterOperatorProgressInsightStatus
type mcpInsightFilter func(insights map[string]*updatev1alpha1.MachineConfigPoolProgressInsightStatus) map[string]*updatev1alpha1.MachineConfigPoolProgressInsightStatus
type nodeInsightFilter func(insights map[string]*updatev1alpha1.NodeProgressInsightStatus) map[string]*updatev1alpha1.NodeProgressInsightStatus
type healthInsightFilter func(insights map[string]*updatev1alpha1.HealthInsightStatus) map[string]*updatev1alpha1.HealthInsightStatus

// type wpInsightFilter func(insights []updatev1alpha1.WorkerPoolInsight) []updatev1alpha1.WorkerPoolInsight
func (c *updateStatusApi) makeKeep(informer string, known sets.Set[string]) keepInsightFunc {
	now := c.now()
	return func(uid string) bool {
		if known.Has(uid) {
			if c.unknownInsightExpirations != nil && c.unknownInsightExpirations[informer] != nil {
				delete(c.unknownInsightExpirations[informer], uid)
			}
			return true
		}

		logKeep := func(expire time.Time) {
			klog.V(2).Infof("USC :: Collector :: Keeping insight %q until %s after it is no longer reported as known by informer %q", uid, expire, informer)
		}

		expireIn := now.Add(unknownInsightGracePeriod)
		switch {
		// Two cases when we first consider an insight as unknown -> set expiration
		case c.unknownInsightExpirations == nil:
			c.unknownInsightExpirations = map[string]insightExpirations{informer: {uid: expireIn}}
			logKeep(expireIn)
			return true

		case c.unknownInsightExpirations[informer][uid].IsZero():
			if _, hasInformer := c.unknownInsightExpirations[informer]; !hasInformer {
				c.unknownInsightExpirations[informer] = insightExpirations{}
			}
			c.unknownInsightExpirations[informer][uid] = expireIn
			logKeep(expireIn)
			return true

		// Already set for expiration but still in grace period -> keep insight
		case c.unknownInsightExpirations[informer][uid].After(now):
			logKeep(c.unknownInsightExpirations[informer][uid])
			return true
		}

		// Already set for expiration and grace period expired -> drop insight
		delete(c.unknownInsightExpirations[informer], uid)
		klog.V(2).Infof("USC :: Collector :: Dropped insight %q because it is no longer reported as known by informer %q", uid, informer)
		return false
	}
}

// makeExpirationFilter considers potential expiration of an insight present in the API based on whether the informer
// knows about it.
// If the informer knows about the insight, it is not dropped from the API and any previous expiration is cancelled.
// If the informer does not know about the insight then it is either set to expire in the future if no expiration is
// set yet, or the expiration is checked to see whether the insight should be dropped.
func (c *updateStatusApi) makeClusterVersionInsightFilter(informer string, known sets.Set[string]) cvInsightFilter {
	return func(insights map[string]*updatev1alpha1.ClusterVersionProgressInsightStatus) map[string]*updatev1alpha1.ClusterVersionProgressInsightStatus {
		keep := c.makeKeep(informer, known)
		filtered := make(map[string]*updatev1alpha1.ClusterVersionProgressInsightStatus, len(insights))

		for i := range insights {
			if keep(insights[i].Name) {
				filtered[insights[i].Name] = insights[i]
			}
		}

		if len(filtered) > 0 {
			return filtered
		}
		return nil
	}
}

func (c *updateStatusApi) makeClusterOperatorInsightFilter(informer string, known sets.Set[string]) coInsightFilter {
	return func(insights map[string]*updatev1alpha1.ClusterOperatorProgressInsightStatus) map[string]*updatev1alpha1.ClusterOperatorProgressInsightStatus {
		keep := c.makeKeep(informer, known)
		filtered := make(map[string]*updatev1alpha1.ClusterOperatorProgressInsightStatus, len(insights))

		for i := range insights {
			if keep(insights[i].Name) {
				filtered[insights[i].Name] = insights[i]
			}
		}

		if len(filtered) > 0 {
			return filtered
		}
		return nil
	}
}

func (c *updateStatusApi) makeMachineConfigPoolInsightFilter(informer string, known sets.Set[string]) mcpInsightFilter {
	return func(insights map[string]*updatev1alpha1.MachineConfigPoolProgressInsightStatus) map[string]*updatev1alpha1.MachineConfigPoolProgressInsightStatus {
		keep := c.makeKeep(informer, known)
		filtered := make(map[string]*updatev1alpha1.MachineConfigPoolProgressInsightStatus, len(insights))

		for i := range insights {
			if keep(insights[i].Name) {
				filtered[insights[i].Name] = insights[i]
			}
		}

		if len(filtered) > 0 {
			return filtered
		}
		return nil
	}
}

func (c *updateStatusApi) makeNodeInsightFilter(informer string, known sets.Set[string]) nodeInsightFilter {
	return func(insights map[string]*updatev1alpha1.NodeProgressInsightStatus) map[string]*updatev1alpha1.NodeProgressInsightStatus {
		keep := c.makeKeep(informer, known)
		filtered := make(map[string]*updatev1alpha1.NodeProgressInsightStatus, len(insights))

		for i := range insights {
			if keep(insights[i].Name) {
				filtered[insights[i].Name] = insights[i]
			}
		}

		if len(filtered) > 0 {
			return filtered
		}
		return nil
	}
}

func (c *updateStatusApi) makeHealthInsightFilter(informer string, known sets.Set[string]) healthInsightFilter {
	return func(insights map[string]*updatev1alpha1.HealthInsightStatus) map[string]*updatev1alpha1.HealthInsightStatus {
		keep := c.makeKeep(informer, known)
		filtered := make(map[string]*updatev1alpha1.HealthInsightStatus, len(insights))

		for i := range insights {
			if keep(i) {
				filtered[i] = insights[i]
			}
		}

		if len(filtered) > 0 {
			return filtered
		}
		return nil
	}
}
