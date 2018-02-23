# Goku - Super Saiyan Kubernetes Deployment
> Watches files to deploy on Kubernetes instantly with CTRL-S using Helm.

### Requirements
* Kubernetes 1.8+
* Minikube (or some other local cluster)
* Tiller installed on Kubernetes cluster (`helm init`)

## Example `goku.yaml` config
All paths to dockerfiles & charts are constructed relative to the `goku.yaml` location.

See `examples/` for further details

```yaml
charts:
- name: testchart
  path: testchart
  images:
  - name: goku/app1
    # The helm variable to override the Docker image name.
    # this will be replaced with <name:TIMESTAMP> on save and be re-deployed.
    imageValueName: app1image
    path: app1
  - name: goku/app2
    imageValueName: app2image
    path: app2
```

## Steps for testing
```bash
# In new terminal:
# To be automated out, setup a port to communicate with tiller (helm server)
PODNAME=$(kubectl get pod -n kube-system -l name=tiller -o jsonpath='{.items[0].metadata.name}')
kubectl -n kube-system port-forward $PODNAME 44134

# Must setup environment to make Docker CLI use minikube VM to build images.
eval $(minikube docker-env)

# Usage:
goku [goku.yaml]
```

#### Bugs & TODO
- BUG: `imageValueName` Can't contain period `, . - _` characters at the moment.
- TODO: Automatically setup port-forwaring to the Tiller gRPC service when Goku is started.
- TODO Add command interface

### Disclaimer
You probably should not use this in production!
