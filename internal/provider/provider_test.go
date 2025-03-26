package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	saladclient "github.com/SaladTechnologies/salad-client"
	// "github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/provider"
	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/models"
	"github.com/virtual-kubelet/virtual-kubelet/node/nodeutil"
)

// from cmd/virtual-kubelet-saladcloud/main.go
func defaultInputs() models.InputVars {
	home, err := homedir.Dir()
	kubeConfig := os.Getenv("KUBECONFIG")
	if err == nil && home != "" {
		kubeConfig = filepath.Join(home, ".kube", "config")
	}

	return models.InputVars{
		NodeName:         "saladcloud-node",
		KubeConfig:       kubeConfig,
		LogLevel:         "info",
		OrganizationName: "",
		TaintKey:         "virtual-kubelet.io/provider",
		TaintEffect:      "NoSchedule",
		TaintValue:       "saladcloud",
		ProjectName:      "",
		ApiKey:           "",
	}
}

func newProvider() (*SaladCloudProvider, error) {
	ctx := context.Background()
	inputs := defaultInputs()
	pc := nodeutil.ProviderConfig{}
	return NewSaladCloudProvider(ctx, inputs, pc)
}

func Test_getCountryCodes(t *testing.T) {
	p, _ := newProvider()

	// No CC in Annotations
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}
	expectCC := &[]saladclient.CountryCode{}

	cc, err := p.getCountryCodes(pod)
	assert.Nil(t, err)
	assert.NotNil(t, cc)
	assert.Equal(t, *expectCC, cc)

	// One CC in Annotations
	pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"salad.com/country-codes": "mx",
			},
		},
	}
	expectCC = &[]saladclient.CountryCode{
		saladclient.CountryCode("mx"),
	}

	cc, err = p.getCountryCodes(pod)
	assert.Nil(t, err)
	assert.NotNil(t, cc)
	assert.Equal(t, *expectCC, cc)

	// Multiple CC in Annotations
	pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"salad.com/country-codes": "mx,ca",
			},
		},
	}
	expectCC = &[]saladclient.CountryCode{
		saladclient.CountryCode("mx"),
		saladclient.CountryCode("ca"),
	}

	cc, err = p.getCountryCodes(pod)
	assert.Nil(t, err)
	assert.NotNil(t, cc)
	assert.Equal(t, *expectCC, cc)

	// One upper-case CC in Annotations
	pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"salad.com/country-codes": "MX",
			},
		},
	}
	expectCC = &[]saladclient.CountryCode{
		saladclient.CountryCode("mx"),
	}

	cc, err = p.getCountryCodes(pod)
	assert.Nil(t, err)
	assert.NotNil(t, cc)
	assert.Equal(t, *expectCC, cc)

	// Multiple mixed case CC in Annotations
	pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"salad.com/country-codes": "MX,ca",
			},
		},
	}
	expectCC = &[]saladclient.CountryCode{
		saladclient.CountryCode("mx"),
		saladclient.CountryCode("ca"),
	}

	cc, err = p.getCountryCodes(pod)
	assert.Nil(t, err)
	assert.NotNil(t, cc)
	assert.Equal(t, *expectCC, cc)

	// Invalid CC in Annotations
	pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"salad.com/country-codes": "XX",
			},
		},
	}
	expectCC = &[]saladclient.CountryCode{}

	cc, err = p.getCountryCodes(pod)
	assert.NotNil(t, err)
	assert.NotNil(t, cc)
	assert.Equal(t, *expectCC, cc)

}
