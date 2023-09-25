package provider

import (
	"errors"
	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/models"
	corev1 "k8s.io/api/core/v1"
)

func MapPodToCreateContainerGroup(pod *corev1.Pod) (*models.CreateContainerGroupModel, error) {
	if len(pod.Spec.Containers) == 0 {
		return nil, errors.New("no containers found in the pod")
	}

	firstContainer := pod.Spec.Containers[0]
	envVars := make(map[string]string)
	for _, envVar := range firstContainer.Env {
		envVars[envVar.Name] = envVar.Value
	}

	cpuRequest := getResourceValue(firstContainer.Resources.Requests, corev1.ResourceCPU)
	memoryRequest := getResourceValue(firstContainer.Resources.Requests, corev1.ResourceMemory)

	containerResources := models.Resources{
		Requests: models.ResourceList{
			CPU:    cpuRequest,
			Memory: memoryRequest,
		},
		Limits: models.ResourceList{
			CPU:    cpuRequest,
			Memory: memoryRequest,
		},
	}

	containerSpec := models.ContainerSpec{
		Resources:            containerResources,
		EnvironmentVariables: envVars,
		Image:                firstContainer.Image,
		Command:              firstContainer.Command,
	}

	group := &models.CreateContainerGroupModel{
		Name:           pod.Name,
		Container:      containerSpec,
		RestartPolicy:  string(pod.Spec.RestartPolicy),
		LivenessProbe:  convertProbe(firstContainer.LivenessProbe),
		ReadinessProbe: convertProbe(firstContainer.ReadinessProbe),
		StartupProbe:   convertProbe(firstContainer.StartupProbe),
		Annotations:    pod.Annotations,
	}

	return group, nil
}

func convertProbe(probe *corev1.Probe) models.Probe {
	if probe == nil {
		return models.Probe{}
	}
	return models.Probe{
		HTTPGetAction: &models.HTTPGetAction{
			Path:   probe.HTTPGet.Path,
			Port:   int32(probe.HTTPGet.Port.IntValue()),
			Scheme: string(probe.HTTPGet.Scheme),
		},
		InitialDelaySeconds: probe.InitialDelaySeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		TimeoutSeconds:      probe.TimeoutSeconds,
		SuccessThreshold:    probe.SuccessThreshold,
		FailureThreshold:    probe.FailureThreshold,
	}
}
func getResourceValue(resources corev1.ResourceList, resourceType corev1.ResourceName) string {
	if val, ok := resources[resourceType]; ok {
		return val.String()
	}
	return ""
}
