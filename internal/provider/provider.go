package provider

import (
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/models"
	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/utils"
	"github.com/google/uuid"
	saladclient "github.com/lucklypriyansh-2/salad-client"
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
}

const (
	defaultCPUCoresNumber  = "10000"
	defaultMemorySize      = "1Ti"
	defaultStorageSize     = "1Ti"
	defaultPodsLimit       = "1000"
	defaultOperatingSystem = "Linux"
)

func NewSaladCloudProvider(ctx context.Context, inputVars models.InputVars, providerConfig nodeutil.ProviderConfig) (*SaladCloudProvider, error) {
	cloudProvider := &SaladCloudProvider{
		inputVars: inputVars,
		apiClient: saladclient.NewAPIClient(saladclient.NewConfiguration()),
		logger:    log.G(ctx),
		podLister: providerConfig.Pods,
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
	ctx, span := trace.StartSpan(ctx, "CreatePod")
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
		if r.StatusCode == 400 {
			if *pd.Type.Get() == "name_conflict" {
				// The exciting duplicate name condition!
				p.logger.Errorf("Name %s has already been used in provider project %s/%s", pod.Name, p.inputVars.OrganizationName, p.inputVars.ProjectName)
			} else {
				p.logger.Errorf("Error type %s in `ContainerGroupsAPI.CreateContainerGroupModel`", *pd.Type.Get())
			}
		} else {
			p.logger.Errorf("Error when calling `ContainerGroupsAPI.CreateContainerGroupModel`", r)
		}
		return err
	}

	// wait for 3 second
	time.Sleep(3 * time.Second)

	startHttpResponse, err := p.apiClient.ContainerGroupsAPI.StartContainerGroup(p.contextWithAuth(), p.inputVars.OrganizationName, p.inputVars.ProjectName, utils.GetPodName(pod.Namespace, pod.Name, nil)).Execute()
	if err != nil {
		// Get response body for error info
		pd, err := utils.GetResponseBody(startHttpResponse)
		if err != nil {
			p.logger.Errorf("`ContainerGroupsAPI.StartContainerGroup`: %s", err)
			return err
		}

		p.logger.Errorf("`ContainerGroupsAPI.StartContainerGroup`: Error: %+v", *pd)
		err = p.DeletePod(ctx, pod)
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

	p.logger.Infof("Container %s created and initialized", pod.Name)
	return nil
}

func (p *SaladCloudProvider) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
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

func (p *SaladCloudProvider) GetPod(ctx context.Context, namespace string, name string) (*corev1.Pod, error) {
	podname := utils.GetPodName(namespace, name, nil)
	resp, r, err := saladclient.NewAPIClient(saladclient.NewConfiguration()).ContainerGroupsAPI.GetContainerGroup(p.contextWithAuth(), p.inputVars.OrganizationName, p.inputVars.ProjectName, podname).Execute()
	if err != nil {
		// Get response body for error info
		pd, err := utils.GetResponseBody(r)
		if err != nil {
			p.logger.Errorf("`ContainerGroupsAPI.GetPod`: %s", err)
			return nil, err
		}

		if r.StatusCode == 404 {
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
	containerGroup, response, err := p.apiClient.ContainerGroupsAPI.GetContainerGroup(p.contextWithAuth(), p.inputVars.OrganizationName, p.inputVars.ProjectName, podname).Execute()
	if err != nil {
		// Get response body for error info
		pd, err := utils.GetResponseBody(response)
		if err != nil {
			p.logger.Errorf("GetPodStatus: %s", err)
			return nil, err
		}

		if response.StatusCode == 404 {
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

func (p *SaladCloudProvider) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts nodeapi.ContainerLogOpts) (io.ReadCloser, error) {
	return nil, nil
}

func (p *SaladCloudProvider) RunInContainer(ctx context.Context, namespace, podName, containerName string, cmd []string, attach nodeapi.AttachIO) error {
	return nil
}

func (p *SaladCloudProvider) AttachToContainer(ctx context.Context, namespace, podName, containerName string, attach nodeapi.AttachIO) error {
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
	creteContainersArray := make([]saladclient.CreateContainer, 0)
	for _, container := range pod.Spec.Containers {
		containerResourceRequirement := saladclient.NewContainerResourceRequirements(int32(cpu), int32(memory))
		createContainer := saladclient.NewCreateContainer(container.Image, *containerResourceRequirement)

		createContainer.SetEnvironmentVariables(p.getContainerEnvironment(pod.ObjectMeta, container))
		if container.Command != nil {
			createContainer.SetCommand(container.Command)
		}
		gpuClasses, err := p.getGPUClasses(pod)
		if err == nil && gpuClasses != nil && len(gpuClasses) > 0 {
			createContainer.Resources.SetGpuClasses(gpuClasses)
		}
		logging := p.getContainerLogging(pod)
		if logging != nil {
			createContainer.Logging.Set(logging)
		}
		creteContainersArray = append(creteContainersArray, *createContainer)
	}
	return creteContainersArray
}

func (p *SaladCloudProvider) createContainerGroup(createContainerList []saladclient.CreateContainer, pod *corev1.Pod) []saladclient.CreateContainerGroup {
	createContainerGroups := make([]saladclient.CreateContainerGroup, 0)
	for _, container := range createContainerList {
		createContainerGroupRequest := *saladclient.NewCreateContainerGroup(utils.GetPodName(pod.Namespace, pod.Name, pod), container, "always", 1)
		readinessProbe, err := p.getWorkloadContainerProbeFrom(pod.Spec.Containers[0].ReadinessProbe)
		if err == nil && readinessProbe != nil {
			createContainerGroupRequest.ReadinessProbe = *readinessProbe
		} else {
			log.G(context.Background()).Errorf("Failed to get readinessProbe ", err)
		}
		livenessProbe, err := p.getWorkloadContainerProbeFrom(pod.Spec.Containers[0].LivenessProbe)
		if err == nil && livenessProbe != nil {
			createContainerGroupRequest.LivenessProbe = *livenessProbe
		} else {
			log.G(context.Background()).Errorf("Failed to get livenessProbe ", err)
		}
		startupProbe, err := p.getWorkloadContainerProbeFrom(pod.Spec.Containers[0].StartupProbe)
		if err == nil && startupProbe != nil {
			createContainerGroupRequest.StartupProbe = *startupProbe
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

func (p *SaladCloudProvider) getWorkloadContainerProbeFrom(k8sProbe *corev1.Probe) (*saladclient.NullableContainerGroupProbe, error) {
	if k8sProbe == nil || *k8sProbe == (corev1.Probe{}) {
		return nil, nil
	}
	probe := saladclient.NewContainerGroupProbe(k8sProbe.InitialDelaySeconds, k8sProbe.PeriodSeconds, k8sProbe.TimeoutSeconds, k8sProbe.SuccessThreshold, k8sProbe.FailureThreshold)
	if k8sProbe.GRPC != nil {
		grpcProbe := saladclient.NewContainerGroupProbeGrpc(*k8sProbe.GRPC.Service, k8sProbe.GRPC.Port)
		probe.SetGrpc(*grpcProbe)
	}
	if k8sProbe.HTTPGet != nil {
		httpProbe := saladclient.NewContainerGroupProbeHttp(k8sProbe.HTTPGet.Path, int32(k8sProbe.HTTPGet.Port.IntValue()))
		for _, header := range k8sProbe.HTTPGet.HTTPHeaders {
			httpProbe.Headers = append(httpProbe.Headers, saladclient.HttpHeadersInner{Name: header.Name, Value: header.Value})
		}
		probe.SetHttp(*httpProbe)
	}

	if k8sProbe.TCPSocket != nil {
		tcpProbe := saladclient.NewContainerGroupProbeTcp(int32(k8sProbe.TCPSocket.Port.IntValue()))
		probe.SetTcp(*tcpProbe)
	}
	if k8sProbe.Exec != nil {
		exec := saladclient.NewContainerGroupProbeExec(k8sProbe.Exec.Command)
		probe.SetExec(*exec)
	}
	return saladclient.NewNullableContainerGroupProbe(probe), nil
}

func (p *SaladCloudProvider) getGPUClasses(pod *corev1.Pod) ([]string, error) {
	gpuRequestedString, ok := pod.ObjectMeta.Annotations["salad.com/gpu-classes"]
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
	countryCodes = append(countryCodes, "US")
	countryCodesFromAnnotation, ok := pod.ObjectMeta.Annotations["salad.com/country-codes"]
	if !ok {
		return countryCodes, nil
	}
	codes := strings.Split(countryCodesFromAnnotation, ",")
	for _, code := range codes {
		cc, err := saladclient.NewCountryCodeFromValue(code)
		if err != nil {
			return []saladclient.CountryCode{}, errors.WithMessage(err, "Invalid country code provided: "+code)
		}
		countryCodes = append(countryCodes, *cc)
	}
	if len(countryCodes) == 0 {
		countryCodes = append(countryCodes, "us")
	}
	return countryCodes, nil
}

func (p *SaladCloudProvider) getNetworking(pod *corev1.Pod) (*saladclient.CreateContainerGroupNetworking, error) {
	protocol, hasProtocol := pod.ObjectMeta.Annotations["salad.com/networking-protocol"]
	port, hasPort := pod.ObjectMeta.Annotations["salad.com/networking-port"]
	auth, hasAuth := pod.ObjectMeta.Annotations["salad.com/networking-auth"]
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
	parsedAuth := false
	if strings.ToLower(auth) == "true" {
		parsedAuth = true
	}
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
	newRelicHost, hasRelicHost := pod.ObjectMeta.Annotations["salad.com/logging-new-relic-host"]
	newRelicIngestionKey, hasRelicIngestionKey := pod.ObjectMeta.Annotations["salad.com/logging-new-relic-ingestion-key"]

	splunkHost, hasSplunkHost := pod.ObjectMeta.Annotations["salad.com/logging-splunk-host"]
	splunkToken, hasSplunkToken := pod.ObjectMeta.Annotations["salad.com/logging-splunk-token"]

	tcpHost, hasTCPHost := pod.ObjectMeta.Annotations["salad.com/logging-tcp-host"]
	tcpPort, hasTCPPort := pod.ObjectMeta.Annotations["salad.com/logging-tcp-port"]

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
