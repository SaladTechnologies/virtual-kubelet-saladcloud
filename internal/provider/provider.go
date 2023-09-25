package provider

import (
	"context"
	"encoding/json"
	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/models"
	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/utils"
	saladclient "github.com/lucklypriyansh-2/salad-client"
	dto "github.com/prometheus/client_model/go"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	nodeapi "github.com/virtual-kubelet/virtual-kubelet/node/api"
	"github.com/virtual-kubelet/virtual-kubelet/node/api/statsv1alpha1"
	"github.com/virtual-kubelet/virtual-kubelet/trace"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

type SaladCloudProvider struct {
	inputVars       models.InputVars
	cpu             string
	memory          string
	pods            string
	storage         string
	operatingSystem string
	apiClient       *saladclient.APIClient
	countryCodes    []saladclient.CountryCode
}

const (
	defaultCPUCoresNumber  = "10000"
	defaultMemorySize      = "1Ti"
	defaultStorageSize     = "1Ti"
	defaultPodsLimit       = "1000"
	defaultOperatingSystem = "Linux"
)

func NewSaladCloudProvider(ctx context.Context, inputVars models.InputVars) (*SaladCloudProvider, error) {
	cloudProvider := &SaladCloudProvider{
		inputVars: inputVars,
		apiClient: saladclient.NewAPIClient(saladclient.NewConfiguration()),
	}
	cloudProvider.setNodeCapacity()
	cloudProvider.setCountryCodes([]string{"US"})

	return cloudProvider, nil
}
func (p *SaladCloudProvider) setCountryCodes(countries []string) {
	for _, countryCode := range countries {
		p.countryCodes = append(p.countryCodes, saladclient.CountryCode(countryCode))
	}
}

func (p *SaladCloudProvider) setNodeCapacity() {
	p.cpu = defaultCPUCoresNumber
	p.memory = defaultMemorySize
	p.pods = defaultPodsLimit
	p.storage = defaultStorageSize
	p.operatingSystem = defaultOperatingSystem
}

func (p *SaladCloudProvider) ConfigureNode(ctx context.Context, node *corev1.Node) {
	node.Status.Capacity = p.getNodeCapacity()
	node.Status.Allocatable = p.getNodeCapacity()
	node.Status.NodeInfo.OperatingSystem = p.operatingSystem
}

func (p *SaladCloudProvider) getNodeCapacity() corev1.ResourceList {
	resourceList := corev1.ResourceList{
		corev1.ResourceCPU:     resource.MustParse(p.cpu),
		corev1.ResourceMemory:  resource.MustParse(p.memory),
		corev1.ResourcePods:    resource.MustParse(p.pods),
		corev1.ResourceStorage: resource.MustParse(p.storage),
	}

	return resourceList
}

func (p *SaladCloudProvider) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	ctx, span := trace.StartSpan(ctx, "CreatePod")
	defer span.End()
	log.G(ctx).Debug("creating a CreatePod", pod.Name)

	createContainerObject := p.createContainersObject(pod)
	createContainerGroup := p.createContainerGroup(createContainerObject, pod)

	_, r, err := p.apiClient.
		ContainerGroupsAPI.CreateContainerGroup(
		p.contextWithAuth(),
		p.inputVars.OrganizationName,
		p.inputVars.ProjectName).CreateContainerGroup(
		createContainerGroup[0],
	).Execute()
	if err != nil {
		log.G(ctx).Errorf("Error when calling `ContainerGroupsAPI.CreateContainerGroupModel`", r)
		return err
	}

	// wait for 3 second
	time.Sleep(3 * time.Second)

	startHttpResponse, err := p.apiClient.ContainerGroupsAPI.StartContainerGroup(p.contextWithAuth(), p.inputVars.OrganizationName, p.inputVars.ProjectName, utils.GetPodName(pod.Namespace, pod.Name, nil)).Execute()
	if err != nil {
		log.G(ctx).Errorf("Error when calling `ContainerGroupsAPI.CreateContainerGroupModel`", startHttpResponse)
		err := p.DeletePod(ctx, pod)
		if err != nil {
			return err
		}
		return err
	}

	now := metav1.NewTime(time.Now())
	pod.ObjectMeta.CreationTimestamp = now
	pod.Status = corev1.PodStatus{
		Phase:     corev1.PodRunning,
		StartTime: &now,
		Conditions: []corev1.PodCondition{
			{
				Type:   corev1.PodInitialized,
				Status: corev1.ConditionTrue,
			},
			{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			},
			{
				Type:   corev1.PodScheduled,
				Status: corev1.ConditionTrue,
			},
		},
	}
	for _, container := range pod.Spec.Containers {
		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, corev1.ContainerStatus{
			Name:         container.Name,
			Image:        container.Image,
			Ready:        true,
			RestartCount: 0,
		})
	}

	log.G(ctx).Infof("Done creating the container and initiating  the startup ", pod)
	return nil
}

func (p *SaladCloudProvider) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	return nil
}

func (p *SaladCloudProvider) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	ctx, span := trace.StartSpan(ctx, "DeletePod")
	defer span.End()
	log.G(ctx).Debug("deleting a pod")
	response, err := p.apiClient.ContainerGroupsAPI.DeleteContainerGroup(p.contextWithAuth(), p.inputVars.OrganizationName, p.inputVars.ProjectName, utils.GetPodName(pod.Namespace, pod.Name, pod)).Execute()
	pod.Status.Phase = corev1.PodSucceeded
	pod.Status.Reason = "Pod Deleted"
	if err != nil {
		log.G(ctx).Errorf("Error when deleting the container ", response)
		return err
	}
	now := metav1.Now()
	for idx := range pod.Status.ContainerStatuses {
		pod.Status.ContainerStatuses[idx].Ready = false
		pod.Status.ContainerStatuses[idx].State = corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				Message:    "Salad Provider Pod Deleted",
				FinishedAt: now,
				Reason:     "Salad Provider Pod Deleted",
			},
		}
	}
	log.G(ctx).Infof("Done deleting the container ", pod)
	return nil
}

func (p *SaladCloudProvider) GetPod(ctx context.Context, namespace string, name string) (*corev1.Pod, error) {

	resp, r, err := saladclient.NewAPIClient(saladclient.NewConfiguration()).ContainerGroupsAPI.GetContainerGroup(p.contextWithAuth(), p.inputVars.OrganizationName, p.inputVars.ProjectName, utils.GetPodName(namespace, name, nil)).Execute()
	if err != nil {
		log.G(ctx).Errorf("Error when calling `ContainerGroupsAPI.GetPod`", r)
		return nil, err
	}
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
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  resp.Name,
					Image: resp.Container.Image,
					Ready: utils.GetPodPhaseFromContainerGroupState(resp.CurrentState) == corev1.PodRunning,
				},
			},
		},
	}

	return pod, nil
}

func (p *SaladCloudProvider) contextWithAuth() context.Context {
	auth := context.WithValue(
		context.Background(),
		saladclient.ContextAPIKeys,
		map[string]saladclient.APIKey{
			"ApiKeyAuth": {Key: p.inputVars.ApiKey},
		},
	)
	return auth
}

func (p *SaladCloudProvider) GetPodStatus(ctx context.Context, namespace string, name string) (*corev1.PodStatus, error) {
	ctx, span := trace.StartSpan(ctx, "GetPodStatus")
	defer span.End()

	containerGroup, response, err := p.apiClient.ContainerGroupsAPI.GetContainerGroup(p.contextWithAuth(), p.inputVars.OrganizationName, p.inputVars.ProjectName, utils.GetPodName(namespace, name, nil)).Execute()
	if err != nil {
		log.G(ctx).Errorf("ContainerGroupsAPI.GetPodStatus ", response)
		return nil, err
	}

	startTime := metav1.NewTime(containerGroup.CreateTime)
	return &corev1.PodStatus{
		Phase:     utils.GetPodPhaseFromContainerGroupState(containerGroup.CurrentState),
		StartTime: &startTime,
		ContainerStatuses: []corev1.ContainerStatus{
			{
				Name:  containerGroup.Name,
				Image: containerGroup.Container.Image,
				Ready: utils.GetPodPhaseFromContainerGroupState(containerGroup.CurrentState) == corev1.PodRunning,
			},
		},
	}, nil

}

func (p *SaladCloudProvider) GetPods(ctx context.Context) ([]*corev1.Pod, error) {

	resp, r, err := p.apiClient.ContainerGroupsAPI.ListContainerGroups(p.contextWithAuth(), p.inputVars.OrganizationName, p.inputVars.ProjectName).Execute()
	if err != nil {
		log.G(ctx).Errorf("Error when list ContainerGroupsAPI.ListContainerGroups ", r)
		return nil, err
	}
	pods := make([]*corev1.Pod, 0)
	for _, containerGroup := range resp.GetItems() {
		startTime := metav1.NewTime(containerGroup.CreateTime)
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  containerGroup.Name,
						Image: containerGroup.Container.Image,
					},
				},
			},
			Status: corev1.PodStatus{
				Phase:     utils.GetPodPhaseFromContainerGroupState(containerGroup.CurrentState),
				StartTime: &startTime,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:  containerGroup.Name,
						Image: containerGroup.Container.Image,
						Ready: utils.GetPodPhaseFromContainerGroupState(containerGroup.CurrentState) == corev1.PodRunning,
					},
				},
			},
		}

		pods = append(pods, pod)

	}
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

func (p *SaladCloudProvider) getContainerEnvironment(podMetadata metav1.ObjectMeta, container corev1.Container) map[string]string {
	marshallerObjectMetadata, err := json.Marshal(podMetadata)
	if err != nil {
		log.G(context.Background()).Errorf("Failed Marshalling ", err)
	}
	envMap := make(map[string]string)
	if marshallerObjectMetadata != nil {
		envMap["POD_METADATA_YAM"] = string(marshallerObjectMetadata)
	}
	for _, env := range container.Env {
		if env.ValueFrom == nil {
			ignore := false
			for _, ignoreEnv := range k8sDefaultEnvVars {
				if ignoreEnv == env.Name {
					ignore = true
					break
				}
			}
			if !ignore {
				envMap[env.Name] = env.Value
			}
		} else {
			// TODO Handle environment variable from source
			log.G(context.Background()).Debugf("Environment variable support from %s is not yet implemented", env.ValueFrom.String())
		}
	}
	return envMap
}

func (p *SaladCloudProvider) createContainersObject(pod *corev1.Pod) []saladclient.CreateContainer {
	cpu, memory := utils.GetPodResource(pod.Spec)
	creteContainersArray := make([]saladclient.CreateContainer, 0)

	for _, container := range pod.Spec.Containers {
		containerResourceRequirement := saladclient.NewContainerResourceRequirements(int32(cpu), int32(memory))
		createContainer := saladclient.NewCreateContainer(container.Image, *containerResourceRequirement)
		createContainer.SetEnvironmentVariables(p.getContainerEnvironment(pod.ObjectMeta, container))
		if container.Command != nil {
			createContainer.SetCommand(container.Command)
		}
		creteContainersArray = append(creteContainersArray, *createContainer)
		// TODO Add support for container Registry auth
	}

	return creteContainersArray

}

func (p *SaladCloudProvider) createContainerGroup(createContainerList []saladclient.CreateContainer, pod *corev1.Pod) []saladclient.CreateContainerGroup {

	createContainerGroups := make([]saladclient.CreateContainerGroup, 0)

	if pod.ObjectMeta.GetAnnotations()["countryCodes"] == "" {
		pod.ObjectMeta.SetAnnotations(map[string]string{
			"countryCodes": "US",
		})
	}

	var countryCodesEnum []saladclient.CountryCode
	for _, countryCode := range strings.Split(pod.ObjectMeta.GetAnnotations()["countryCodes"], ",") {
		countryCodeEnum := saladclient.CountryCode(countryCode)
		countryCodesEnum = append(countryCodesEnum, countryCodeEnum)
	}

	for _, container := range createContainerList {
		createContainerGroupRequest := *saladclient.NewCreateContainerGroup(utils.GetPodName(pod.Namespace, pod.Name, pod), container, "always", 1)
		createContainerGroupRequest.SetCountryCodes(countryCodesEnum)
		createContainerGroups = append(createContainerGroups, createContainerGroupRequest)
	}

	return createContainerGroups

}
