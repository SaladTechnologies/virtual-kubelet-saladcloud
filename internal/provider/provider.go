package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	saladclient "github.com/SaladTechnologies/salad-client"
	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/models"
	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/utils"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	nodeapi "github.com/virtual-kubelet/virtual-kubelet/node/api"
	"github.com/virtual-kubelet/virtual-kubelet/node/nodeutil"
	"github.com/virtual-kubelet/virtual-kubelet/trace"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
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
	logger          log.Logger
	podsTracker     *PodsTracker
	podLister       corev1listers.PodLister
	secretLister    corev1listers.SecretLister
}

const (
	// Set defaults to allow ...
	defaultPodsLimit = "1000"
	// pods at per pod usage of...
	// 16 vCPU
	defaultCPUCoresNumber = "16000"
	// 60G memory
	defaultMemorySize = "60Ti"
	// 50G storage
	defaultStorageSize = "50Ti"

	defaultOperatingSystem = "Linux"
)

func NewSaladCloudProvider(ctx context.Context, inputVars models.InputVars, providerConfig nodeutil.ProviderConfig) (*SaladCloudProvider, error) {
	cloudProvider := &SaladCloudProvider{
		inputVars:    inputVars,
		apiClient:    saladclient.NewAPIClient(saladclient.NewConfiguration()),
		logger:       log.G(ctx),
		podLister:    providerConfig.Pods,
		secretLister: providerConfig.Secrets,
	}
	cloudProvider.setNodeCapacity()

	return cloudProvider, nil
}

func (p *SaladCloudProvider) setNodeCapacity() {
	p.cpu = defaultCPUCoresNumber
	p.memory = defaultMemorySize
	p.pods = defaultPodsLimit
	p.storage = defaultStorageSize
	p.operatingSystem = defaultOperatingSystem
}

func (p *SaladCloudProvider) ConfigureNode(_ context.Context, node *corev1.Node) {
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

func (p *SaladCloudProvider) NotifyPods(ctx context.Context, notifierCallback func(*corev1.Pod)) {
	p.logger.Debug("Notify pods set")
	p.podsTracker = &PodsTracker{
		podLister:      p.podLister,
		updateCallback: notifierCallback,
		handler:        p,
		ctx:            ctx,
		logger:         p.logger,
	}
	go p.podsTracker.BeginPodTracking(ctx)
}

func (p *SaladCloudProvider) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	_, span := trace.StartSpan(ctx, "CreatePod")
	defer span.End()
	p.logger.Infof("CreatePod: %s", pod.Name)
	createContainerObject := p.createContainersObject(pod)
	p.logger.Debugf(" createContainerObject: %+v", createContainerObject)
	createContainerGroup := p.createContainerGroup(createContainerObject, pod)
	p.logger.Debugf(" createContainerGroup: %+v", createContainerGroup[0])

	_, r, err := p.apiClient.
		ContainerGroupsAPI.CreateContainerGroup(
		p.contextWithAuth(),
		p.inputVars.OrganizationName,
		p.inputVars.ProjectName).CreateContainerGroup(
		createContainerGroup[0],
	).Execute()
	if err != nil {
		// Get response body for error info
		pd, err := utils.GetResponseBody(r)
		if err != nil {
			p.logger.Errorf("CreatePod: %s", err)
			return err
		}

		// Also handle 403 and 429?
		if r != nil && r.StatusCode == http.StatusBadRequest {
			if *pd.Type == "name_conflict" {
				// The exciting duplicate name condition!
				p.logger.Errorf("Name %s has already been used in provider project %s/%s", pod.Name, p.inputVars.OrganizationName, p.inputVars.ProjectName)
			} else {
				p.logger.Errorf("Error type %s in `ContainerGroupsAPI.CreateContainerGroupModel`", *pd.Type)
			}
		} else {
			p.logger.Errorf("Error when calling `ContainerGroupsAPI.CreateContainerGroupModel`", r)
		}
		return err
	}

	now := metav1.NewTime(time.Now())
	pod.CreationTimestamp = now
	pod.Status = corev1.PodStatus{
		Phase:     corev1.PodPending,
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

	p.logger.Infof("Container %s created and initialized", pod.Name)
	return nil
}

func (p *SaladCloudProvider) UpdatePod(_ context.Context, pod *corev1.Pod) error {
	p.logger.Debugf("UpdatePod: %s: %+v", utils.GetPodName(pod.Namespace, pod.Name, pod), pod)
	return nil
}

func (p *SaladCloudProvider) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	_, span := trace.StartSpan(ctx, "DeletePod")
	defer span.End()
	p.logger.Debugf("Deleting pod %s", utils.GetPodName(pod.Namespace, pod.Name, pod))
	response, err := p.apiClient.ContainerGroupsAPI.DeleteContainerGroup(p.contextWithAuth(), p.inputVars.OrganizationName, p.inputVars.ProjectName, utils.GetPodName(pod.Namespace, pod.Name, pod)).Execute()
	pod.Status.Phase = corev1.PodSucceeded
	pod.Status.Reason = "Pod Deleted"
	if err != nil {
		// Get response body for error info
		pd, err := utils.GetResponseBody(response)
		if err != nil {
			p.logger.Errorf("`ContainerGroupsAPI.DeletePod`: %s", err)
			return err
		}

		p.logger.Errorf("`ContainerGroupsAPI.DeletePod`: Error: %+v", *pd)
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
		p.logger.Infof("Container %s deleted", pod.Status.ContainerStatuses[idx].Name)
	}
	return nil
}

func (p *SaladCloudProvider) GetPod(_ context.Context, namespace string, name string) (*corev1.Pod, error) {
	podname := utils.GetPodName(namespace, name, nil)
	resp, r, err := saladclient.NewAPIClient(saladclient.NewConfiguration()).ContainerGroupsAPI.GetContainerGroup(p.contextWithAuth(), p.inputVars.OrganizationName, p.inputVars.ProjectName, podname).Execute()
	if err != nil {
		// Get response body for error info
		pd, err := utils.GetResponseBody(r)
		if err != nil {
			p.logger.Errorf("`ContainerGroupsAPI.GetPod`: %s", err)
			return nil, err
		}

		if r != nil && r.StatusCode == http.StatusNotFound {
			p.logger.Warnf("`ContainerGroupsAPI.GetPod`: %s not found", podname)
		} else {
			p.logger.Errorf("`ContainerGroupsAPI.GetPod`: Error: %+v", *pd)
		}
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
	_, span := trace.StartSpan(ctx, "GetPodStatus")
	defer span.End()

	podname := utils.GetPodName(namespace, name, nil)
	containerGroup, response, err := p.apiClient.ContainerGroupsAPI.
		GetContainerGroup(p.contextWithAuth(), p.inputVars.OrganizationName, p.inputVars.ProjectName, podname).
		Execute()
	if err != nil {
		// Get response body for error info
		pd, err := utils.GetResponseBody(response)
		if err != nil {
			p.logger.Errorf("GetPodStatus: %s", err)
			return nil, err
		}

		if response != nil && response.StatusCode == http.StatusNotFound {
			p.logger.WithField("namespace", namespace).
				WithField("name", podname).
				Warnf("Not Found")
		} else {
			p.logger.WithField("namespace", namespace).
				WithField("name", name).
				Errorf("ContainerGroupsAPI.GetPodStatus: %+v ", *pd)
		}
		return nil, models.NewSaladCloudError(err, response)
	}

	phase := utils.GetPodPhaseFromContainerGroupState(containerGroup.CurrentState)
	ready := containerGroup.CurrentState.Status == saladclient.CONTAINERGROUPSTATUS_RUNNING &&
		containerGroup.CurrentState.InstanceStatusCounts.RunningCount > 0
	p.logger.Infof("Pod %s computed status - Phase: %v, Ready: %v, Status: %v, RunningCount: %d",
		podname, phase, ready, containerGroup.CurrentState.Status, containerGroup.CurrentState.InstanceStatusCounts.RunningCount)

	startTime := metav1.NewTime(containerGroup.CreateTime)
	return &corev1.PodStatus{
		Phase:     phase,
		StartTime: &startTime,
		Conditions: []corev1.PodCondition{
			{Type: corev1.PodReady, Status: getConditionStatus(ready)},
			{Type: corev1.ContainersReady, Status: getConditionStatus(ready)},
		},
		ContainerStatuses: []corev1.ContainerStatus{
			{
				Name:  containerGroup.Name,
				Image: containerGroup.Container.Image,
				Ready: ready,
				State: getContainerState(containerGroup.CurrentState),
			},
		},
	}, nil
}

func (p *SaladCloudProvider) GetPods(_ context.Context) ([]*corev1.Pod, error) {
	resp, r, err := p.apiClient.ContainerGroupsAPI.ListContainerGroups(p.contextWithAuth(), p.inputVars.OrganizationName, p.inputVars.ProjectName).Execute()
	if err != nil {
		// Get response body for error info
		pd, err := utils.GetResponseBody(r)
		if err != nil {
			p.logger.Errorf("GetPods: %s", err)
			return nil, err
		}

		p.logger.Errorf("`ContainerGroupsAPI.GetPods`: Error: %+v", *pd)
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

func (p *SaladCloudProvider) GetContainerLogs(_ context.Context, namespace, podName, containerName string, opts nodeapi.ContainerLogOpts) (io.ReadCloser, error) {
	return nil, nil
}

func (p *SaladCloudProvider) RunInContainer(ctx context.Context, namespace, podName, containerName string, cmd []string, attach nodeapi.AttachIO) error {
	return nil
}

func (p *SaladCloudProvider) AttachToContainer(_ context.Context, namespace, podName, containerName string, attach nodeapi.AttachIO) error {
	return nil
}

func (p *SaladCloudProvider) GetStatsSummary(context.Context) (*stats.Summary, error) {
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
	createContainersArray := make([]saladclient.CreateContainer, 0)
	for _, container := range pod.Spec.Containers {
		containerResourceRequirement := saladclient.NewContainerResourceRequirements(int32(cpu), int32(memory))
		createContainer := saladclient.NewCreateContainer(container.Image, *containerResourceRequirement)

		createContainer.SetEnvironmentVariables(p.getContainerEnvironment(pod.ObjectMeta, container))
		if container.Command != nil {
			createContainer.SetCommand(container.Command)
		}

		// Handle image pull secrets
		if ips, err := p.getImagePullSecrets(pod); err != nil {
			p.logger.Errorf("Error getting image pull secrets: %v", err)
		} else if len(ips) > 0 {
			// SaladCloud currently supports one registry auth per container
			auth := saladclient.CreateContainerRegistryAuthentication{
				Basic: saladclient.NewCreateContainerRegistryAuthenticationBasic(ips[0].Username, ips[0].Password),
			}
			createContainer.RegistryAuthentication = &auth
		}

		gpuClasses, err := p.getGPUClasses(pod)
		if err == nil && gpuClasses != nil && len(gpuClasses) > 0 {
			createContainer.Resources.SetGpuClasses(gpuClasses)
		}
		logging := p.getContainerLogging(pod)
		if logging != nil {
			createContainer.Logging = logging
		}
		priority, err := p.getContainerPriority(pod)
		if err == nil && priority != nil {
			createContainer.Priority.Set(priority)
		}
		createContainersArray = append(createContainersArray, *createContainer)
	}
	return createContainersArray
}

func (p *SaladCloudProvider) getWorkloadContainerLivenessProbeFrom(
	k8sProbe *corev1.Probe,
) (*saladclient.ContainerGroupLivenessProbe, error) {

	if k8sProbe == nil || *k8sProbe == (corev1.Probe{}) {
		// No probe specified.
		return nil, nil
	}

	// Create the typed LivenessProbe:
	probe := saladclient.NewContainerGroupLivenessProbe(
		k8sProbe.InitialDelaySeconds,
		k8sProbe.PeriodSeconds,
		k8sProbe.TimeoutSeconds,
		k8sProbe.SuccessThreshold,
		k8sProbe.FailureThreshold,
	)

	// Fill in gRPC details if present:
	if k8sProbe.GRPC != nil {
		grpcProbe := saladclient.NewContainerGroupProbeGrpc(*k8sProbe.GRPC.Service, k8sProbe.GRPC.Port)
		probe.SetGrpc(*grpcProbe)
	}

	// Fill in HTTP details if present:
	if k8sProbe.HTTPGet != nil {
		httpProbe := saladclient.NewContainerGroupProbeHttp(
			k8sProbe.HTTPGet.Path,
			int32(k8sProbe.HTTPGet.Port.IntValue()),
		)
		for _, header := range k8sProbe.HTTPGet.HTTPHeaders {
			httpProbe.Headers = append(httpProbe.Headers,
				saladclient.HttpHeadersInner{Name: header.Name, Value: header.Value})
		}
		probe.SetHttp(*httpProbe)
	}

	// Fill in TCP details if present:
	if k8sProbe.TCPSocket != nil {
		tcpProbe := saladclient.NewContainerGroupProbeTcp(
			int32(k8sProbe.TCPSocket.Port.IntValue()),
		)
		probe.SetTcp(*tcpProbe)
	}

	// Fill in Exec details if present:
	if k8sProbe.Exec != nil {
		execProbe := saladclient.NewContainerGroupProbeExec(k8sProbe.Exec.Command)
		probe.SetExec(*execProbe)
	}

	// Wrap in nullable and return
	return probe, nil
}

func (p *SaladCloudProvider) getWorkloadContainerReadinessProbeFrom(
	k8sProbe *corev1.Probe,
) (*saladclient.ContainerGroupReadinessProbe, error) {

	if k8sProbe == nil || *k8sProbe == (corev1.Probe{}) {
		return nil, nil
	}

	// Create the typed ReadinessProbe:
	probe := saladclient.NewContainerGroupReadinessProbe(
		k8sProbe.InitialDelaySeconds,
		k8sProbe.PeriodSeconds,
		k8sProbe.TimeoutSeconds,
		k8sProbe.SuccessThreshold,
		k8sProbe.FailureThreshold,
	)

	// Optional gRPC:
	if k8sProbe.GRPC != nil {
		grpcProbe := saladclient.NewContainerGroupProbeGrpc(*k8sProbe.GRPC.Service, k8sProbe.GRPC.Port)
		probe.SetGrpc(*grpcProbe)
	}

	// Optional HTTP:
	if k8sProbe.HTTPGet != nil {
		httpProbe := saladclient.NewContainerGroupProbeHttp(
			k8sProbe.HTTPGet.Path,
			int32(k8sProbe.HTTPGet.Port.IntValue()),
		)
		for _, header := range k8sProbe.HTTPGet.HTTPHeaders {
			httpProbe.Headers = append(httpProbe.Headers,
				saladclient.HttpHeadersInner{Name: header.Name, Value: header.Value})
		}
		probe.SetHttp(*httpProbe)
	}

	// Optional TCP:
	if k8sProbe.TCPSocket != nil {
		tcpProbe := saladclient.NewContainerGroupProbeTcp(
			int32(k8sProbe.TCPSocket.Port.IntValue()),
		)
		probe.SetTcp(*tcpProbe)
	}

	// Optional Exec:
	if k8sProbe.Exec != nil {
		execProbe := saladclient.NewContainerGroupProbeExec(k8sProbe.Exec.Command)
		probe.SetExec(*execProbe)
	}

	return probe, nil
}

func (p *SaladCloudProvider) getWorkloadContainerStartupProbeFrom(
	k8sProbe *corev1.Probe,
) (*saladclient.ContainerGroupStartupProbe, error) {

	if k8sProbe == nil || *k8sProbe == (corev1.Probe{}) {
		return nil, nil
	}

	// Create the typed StartupProbe:
	probe := saladclient.NewContainerGroupStartupProbe(
		k8sProbe.InitialDelaySeconds,
		k8sProbe.PeriodSeconds,
		k8sProbe.TimeoutSeconds,
		k8sProbe.SuccessThreshold,
		k8sProbe.FailureThreshold,
	)

	// gRPC:
	if k8sProbe.GRPC != nil {
		grpcProbe := saladclient.NewContainerGroupProbeGrpc(*k8sProbe.GRPC.Service, k8sProbe.GRPC.Port)
		probe.SetGrpc(*grpcProbe)
	}

	// HTTP:
	if k8sProbe.HTTPGet != nil {
		httpProbe := saladclient.NewContainerGroupProbeHttp(
			k8sProbe.HTTPGet.Path,
			int32(k8sProbe.HTTPGet.Port.IntValue()),
		)
		for _, header := range k8sProbe.HTTPGet.HTTPHeaders {
			httpProbe.Headers = append(httpProbe.Headers,
				saladclient.HttpHeadersInner{Name: header.Name, Value: header.Value})
		}
		probe.SetHttp(*httpProbe)
	}

	// TCP:
	if k8sProbe.TCPSocket != nil {
		tcpProbe := saladclient.NewContainerGroupProbeTcp(
			int32(k8sProbe.TCPSocket.Port.IntValue()),
		)
		probe.SetTcp(*tcpProbe)
	}

	// Exec:
	if k8sProbe.Exec != nil {
		execProbe := saladclient.NewContainerGroupProbeExec(k8sProbe.Exec.Command)
		probe.SetExec(*execProbe)
	}

	return probe, nil
}

func (p *SaladCloudProvider) createContainerGroup(createContainerList []saladclient.CreateContainer, pod *corev1.Pod) []saladclient.CreateContainerGroup {
	createContainerGroups := make([]saladclient.CreateContainerGroup, 0)
	for _, container := range createContainerList {
		createContainerGroupRequest := *saladclient.NewCreateContainerGroup(
			utils.GetPodName(pod.Namespace, pod.Name, pod),
			container,
			true,
			saladclient.CONTAINERRESTARTPOLICY_ALWAYS,
			int32(1),
		)
		readinessProbe, err := p.getWorkloadContainerReadinessProbeFrom(pod.Spec.Containers[0].ReadinessProbe)
		if err == nil && readinessProbe != nil {
			createContainerGroupRequest.ReadinessProbe = readinessProbe
		} else {
			log.G(context.Background()).Errorf("Failed to get readinessProbe ", err)
		}
		livenessProbe, err := p.getWorkloadContainerLivenessProbeFrom(pod.Spec.Containers[0].LivenessProbe)
		if err == nil && livenessProbe != nil {
			createContainerGroupRequest.LivenessProbe = livenessProbe
		} else {
			log.G(context.Background()).Errorf("Failed to get livenessProbe ", err)
		}
		startupProbe, err := p.getWorkloadContainerStartupProbeFrom(pod.Spec.Containers[0].StartupProbe)
		if err == nil && startupProbe != nil {
			createContainerGroupRequest.StartupProbe = startupProbe
		} else {
			log.G(context.Background()).Errorf("Failed to get startupProbe ", err)
		}
		countryCodes, err := p.getCountryCodes(pod)
		if err != nil {
			log.G(context.Background()).Errorf("Failed to get countryCodes ", err)
		} else {
			createContainerGroupRequest.SetCountryCodes(countryCodes)
		}
		networking, err := p.getNetworking(pod)
		if err != nil {
			log.G(context.Background()).Errorf("Failed to get networking ", err)
		} else if networking != nil {
			createContainerGroupRequest.SetNetworking(*networking)
		}
		restartPolicy, err := p.getRestartPolicy(pod)
		if err != nil {
			log.G(context.Background()).Errorf("Failed to get restartPolicy ", err)
		} else {
			createContainerGroupRequest.SetRestartPolicy(*restartPolicy)
		}
		createContainerGroups = append(createContainerGroups, createContainerGroupRequest)
	}
	return createContainerGroups
}

func (p *SaladCloudProvider) getGPUClasses(pod *corev1.Pod) ([]string, error) {
	gpuRequestedString, ok := pod.Annotations["salad.com/gpu-classes"]
	if !ok {
		return nil, nil
	}
	gpuRequested := strings.Split(gpuRequestedString, ",")
	saladClientGpuIds := make([]string, 0)
	var gpuClasses *saladclient.GpuClassesList = nil

	for _, gpu := range gpuRequested {
		gpuCleaned := strings.TrimSpace(strings.ToLower(gpu))
		_, uuidErr := uuid.Parse(gpuCleaned)
		if uuidErr == nil {
			saladClientGpuIds = append(saladClientGpuIds, gpuCleaned)
		} else {
			if gpuClasses == nil {
				classes, _, err := p.apiClient.OrganizationDataAPI.ListGpuClasses(p.contextWithAuth(), p.inputVars.OrganizationName).Execute()
				if err != nil {
					log.G(context.Background()).Errorf("Failed to get gpuClasses ", err)
					return nil, err
				} else {
					gpuClasses = classes
				}
			}
			for _, gpuClass := range gpuClasses.Items {
				if strings.TrimSpace(strings.ToLower(gpuClass.Name)) == gpuCleaned {
					saladClientGpuIds = append(saladClientGpuIds, gpuClass.Id)
					break
				}
			}
		}
	}
	return saladClientGpuIds, nil
}

func (p *SaladCloudProvider) getCountryCodes(pod *corev1.Pod) ([]saladclient.CountryCode, error) {
	countryCodes := make([]saladclient.CountryCode, 0)
	countryCodesFromAnnotation, ok := pod.Annotations["salad.com/country-codes"]
	if !ok {
		// CC is optional, nothing to see here
		return countryCodes, nil
	}
	codes := strings.Split(countryCodesFromAnnotation, ",")
	for _, code := range codes {
		cc, err := saladclient.NewCountryCodeFromValue(strings.ToLower(code))
		if err != nil {
			return []saladclient.CountryCode{}, errors.WithMessage(err, "Invalid country code provided: "+code)
		}
		countryCodes = append(countryCodes, *cc)
	}
	p.countryCodes = countryCodes
	return countryCodes, nil
}

func (p *SaladCloudProvider) getNetworking(pod *corev1.Pod) (*saladclient.CreateContainerGroupNetworking, error) {
	protocol, hasProtocol := pod.Annotations["salad.com/networking-protocol"]
	port, hasPort := pod.Annotations["salad.com/networking-port"]
	auth, hasAuth := pod.Annotations["salad.com/networking-auth"]
	if !hasProtocol || !hasPort || !hasAuth {
		return nil, nil
	}
	networkingProtocol, err := saladclient.NewContainerNetworkingProtocolFromValue(protocol)
	if err != nil {
		return nil, err
	}
	parsedPortInt, err := strconv.Atoi(port)
	if err != nil {
		return nil, err
	}
	parsedAuth := strings.ToLower(auth) == "true"
	return saladclient.NewCreateContainerGroupNetworking(*networkingProtocol, int32(parsedPortInt), parsedAuth), nil
}

func (p *SaladCloudProvider) getRestartPolicy(pod *corev1.Pod) (*saladclient.ContainerRestartPolicy, error) {
	restartPolicy := "never"
	if pod.Spec.RestartPolicy == corev1.RestartPolicyAlways {
		restartPolicy = "always"
	}
	if pod.Spec.RestartPolicy == corev1.RestartPolicyOnFailure {
		restartPolicy = "on_failure"
	}
	if pod.Spec.RestartPolicy == corev1.RestartPolicyNever {
		restartPolicy = "never"
	}
	return saladclient.NewContainerRestartPolicyFromValue(restartPolicy)
}

func (p *SaladCloudProvider) getContainerLogging(pod *corev1.Pod) *saladclient.ContainerLogging {
	newRelicHost, hasRelicHost := pod.Annotations["salad.com/logging-new-relic-host"]
	newRelicIngestionKey, hasRelicIngestionKey := pod.Annotations["salad.com/logging-new-relic-ingestion-key"]

	splunkHost, hasSplunkHost := pod.Annotations["salad.com/logging-splunk-host"]
	splunkToken, hasSplunkToken := pod.Annotations["salad.com/logging-splunk-token"]

	tcpHost, hasTCPHost := pod.Annotations["salad.com/logging-tcp-host"]
	tcpPort, hasTCPPort := pod.Annotations["salad.com/logging-tcp-port"]

	if !hasRelicHost && !hasRelicIngestionKey && !hasSplunkHost && !hasSplunkToken && !hasTCPHost && !hasTCPPort {
		return nil
	}

	containerLogging := saladclient.NewContainerLogging()

	if hasRelicHost && hasRelicIngestionKey {
		newRelic := saladclient.NewContainerLoggingNewRelic(newRelicHost, newRelicIngestionKey)
		containerLogging.SetNewRelic(*newRelic)
	}

	if hasSplunkHost && hasSplunkToken {
		newSplunk := saladclient.NewContainerLoggingSplunk(splunkHost, splunkToken)
		containerLogging.SetSplunk(*newSplunk)
	}

	if hasTCPHost && hasTCPPort {
		tcpPortInt, err := strconv.Atoi(tcpPort)
		if err != nil {
			log.G(context.Background()).Errorf("Failed to convert TCP port for logging")
		} else {
			newTCP := saladclient.NewContainerLoggingTcp(tcpHost, int32(tcpPortInt))
			containerLogging.SetTcp(*newTCP)
		}
	}
	return containerLogging
}

func getConditionStatus(ready bool) corev1.ConditionStatus {
	if ready {
		return corev1.ConditionTrue
	}
	return corev1.ConditionFalse
}

func getContainerState(state saladclient.ContainerGroupState) corev1.ContainerState {
	if state.Status == saladclient.CONTAINERGROUPSTATUS_RUNNING &&
		state.InstanceStatusCounts.RunningCount > 0 {
		return corev1.ContainerState{
			Running: &corev1.ContainerStateRunning{},
		}
	}
	return corev1.ContainerState{
		Waiting: &corev1.ContainerStateWaiting{
			Reason:  cases.Title(language.English).String(string(state.Status)),
			Message: fmt.Sprintf("Container group status: %s, running count: %d", state.Status, state.InstanceStatusCounts.RunningCount),
		},
	}
}

func (p *SaladCloudProvider) getImagePullSecrets(pod *corev1.Pod) ([]saladclient.CreateContainerRegistryAuthenticationBasic, error) {
	ips := make([]saladclient.CreateContainerRegistryAuthenticationBasic, 0)

	for _, ref := range pod.Spec.ImagePullSecrets {
		secret, err := p.secretLister.Secrets(pod.Namespace).Get(ref.Name)
		if err != nil {
			return ips, err
		}

		switch secret.Type {
		case corev1.SecretTypeDockercfg:
			creds, err := p.readDockerCfgSecret(secret)
			if err != nil {
				return ips, err
			}
			ips = append(ips, creds...)
		case corev1.SecretTypeDockerConfigJson:
			creds, err := p.readDockerConfigJSONSecret(secret)
			if err != nil {
				return ips, err
			}
			ips = append(ips, creds...)
		default:
			return nil, fmt.Errorf("unsupported secret type %q for image pull secret", secret.Type)
		}
	}
	return ips, nil
}

func (p *SaladCloudProvider) readDockerCfgSecret(secret *corev1.Secret) ([]saladclient.CreateContainerRegistryAuthenticationBasic, error) {
	ips := make([]saladclient.CreateContainerRegistryAuthenticationBasic, 0)
	repoData, ok := secret.Data[corev1.DockerConfigKey]
	if !ok {
		return ips, fmt.Errorf("no dockercfg data in secret")
	}

	var authConfigs map[string]struct {
		Auth  string `json:"auth"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(repoData, &authConfigs); err != nil {
		return ips, err
	}

	for server, auth := range authConfigs {
		decoded, err := base64.StdEncoding.DecodeString(auth.Auth)
		if err != nil {
			return ips, fmt.Errorf("error decoding auth for %s: %w", server, err)
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return ips, fmt.Errorf("malformed auth for %s", server)
		}

		ips = append(ips, saladclient.CreateContainerRegistryAuthenticationBasic{
			Username: parts[0],
			Password: parts[1],
		})
	}
	return ips, nil
}

func (p *SaladCloudProvider) readDockerConfigJSONSecret(secret *corev1.Secret) ([]saladclient.CreateContainerRegistryAuthenticationBasic, error) {
	ips := make([]saladclient.CreateContainerRegistryAuthenticationBasic, 0)
	repoData, ok := secret.Data[corev1.DockerConfigJsonKey]
	if !ok {
		return ips, fmt.Errorf("no dockerconfigjson data in secret")
	}

	var config struct {
		Auths map[string]struct {
			Auth  string `json:"auth"`
			Email string `json:"email"`
		} `json:"auths"`
	}
	if err := json.Unmarshal(repoData, &config); err != nil {
		return ips, err
	}

	for server, auth := range config.Auths {
		decoded, err := base64.StdEncoding.DecodeString(auth.Auth)
		if err != nil {
			return ips, fmt.Errorf("error decoding auth for %s: %w", server, err)
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return ips, fmt.Errorf("malformed auth for %s", server)
		}

		ips = append(ips, saladclient.CreateContainerRegistryAuthenticationBasic{
			Username: parts[0],
			Password: parts[1],
		})
	}
	return ips, nil
}

func (p *SaladCloudProvider) getContainerPriority(pod *corev1.Pod) (*saladclient.ContainerGroupPriority, error) {
	priority, ok := pod.Annotations["salad.com/container-group-priority"]
	if !ok {
		return nil, nil
	}

	return saladclient.NewContainerGroupPriorityFromValue(priority)
}
