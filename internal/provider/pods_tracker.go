package provider

import (
	"context"
	"errors"
	openapi "github.com/lucklypriyansh-2/salad-client"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		updatedPod := pod.DeepCopy()
		ok := pt.handlePodUpdates(updatedPod)
		if ok {
			pt.updateCallback(updatedPod)
		}
	}
}

func (pt *PodsTracker) removeStalePods() {
	pt.logger.Debug("Pod notifier remove stale pods called")
}

func (pt *PodsTracker) handlePodUpdates(pod *corev1.Pod) bool {
	if pt.isPodStatusUpdateRequired(pod) {
		pt.logger.Infof("pod %s will skip pod status update", pod.Name)
		return false
	}
	newStatus, err := pt.handler.GetPodStatus(pt.ctx, pod.Namespace, pod.Name)
	if err == nil && newStatus != nil {
		newStatus.DeepCopyInto(&pod.Status)
		return true
	}
	var openApiErr openapi.GenericOpenAPIError
	if pod.Status.Phase == corev1.PodRunning && errors.As(err, &openApiErr) {
		pod.Status.Phase = corev1.PodFailed
		pod.Status.Reason = "NotFoundOnProvider"
		pod.Status.Message = "the workload has been deleted from salad cloud"
		now := metav1.NewTime(time.Now())
		for i := range pod.Status.ContainerStatuses {
			if pod.Status.ContainerStatuses[i].State.Running == nil {
				continue
			}

			pod.Status.ContainerStatuses[i].State.Terminated = &corev1.ContainerStateTerminated{
				ExitCode:    137,
				Reason:      "NotFoundOnProvider",
				Message:     "the workload has been deleted from salad cloud",
				FinishedAt:  now,
				StartedAt:   pod.Status.ContainerStatuses[i].State.Running.StartedAt,
				ContainerID: pod.Status.ContainerStatuses[i].ContainerID,
			}
			pod.Status.ContainerStatuses[i].State.Running = nil
		}
		return true
	}
	return false
}

func (pt *PodsTracker) isPodStatusUpdateRequired(pod *corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodSucceeded || // Pod completed its execution
		pod.Status.Phase == corev1.PodFailed ||
		pod.Status.Reason == "ProviderFailed" || // in case if provider failed to create/register the pod
		pod.DeletionTimestamp != nil // Terminating
}
