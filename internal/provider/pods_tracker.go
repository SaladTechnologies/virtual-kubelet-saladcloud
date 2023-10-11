package provider

import (
	"context"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"time"
)

// Define the intervals for pod status updates and stale pod cleanup
var podStatusUpdateInterval = 5 * time.Second
var stalePodCleanupInterval = 5 * time.Minute

type PodsTrackerHandler interface {
	GetPods(ctx context.Context) ([]*corev1.Pod, error)
	GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error)
	DeletePod(ctx context.Context, pod *corev1.Pod) error
}

type PodsTracker struct {
	ctx            context.Context
	logger         log.Logger
	podLister      corev1listers.PodLister
	updateCallback func(*corev1.Pod)
	handler        PodsTrackerHandler
}

func (pt *PodsTracker) BeginPodTracking(ctx context.Context) {
	statusUpdatesTimer := time.NewTimer(podStatusUpdateInterval)
	cleanupTimer := time.NewTimer(stalePodCleanupInterval)
	defer statusUpdatesTimer.Stop()
	defer cleanupTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.G(ctx).WithError(ctx.Err()).Debug("Pod status update loop exiting")
			return
		case <-statusUpdatesTimer.C:
			pt.updatePods()
			statusUpdatesTimer.Reset(podStatusUpdateInterval)
		case <-cleanupTimer.C:
			pt.removeStalePods()
			cleanupTimer.Reset(stalePodCleanupInterval)
		}
	}
}

func (pt *PodsTracker) updatePods() {
	pt.logger.Debug("Pod notifier update pods called")
	k8sPods, err := pt.podLister.List(labels.Everything())
	if err != nil {
		pt.logger.WithError(err).Errorf("failed to retrieve pods list")
		return
	}
	for _, pod := range k8sPods {
		pt.logger.Debug("update pod", pod.Name)
	}
}

func (pt *PodsTracker) removeStalePods() {
	pt.logger.Debug("Pod notifier remove stale pods called")
}
