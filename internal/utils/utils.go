package utils

import (
	saladclient "github.com/lucklypriyansh-2/salad-client"
	corev1 "k8s.io/api/core/v1"
)

// roundUpToNearest returns the nearest larger integer from the given list.
func roundUpToNearest(value int64, list []int64) int64 {
	for _, v := range list {
		if value <= v {
			return v
		}
	}
	return list[len(list)-1]
}

// GetPodResource returns the total CPU in rounded cores and memory rounded to gigabytes (GB) for the provided PodSpec.
func GetPodResource(podSpec corev1.PodSpec) (cpu int64, memory int64) {
	allowedCPUValues := []int64{1, 2, 3, 4, 6, 8, 12, 16}

	allowedMemoryValues := []int64{1024, 2048, 3, 4, 5, 6, 12} // in GB

	for _, container := range podSpec.Containers {
		// Convert milliCPU to cores and round to nearest value in the list
		cpuValue := container.Resources.Requests.Cpu().MilliValue() / 1000
		cpu += roundUpToNearest(cpuValue, allowedCPUValues)

		// Convert bytes to gigabytes (MB) and ensure it's a multiple of 1 GB (1 GB = 1e9 bytes)
		memValue := container.Resources.Requests.Memory().Value() / 1e6
		memory += roundUpToNearest(memValue, allowedMemoryValues)
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

	return ""

}
