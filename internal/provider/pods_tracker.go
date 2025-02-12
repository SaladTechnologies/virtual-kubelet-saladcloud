package provider

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/models"
	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/utils"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	corev1listers "k8s.io/client-go/listers/core/v1"
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
	pt.logger.Debug("remove stale Pods from cluster")
	clusterPods, err := pt.podLister.List(labels.Everything())
	if err != nil {
		pt.logger.WithError(err).Errorf("removeStalePodsInCluster: failed to retrieve pods list")
		return
	}
	activePods, err := pt.handler.GetPods(pt.ctx)
	if err != nil {
		pt.logger.WithError(err).Errorf("removeStalePodsInCluster: failed to retrieve active container groups")
		return
	}
	clusterPodMap := make(map[string]bool)
	for _, pod := range clusterPods {
		key := utils.GetPodName(pod.Namespace, pod.Name, pod)
		clusterPodMap[key] = true
	}
	for i := range activePods {
		if _, exists := clusterPodMap[activePods[i].Spec.Containers[0].Name]; !exists {
			pt.logger.Debugf("removeStalePodsInCluster: removing stale pod: %s", activePods[i].Name)
			err := pt.handler.DeletePod(pt.ctx, activePods[i])
			if err != nil {
				pt.logger.WithError(err).Errorf("removeStalePodsInCluster: failed to remove stale pod %v", activePods[i].Name)
			}
		}
	}
}

func (pt *PodsTracker) handlePodUpdates(pod *corev1.Pod) bool {
	pt.logger.Debug("Processing Pod Updates")
	if pt.isPodStatusUpdateRequired(pod) {
		pt.logger.Infof("handlePodStatusUpdate: Skipping pod status update for pod %s", pod.Name)
		return false
	}
	podCurrentStatus, err := pt.handler.GetPodStatus(pt.ctx, pod.Namespace, pod.Name)
	if err == nil && podCurrentStatus != nil {
		podCurrentStatus.DeepCopyInto(&pod.Status)
		return true
	}
	if err != nil {
		var apiError *models.APIError
		if errors.As(err, &apiError) && pod.Status.Phase == corev1.PodRunning && apiError.StatusCode == http.StatusNotFound {
			return pt.handlePodNotFound(pod)
		}
		pt.logger.WithError(err).Errorf("handlePodStatusUpdate: Failed to retrieve pod %v status from provider", pod.Name)
		return false
	}
	return true
}

func (pt *PodsTracker) handlePodNotFound(pod *corev1.Pod) bool {
	pt.logger.Infof("handlePodNotFound: Pod %s not found on the provider, updating status", pod.Name)
	for i := range pod.Status.ContainerStatuses {
		pod.Status.Phase = corev1.PodFailed
		pod.Status.Reason = "NotFoundOnProvider"
		pod.Status.Message = "The container group has been deleted"
		now := metav1.NewTime(time.Now())
		if pod.Status.ContainerStatuses[i].State.Running == nil {
			continue
		}
		pod.Status.ContainerStatuses[i].State.Terminated = &corev1.ContainerStateTerminated{
			ExitCode:    137,
			Reason:      "NotFoundOnProvider",
			Message:     "The container group has been deleted",
			FinishedAt:  now,
			StartedAt:   pod.Status.ContainerStatuses[i].State.Running.StartedAt,
			ContainerID: pod.Status.ContainerStatuses[i].ContainerID,
		}
		pod.Status.ContainerStatuses[i].State.Running = nil
	}
	return true
}

func (pt *PodsTracker) isPodStatusUpdateRequired(pod *corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodSucceeded || // Pod completed its execution
		pod.Status.Phase == corev1.PodFailed ||
		pod.Status.Reason == "ProviderFailed" || // in case if provider failed to create/register the pod
		pod.DeletionTimestamp != nil // Terminating
}
