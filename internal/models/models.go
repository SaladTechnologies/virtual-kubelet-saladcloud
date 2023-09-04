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
}
