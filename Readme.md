# Goku - Super Saiyan Kubernetes Deployment
> Watchs app code to deploy new Docker images to Kubernetes

**This tool is still in very early alpha stages**

### Requirements
* Minikube + kubectl
* Helm tiller installed on Kubernetes cluster (`helm init`)

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


## CLI Usage:
```
Usage:
  goku [command]
Available Commands:
  config      Checks goku.yaml config structure
  help        Help about any command
  init        Download kubernetes binaries locally
  start       Create a new minikube, enable addons: ingress, helm, heapster
  version     Print the version number of Goku
  watch       Watch goku managed containers for changes and redeploy to Kubernetes via Helm
Flags:
  -h, --help     help for goku
  -t, --toggle   Help message for toggle
Use "goku [command] --help" for more information about a command.
```

#### Bugs & TODO
- BUG: Helm values `imageValueName` Can't contain period `, . - _` characters at the moment.
- TODO check that `kubectl config get-context` == 'minikube'`. Not some other production cluster!!!
### Disclaimer
You probably should not use this in production!

