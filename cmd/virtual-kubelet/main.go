package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/models"
	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/provider"
	"github.com/mitchellh/go-homedir"
	logrus "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	logruslogger "github.com/virtual-kubelet/virtual-kubelet/log/logrus"
	v1 "k8s.io/api/core/v1"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/virtual-kubelet/virtual-kubelet/node"
	"github.com/virtual-kubelet/virtual-kubelet/node/nodeutil"
)

var (
	binaryFilename = filepath.Base(os.Args[0])
	description    = fmt.Sprintf("%s implements a node on a Kubernetes cluster using StackPath Workload API to run pods.", binaryFilename)
)

var Inputs = models.InputVars{
	NodeName:         "saladcloud-edge-provider",
	KubeConfig:       os.Getenv("KUBECONFIG"),
	LogLevel:         "info",
	OrganizationName: "organizationName_example",
	TaintKey:         "virtual-kubelet.io/provider",
	TaintEffect:      string(v1.TaintEffectNoSchedule),
	TaintValue:       "saladCloud",
	ProjectName:      "projectName_example",
	ApiKey:           "apiKey_example",
}

func init() {
	home, _ := homedir.Dir()
	if home != "" {
		Inputs.KubeConfig = filepath.Join(home, ".kube", "config")
	}

	virtualKubeletCommand.Flags().StringVar(&Inputs.NodeName, "nodename", Inputs.NodeName, "kubernetes node name")
	virtualKubeletCommand.Flags().StringVar(&Inputs.KubeConfig, "kube-config", Inputs.KubeConfig, "kubeconfig file")
	virtualKubeletCommand.Flags().StringVar(&Inputs.OrganizationName, "organizationName", Inputs.OrganizationName, "Organization Name To Be used by Salad Client")
	virtualKubeletCommand.Flags().StringVar(&Inputs.ApiKey, "api-key", Inputs.ApiKey, "Api Key for the Salad Client")
	requiredFlagError := virtualKubeletCommand.MarkFlagRequired("organizationName")
	if requiredFlagError != nil {
		logrus.WithError(requiredFlagError).Fatal("Error marking organizationName as required")
	}
	virtualKubeletCommand.Flags().StringVar(&Inputs.ProjectName, "projectName", Inputs.ProjectName, "project Name to be used by Salad Client")
	requiredFlagError = virtualKubeletCommand.MarkFlagRequired("projectName")
	if requiredFlagError != nil {
		logrus.WithError(requiredFlagError).Fatal("Error marking projectName as required")
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := virtualKubeletCommand.ExecuteContext(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			logrus.WithError(err).Fatal("error running command")
		}
	}

}

func runNode(ctx context.Context) error {

	Inputs.NodeName = fmt.Sprintf("%s-%s", Inputs.NodeName, randSeq(3))

	node, err := nodeutil.NewNode(Inputs.NodeName, func(pc nodeutil.ProviderConfig) (nodeutil.Provider, node.NodeProvider, error) {
		p, err := provider.NewSaladCloudProvider(context.Background(), Inputs)
		if err != nil {
			return nil, nil, err
		}
		p.ConfigureNode(ctx, pc.Node)
		return p, nil, nil
	}, withClient, withTaint)
	if err != nil {
		log.G(ctx).Fatal(err)
	}

	go func() {
		if err := node.Run(ctx); err != nil {
			log.G(ctx).Fatal(err)
		}
	}()

	err = node.WaitReady(ctx, 0)
	if err != nil {
		log.G(ctx).Fatal(err)
	}

	<-node.Done()
	err = node.Err()
	if err != nil {
		log.G(ctx).Fatal(err)
	}
	return err

}

var virtualKubeletCommand = &cobra.Command{
	Use:   binaryFilename,
	Short: description,
	Long:  description,
	Run: func(cmd *cobra.Command, args []string) {

		logger := logrus.StandardLogger()
		logLevel, err := logrus.ParseLevel(Inputs.LogLevel)

		if err != nil {
			logrus.WithError(err).Fatal("Error parsing log level")
		}
		logger.SetLevel(logLevel)

		ctx := log.WithLogger(cmd.Context(), logruslogger.FromLogrus(logrus.NewEntry(logger)))

		if err := runNode(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.G(ctx).Fatal(err)
			} else {
				log.G(ctx).Debug(err)
			}
		}
	},
}

// withTaint sets up the taint for the node
func withTaint(cfg *nodeutil.NodeConfig) error {
	if Inputs.DisableTaint {
		return nil
	}

	taint := v1.Taint{
		Key:   Inputs.TaintKey,
		Value: Inputs.TaintValue,
	}
	switch Inputs.TaintEffect {
	case string(v1.TaintEffectNoSchedule):
		taint.Effect = v1.TaintEffectNoSchedule
	case string(v1.TaintEffectNoExecute):
		taint.Effect = v1.TaintEffectNoExecute
	case string(v1.TaintEffectPreferNoSchedule):
		taint.Effect = v1.TaintEffectPreferNoSchedule
	default:
		return errdefs.InvalidInputf("taint effect %q is not supported", Inputs.TaintEffect)
	}
	cfg.NodeSpec.Spec.Taints = append(cfg.NodeSpec.Spec.Taints, taint)
	return nil
}

func withClient(cfg *nodeutil.NodeConfig) error {
	client, err := nodeutil.ClientsetFromEnv(Inputs.KubeConfig)
	if err != nil {
		return err
	}
	return nodeutil.WithClient(client)(cfg)

}

var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyz")

// https://stackoverflow.com/a/22892986/2709066
func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
