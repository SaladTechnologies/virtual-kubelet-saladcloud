package main

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"

	"github.com/SaladTechnologies/virtual-kubelet-saladcloud/internal/provider"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/virtual-kubelet/virtual-kubelet/node"
	"github.com/virtual-kubelet/virtual-kubelet/node/nodeutil"
)

func main() {
	log.Info("SaladCloud Virtual Kubelet Provider")

	name := fmt.Sprintf("saladcloud-%s", randSeq(8))
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
