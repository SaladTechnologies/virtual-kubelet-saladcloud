# SaladCloud Virtual Kubelet

[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/virtual-kubelet-saladcloud-chart)](https://artifacthub.io/packages/search?repo=virtual-kubelet-saladcloud-chart) [![License](https://img.shields.io/github/license/SaladTechnologies/virtual-kubelet-saladcloud)](./LICENSE) [![CI Workflow](https://github.com/SaladTechnologies/virtual-kubelet-saladcloud/actions/workflows/ci.yml/badge.svg?branch=main&event=push)](https://github.com/SaladTechnologies/virtual-kubelet-saladcloud/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/SaladTechnologies/virtual-kubelet-saladcloud)](https://goreportcard.com/report/github.com/SaladTechnologies/virtual-kubelet-saladcloud)

<center>
  <a href="https://github.com/SaladTechnologies/virtual-kubelet-saladcloud"><img alt="SaladCloud Virtual Kubelet" src="./images/saladcloud-virtual-kubelet-banner.png" width="100%" /></a>
</center>

Salad's Virtual Kubelet (VK) provider for SaladCloud enables running Kubernetes (K8s) pods as SaladCloud container group deployments. This prototype is currently in development, and not yet ready for production scenarios. If you are interested, please reach out to us to discuss your specific needs.

## How it Works

The SaladCloud Virtual Kubelet creates a _virtual node_ in your K8s cluster.

<center>
  <a href="https://github.com/SaladTechnologies/virtual-kubelet-saladcloud"><img alt="architecture diagram of the SaladCloud Virtual Kubelet" src="./images/saladcloud-virtual-kubelet-architecture.png" width="100%" /></a>
</center>

To the K8s API, it looks like a real node. However, when you schedule a pod on the virtual node, a container group deployment is created using the SaladCloud API instead of running the pod on a node in the K8s cluster. The container group deployment runs the pod on a remote, GPU-enabled node on the SaladCloud network.

## Demonstration

This was used in a presentation at KubeCon2023, the script and pod spec file for the QR code workload are in the [demo](demo) directory.

## Deployment

See the [Helm chart](./charts/virtual-kubelet-saladcloud-chart) for deployment instructions.
