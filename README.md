# Node Observability Agent

The agent exposes port 9000 by default, unless a `--port` parameter is passed to it. 

It accepts requests for the following endpoints:

- Kubelet Profiling: `/kubelet/profiling`
- CRIO Profiling: `/crio/profiling`

## Run the agent

## Running

The agent can be run locally but is best run in a pod on a Kubernetes cluster.


E.g.:

```bash
./node-observability-agent --tokenFile /var/run/secrets/kubernetes.io/serviceaccount/token --storage /host/tmp/pprofs/
```

Where:
* tokenFile: is a file that contains the JWT token that has permissions to trigger Kubelet and CRIO profiling
* storage: is a folder to which the profiling files will be stored.
