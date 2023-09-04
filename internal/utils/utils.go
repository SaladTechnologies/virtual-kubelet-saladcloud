package utils

import (
	saladclient "github.com/lucklypriyansh-2/salad-client"
	corev1 "k8s.io/api/core/v1"
)

// write a function that takes the podspec and return the number of cpu and memory in a integer format and covert cpu to core format and memory to gigabyte format
func GetPodResource(podSpec corev1.PodSpec) (cpu int64, memory int64) {
	for _, container := range podSpec.Containers {
		cpu += container.Resources.Requests.Cpu().MilliValue()
		memory += container.Resources.Requests.Memory().Value()
	}
	return
}

func GetPodName(nameSpace, containerGroup string) string {
	return "salad-cloud-" + nameSpace + "-" + containerGroup
}

func GetPodPhaseFromContainerGroupState(containerGroupState saladclient.ContainerGroupState) corev1.PodPhase {

	switch containerGroupState.Status {
	case saladclient.CONTAINERGROUPSTATUS_PENDING:
		return corev1.PodPending
	case saladclient.CONTAINERGROUPSTATUS_RUNNING:
		{
			if containerGroupState.InstanceStatusCount.RunningCount > 0 {
				return corev1.PodRunning
			}
			return corev1.PodPending
		}
	case saladclient.CONTAINERGROUPSTATUS_FAILED:
		return corev1.PodFailed
	case saladclient.CONTAINERGROUPSTATUS_SUCCEEDED:
		return corev1.PodSucceeded
	case saladclient.CONTAINERGROUPSTATUS_STOPPED:
		return corev1.PodSucceeded
	case saladclient.CONTAINERGROUPSTATUS_DEPLOYING:
		return corev1.PodPending

	}

}
