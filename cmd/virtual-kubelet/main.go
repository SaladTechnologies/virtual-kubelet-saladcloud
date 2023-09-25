package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/models"
	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/provider"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	logruslogger "github.com/virtual-kubelet/virtual-kubelet/log/logrus"
	"github.com/virtual-kubelet/virtual-kubelet/node"
	"github.com/virtual-kubelet/virtual-kubelet/node/nodeutil"
	v1 "k8s.io/api/core/v1"
)

var (
	binaryFilename     = filepath.Base(os.Args[0])
	description        = fmt.Sprintf("%s implements a node on a Kubernetes cluster using Workload API to run pods.", binaryFilename)
	inputs             = defaultInputs()
	letters            = []rune("0123456789abcdefghijklmnopqrstuvwxyz")
	taintEffectMapping = map[string]v1.TaintEffect{
		"NoSchedule":       v1.TaintEffectNoSchedule,
		"NoExecute":        v1.TaintEffectNoExecute,
		"PreferNoSchedule": v1.TaintEffectPreferNoSchedule,
	}
)

func defaultInputs() models.InputVars {
	home, err := homedir.Dir()
	kubeConfig := os.Getenv("KUBECONFIG")
	if err == nil && home != "" {
		kubeConfig = filepath.Join(home, ".kube", "config")
	}

	return models.InputVars{
		NodeName:         "saladcloud-edge-provider",
		KubeConfig:       kubeConfig,
		LogLevel:         "info",
		OrganizationName: "",
		TaintKey:         "virtual-kubelet.io/provider",
		TaintEffect:      "NoSchedule",
		TaintValue:       "saladCloud",
		ProjectName:      "",
		ApiKey:           "",
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := virtualKubeletCommand.ExecuteContext(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logrus.WithError(err).Error("Failed to execute virtual kubelet command")
		os.Exit(1)
	}
}

func initCommandFlags() {
	virtualKubeletCommand.Flags().StringVar(&inputs.NodeName, "nodename", inputs.NodeName, "Kubernetes node name")
	virtualKubeletCommand.Flags().StringVar(&inputs.KubeConfig, "kube-config", inputs.KubeConfig, "Kubeconfig file")
	virtualKubeletCommand.Flags().StringVar(&inputs.OrganizationName, "organizationName", inputs.OrganizationName, "Organization name for Salad Client")
	virtualKubeletCommand.Flags().BoolVar(&inputs.DisableTaint, "Disable taint flag", inputs.DisableTaint, "Disable the tainted effect")
	virtualKubeletCommand.Flags().StringVar(&inputs.ApiKey, "api-key", inputs.ApiKey, "API key for the Salad Client")
	virtualKubeletCommand.Flags().StringVar(&inputs.ProjectName, "projectName", inputs.ProjectName, "Project name for Salad Client")

	markFlagRequired("organizationName")
	markFlagRequired("projectName")
}

func markFlagRequired(flagName string) {
	if err := virtualKubeletCommand.MarkFlagRequired(flagName); err != nil {
		logrus.WithError(err).Errorf("Failed to mark %s as required", flagName)
		os.Exit(1)
	}
}

func runNode(ctx context.Context) error {
	logrus.Infof("Running node with name prefix: %s", inputs.NodeName)
	inputs.NodeName = fmt.Sprintf("%s-%s", inputs.NodeName, randSeq(3))

	node, err := nodeutil.NewNode(inputs.NodeName, newSaladCloudProvider, withClient, withTaint)
	if err != nil {
		logrus.WithError(err).Error("Failed to create new node")
		return err
	}

	go func() {
		if err := node.Run(ctx); err != nil {
			logrus.WithError(err).Error("Node runtime error")
		}
	}()

	if err = node.WaitReady(ctx, 0); err != nil {
		logrus.WithError(err).Error("Node readiness error")
		return err
	}

	<-node.Done()
	if err = node.Err(); err != nil {
		logrus.WithError(err).Error("Node finished with error")
		return err
	}
	return nil
}

func newSaladCloudProvider(pc nodeutil.ProviderConfig) (nodeutil.Provider, node.NodeProvider, error) {
	p, err := provider.NewSaladCloudProvider(context.Background(), inputs)
	if err != nil {
		logrus.WithError(err).Error("Failed to create SaladCloud provider")
		return nil, nil, err
	}
	p.ConfigureNode(context.Background(), pc.Node)
	return p, nil, nil
}

func withTaint(cfg *nodeutil.NodeConfig) error {
	if inputs.DisableTaint {
		return nil
	}

	taintEffect, validEffect := taintEffectMapping[inputs.TaintEffect]
	if !validEffect {
		err := errdefs.InvalidInputf("Taint effect %q is not supported", inputs.TaintEffect)
		logrus.WithError(err).Error("Invalid taint effect provided")
		return err
	}

	cfg.NodeSpec.Spec.Taints = append(cfg.NodeSpec.Spec.Taints, v1.Taint{
		Key:    inputs.TaintKey,
		Value:  inputs.TaintValue,
		Effect: taintEffect,
	})

	return nil
}

func withClient(cfg *nodeutil.NodeConfig) error {
	client, err := nodeutil.ClientsetFromEnv(inputs.KubeConfig)
	if err != nil {
		logrus.WithError(err).Error("Failed to retrieve clientset from environment")
		return err
	}
	cfg.Client = client
	return nil
}

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

var virtualKubeletCommand = &cobra.Command{
	Use:   binaryFilename,
	Short: description,
	Long:  description,
	Run: func(cmd *cobra.Command, args []string) {
		logger := logrus.StandardLogger()
		if logLevel, err := logrus.ParseLevel(inputs.LogLevel); err == nil {
			logger.SetLevel(logLevel)
		} else {
			logrus.WithError(err).Error("Failed to parse log level, defaulting to INFO")
		}

		ctx := log.WithLogger(cmd.Context(), logruslogger.FromLogrus(logrus.NewEntry(logger)))
		if err := runNode(ctx); err != nil {
			logrus.WithError(err).Fatal("Node failed to run")
		}
	},
}

func init() {
	initCommandFlags()
}
