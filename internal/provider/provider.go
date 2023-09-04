package provider

import (
	"context"
	"fmt"
	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/models"
	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/utils"
	saladclient "github.com/lucklypriyansh-2/salad-client"
	dto "github.com/prometheus/client_model/go"
	nodeapi "github.com/virtual-kubelet/virtual-kubelet/node/api"
	"github.com/virtual-kubelet/virtual-kubelet/node/api/statsv1alpha1"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

type SaladCloudProvider struct {
	inputVars models.InputVars
}

func NewSaladCloudProvider(ctx context.Context, inputVars models.InputVars) (*SaladCloudProvider, error) {
	return &SaladCloudProvider{
		inputVars: inputVars,
	}, nil
}

func (p *SaladCloudProvider) CreatePod(ctx context.Context, pod *corev1.Pod) error {

	cpu, memory := utils.GetPodResource(pod.Spec)
	createContainerGroup := *saladclient.NewCreateContainerGroup(utils.GetPodName(pod.Namespace, pod.Name), *saladclient.NewCreateContainer(pod.Spec.Containers[0].Image, *saladclient.NewContainerResourceRequirements(int32(cpu), int32(memory))), saladclient.ContainerRestartPolicy("always"), int32(123)) // CreateContainerGroup |
	configuration := saladclient.NewConfiguration()
	apiClient := saladclient.NewAPIClient(configuration)
	resp, r, err := apiClient.ContainerGroupsAPI.CreateContainerGroup(context.Background(), p.inputVars.ProjectName, p.inputVars.OrganizationName).CreateContainerGroup(createContainerGroup).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ContainerGroupsAPI.CreateContainerGroup``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateContainerGroup`: ContainerGroup
	fmt.Fprintf(os.Stdout, "Response from `ContainerGroupsAPI.CreateContainerGroup`: %v\n", resp)
	return nil
}

func (p *SaladCloudProvider) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	return nil
}

func (p *SaladCloudProvider) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	return nil
}

func (p *SaladCloudProvider) GetPod(ctx context.Context, namespace string, name string) (*corev1.Pod, error) {

	configuration := saladclient.NewConfiguration()
	apiClient := saladclient.NewAPIClient(configuration)
	resp, r, err := apiClient.ContainerGroupsAPI.GetContainerGroup(context.Background(), p.inputVars.OrganizationName, p.inputVars.ProjectName, utils.GetPodName(namespace, name)).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ContainerGroupsAPI.GetContainerGroup``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return nil, err
	}
	fmt.Fprintf(os.Stdout, "Response from `ContainerGroupsAPI.GetContainerGroup`: %v\n", resp)
	startTime := metav1.NewTime(resp.CreateTime)
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  resp.Name,
					Image: resp.Container.Image,
				},
			},
		},
		Status: corev1.PodStatus{
			Phase:     utils.GetPodPhaseFromContainerGroupState(resp.CurrentState),
			StartTime: &startTime,
		},
	}

	return pod, nil
}

func (p *SaladCloudProvider) GetPodStatus(ctx context.Context, namespace string, name string) (*corev1.PodStatus, error) {
	return nil, nil
}

func (p *SaladCloudProvider) GetPods(ctx context.Context) ([]*corev1.Pod, error) {
	pods := make([]*corev1.Pod, 0)
	return pods, nil
}

func (p *SaladCloudProvider) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts nodeapi.ContainerLogOpts) (io.ReadCloser, error) {
	return nil, nil
}

func (p *SaladCloudProvider) RunInContainer(ctx context.Context, namespace, podName, containerName string, cmd []string, attach nodeapi.AttachIO) error {
	return nil
}

func (p *SaladCloudProvider) AttachToContainer(ctx context.Context, namespace, podName, containerName string, attach nodeapi.AttachIO) error {
	return nil
}

func (p *SaladCloudProvider) GetStatsSummary(context.Context) (*statsv1alpha1.Summary, error) {
	return nil, nil
}

func (p *SaladCloudProvider) GetMetricsResource(context.Context) ([]*dto.MetricFamily, error) {
	return nil, nil
}

func (p *SaladCloudProvider) PortForward(ctx context.Context, namespace, pod string, port int32, stream io.ReadWriteCloser) error {
	return nil
}
