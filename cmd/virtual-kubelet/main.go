package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/provider"
	"github.com/sirupsen/logrus"

	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/virtual-kubelet/virtual-kubelet/node"
	"github.com/virtual-kubelet/virtual-kubelet/node/nodeutil"
)

var (
	buildVersion   = "N/A"
	k8sVersion     = "v1.25.0" // This should follow the version of k8s.io we are importing
	binaryFilename = filepath.Base(os.Args[0])
	description    = fmt.Sprintf("%s implements a node on a Kubernetes cluster using StackPath Workload API to run pods.", binaryFilename)
	listenPort     = int32(10250)
)

func main() {
	log.Info("SaladCloud Virtual Kubelet Provider")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := virtualKubeletCommand.ExecuteContext(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			logrus.WithError(err).Fatal("error running command")
		}
	}

}

func runNode(name string) {
	node, err := nodeutil.NewNode(name, func(pc nodeutil.ProviderConfig) (nodeutil.Provider, node.NodeProvider, error) {
		p, err := provider.NewSaladCloudProvider(context.Background())
		if err != nil {
			return nil, nil, err
		}

		return p, nil, nil
	}, withClient)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		if err := node.Run(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()

	err = node.WaitReady(context.Background(), 0)
	if err != nil {
		log.Fatal(err)
	}

	<-node.Done()
	err = node.Err()
	if err != nil {
		log.Fatal(err)
	}

}

var virtualKubeletCommand = &cobra.Command{
	Use:   binaryFilename,
	Short: description,
	Long:  description,
	Run: func(cmd *cobra.Command, args []string) {
		name := fmt.Sprintf("saladcloud-%s", randSeq(8))
		runNode(name)
	},
}

func withClient(cfg *nodeutil.NodeConfig) error {
	var conf string
	home, _ := homedir.Dir()
	if home != "" {
		conf = filepath.Join(home, ".kube", "config")
	} else {
		conf = "config"
	}

	client, err := nodeutil.ClientsetFromEnv(conf)
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
