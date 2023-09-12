package models

type InputVars struct {
	NodeName         string
	KubeConfig       string
	DisableTaint     bool
	LogLevel         string
	TaintKey         string
	TaintEffect      string
	TaintValue       string
	OrganizationName string
	ProjectName      string
	ApiKey           string
}

type CreateContainerGroupModel struct {
	Name           string            `json:"name"`
	Container      ContainerSpec     `json:"container"`
	RestartPolicy  string            `json:"restart_policy"`
	LivenessProbe  Probe             `json:"liveness_probe"`
	ReadinessProbe Probe             `json:"readiness_probe"`
	StartupProbe   Probe             `json:"startup_probe"`
	Annotations    map[string]string `json:"annotations"`
}

type ContainerSpec struct {
	Resources            Resources         `json:"resources"`
	EnvironmentVariables map[string]string `json:"environment_variables"`
	Image                string            `json:"image"`
	Command              []string          `json:"command"`
}

type Resources struct {
	Requests ResourceList `json:"requests"`
	Limits   ResourceList `json:"limits"`
}

type ResourceList struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type Probe struct {
	HTTPGetAction       *HTTPGetAction `json:"httpGet,omitempty"`
	InitialDelaySeconds int32          `json:"initialDelaySeconds,omitempty"`
	PeriodSeconds       int32          `json:"periodSeconds,omitempty"`
	TimeoutSeconds      int32          `json:"timeoutSeconds,omitempty"`
	SuccessThreshold    int32          `json:"successThreshold,omitempty"`
	FailureThreshold    int32          `json:"failureThreshold,omitempty"`
}

type HTTPGetAction struct {
	Path   string `json:"path"`
	Port   int32  `json:"port"`
	Scheme string `json:"scheme"`
}
