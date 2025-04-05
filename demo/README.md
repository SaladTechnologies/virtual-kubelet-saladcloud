# Salad Cloud Virtual Kubelet Demo

These files were originally used in dtroyer's KubeCon2023 talk.

## Run the Demo

This outlines running the demo on Docker Desktop's Kubernetes implementation:

* Set up the environment with a .env file (or equivalent) similar to this one:
  ```bash
  export LOG_LEVEL=INFO
  export SALAD_API_KEY=<api-key>
  export SALAD_ORGANIZATION_NAME=salad
  export SALAD_PROJECT_NAME=demo
  export NAMESPACE=saladcloud-demo
  export NODE_NAME=${SALAD_PROJECT_NAME}
  ```
* Start with a fresh Kubernetes environment.  Only the docker-desktop conrol plane should be displayed
  with `kubectl get node` and `kubectl get pod`.  `demo.sh status` will run both of those commands
  at once.
* Run the virtual kubelet via its Helm chart using `demo.sh start`.
* Run `demo.sh status` again to see that the virtual kubelet is registered as an agent
* Start the QR code workload with `kubectl apply -f qr.yaml`.
* Run `demo.sh status` again to see two additional pods listed that are the requested container groups.
  The pod names should match the Container Groups in the Salad Portal (NOTE: with a prefix!)
* Note that Kubernetes pods are mapped to SCE Container Groups 1:1, specifying 2 replicas on the YAML spec
  will result in two Contarer Groups being created.  At this time there is not a mechanism to specify the
  Container Group replicas and it will always be 1.
* Once a Container Group is shown running, grab the assigned URL from the portal and paste it into a browser.
  Commence generating crazy QR codes that look like city skylines or a plaid flannel shirt.
* Run `kubectl delete -f qr.yaml` to stop the Container Groups.
* Run `demo.sh stop` to stop the virtual kubelet pod.
