package utils

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	saladclient "github.com/SaladTechnologies/salad-client"
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

// GetPodResource returns the total CPU in rounded cores and memory rounded to gibibytes (GiB) for the provided PodSpec.
func GetPodResource(podSpec corev1.PodSpec) (cpu int64, memory int64) {
	allowedCPUValues := []int64{1, 2, 3, 4, 6, 8, 12, 16}

	allowedMemoryValues := []int64{1024, 2048, 3072, 4 * 1024, 6 * 1024, 8 * 1024, 12 * 1024, 16 * 1024, 24 * 1024, 30 * 1024, 38 * 1024, 60 * 1024} // in GiB

	for _, container := range podSpec.Containers {
		// Convert milliCPU to cores and round to nearest value in the list
		cpuValue := container.Resources.Requests.Cpu().MilliValue() / 1000
		cpu += roundUpToNearest(cpuValue, allowedCPUValues)

		// Convert bytes to gibibytes (GiB) and ensure it's a multiple of 1 GiB (1 GiB = 1024*1024 bytes)
		memValue := container.Resources.Requests.Memory().Value() / (1024 * 1024)
		memory += roundUpToNearest(memValue, allowedMemoryValues)
	}
	return
}

func GetPodName(nameSpace, containerGroup string, pod *corev1.Pod) string {
	if nameSpace == "" {
		nameSpace = pod.Namespace
	}
	if containerGroup == "" {
		containerGroup = pod.Name
	}
	if nameSpace == "" && containerGroup == "" && pod.Spec.Containers[0].Name != "" {
		return pod.Spec.Containers[0].Name
	}
	return nameSpace + "-" + containerGroup
}

func GetPodPhaseFromContainerGroupState(containerGroupState saladclient.ContainerGroupState) corev1.PodPhase {
	switch containerGroupState.Status {
	case saladclient.CONTAINERGROUPSTATUS_PENDING:
		return corev1.PodPending
	case saladclient.CONTAINERGROUPSTATUS_RUNNING:
		{
			if containerGroupState.InstanceStatusCounts.RunningCount > 0 {
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

func GetResponseBody(response *http.Response) (*saladclient.ProblemDetails, error) {
	if response == nil {
		return nil, fmt.Errorf("response was nil")
	}

	// Get response body for error info
	body, _ := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if closeErr != nil {
		return nil, closeErr
	}
	response.Body = io.NopCloser(bytes.NewBuffer(body))

	// Get a ProblemDetails struct from the body
	pd := saladclient.NewNullableProblemDetails(nil)
	err := pd.UnmarshalJSON(body)
	if err != nil {
		// Can't decode ProblemDetails, just make one and return the string
		npd := saladclient.NewProblemDetails()
		npd.SetType("unknown_error")
		npd.SetTitle("Error decoding response body")
		npd.SetDetail(string(body))
		pd.Set(npd)
	}
	return pd.Get(), nil
}
