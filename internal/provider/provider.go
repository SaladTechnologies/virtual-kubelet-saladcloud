package provider

import (
	"context"
	"io"

	dto "github.com/prometheus/client_model/go"
	nodeapi "github.com/virtual-kubelet/virtual-kubelet/node/api"
	"github.com/virtual-kubelet/virtual-kubelet/node/api/statsv1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type SaladCloudProvider struct {
}

func NewSaladCloudProvider(ctx context.Context) (*SaladCloudProvider, error) {
	return &SaladCloudProvider{}, nil
}

func (p *SaladCloudProvider) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	return nil
}

func (p *SaladCloudProvider) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	return nil
}

func (p *SaladCloudProvider) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	return nil
}

func (p *SaladCloudProvider) GetPod(ctx context.Context, namespace string, name string) (*corev1.Pod, error) {
	return nil, nil
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
