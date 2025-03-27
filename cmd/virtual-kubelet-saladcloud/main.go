package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/models"
	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/provider"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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
	var kubeConfig string

	// Default config file in $HOME
	home, err := homedir.Dir()
	if err == nil && home != "" {
		kubeConfig = filepath.Join(home, ".kube", "config")
	}

	// If KUBECONFIG is defined use it instead, if it is empty
	// set the default config file name to ""
	if kc, ok := os.LookupEnv("KUBECONFIG"); ok {
		kubeConfig = kc
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
	virtualKubeletCommand.Flags().StringVar(&inputs.KubeConfig, "kubeconfig", inputs.KubeConfig, "Kubeconfig file")
	virtualKubeletCommand.Flags().MarkHidden("kube-config")
	virtualKubeletCommand.MarkFlagsMutuallyExclusive("kube-config", "kubeconfig")
	virtualKubeletCommand.Flags().BoolVar(&inputs.DisableTaint, "disable-taint", inputs.DisableTaint, "Disable the tainted effect")
	virtualKubeletCommand.Flags().StringVar(&inputs.LogLevel, "log-level", inputs.LogLevel, "Log level for the node")
	virtualKubeletCommand.Flags().StringVar(&inputs.ApiKey, "sce-api-key", inputs.ApiKey, "SaladCloud API Key")
	virtualKubeletCommand.Flags().StringVar(&inputs.OrganizationName, "sce-organization-name", inputs.OrganizationName, "SaladCloud Organization Name")
	virtualKubeletCommand.Flags().StringVar(&inputs.ProjectName, "sce-project-name", inputs.ProjectName, "SaladCloud Project Name")
}

func runNode(ctx context.Context) error {
	logrus.Infof("Running node with name: %s", inputs.NodeName)
	logrus.Infof("Running node with log level: %s", inputs.LogLevel)

	node, err := nodeutil.NewNode(inputs.NodeName, func(config nodeutil.ProviderConfig) (nodeutil.Provider, node.NodeProvider, error) {
		return newSaladCloudProvider(ctx, config)
	}, withClient, withTaint)
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

func newSaladCloudProvider(ctx context.Context, pc nodeutil.ProviderConfig) (nodeutil.Provider, node.NodeProvider, error) {
	p, err := provider.NewSaladCloudProvider(ctx, inputs, pc)
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
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		v := viper.New()
		v.SetEnvPrefix("SALAD")
		v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
		v.AutomaticEnv()
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			envName := ""
			switch f.Name {
			case "log-level":
				envName = "VK_LOG_LEVEL"
			case "nodename":
				envName = "VK_NODE_NAME"
			case "sce-api-key":
				envName = "CLOUD_API_KEY"
			case "sce-organization-name":
				envName = "CLOUD_ORGANIZATION_NAME"
			case "sce-project-name":
				envName = "CLOUD_PROJECT_NAME"
			}

			if envName != "" && !f.Changed && v.IsSet(envName) {
				_ = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", v.Get(envName)))
			}
		})

		if inputs.ApiKey == "" {
			logrus.Fatal("A SaladCloud API Key is required")
		}

		if inputs.OrganizationName == "" {
			logrus.Fatal("A SaladCloud organization name is required")
		}

		if inputs.ProjectName == "" {
			logrus.Fatal("A SaladCloud project name is required")
		}

		if !cmd.Flags().Changed("nodename") {
			inputs.NodeName = fmt.Sprintf("%s-%s", inputs.NodeName, randSeq(3))
		}
	},
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
