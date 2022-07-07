# Node Observability Agent

The agent exposes port 9000 by default, unless a `--port` parameter is passed to it. 

List of required parameters to be passed to the agent:
- `NODE_IP` environment variable: IP address of the node on which to perform the profiling
- `--storage` flag : folder to which the pprof files are saved
- `--tokenFile` flag : file containing token to be used for kubelet profiling http request
- `--crioUnixSocket` flag : file referring to the unix socket to be used for CRIO profiling

It accepts requests for the following endpoints:

- Kubelet + CRIO Profiling: `/node-observability-pprof`
- Status update: `/node-observability-status`

The agent doesn't accept concurrent requests: only one profiling request can run at a time. 
Therefore, `/node-observability-status` as well as `/node-observability-pprof` will return a 409 error if the agent is already running a profiling request. 
In case of error, `/node-observability-status` and `/node-observability-pprof` will return a 500 error. The agent will remain in error until an admin has cleared the `agent.err` file that is stored in the `storageFolder`. 

## Run the agent

## Running

The agent can be run locally but is best run in a pod on a Kubernetes cluster.

```bash
$ ./hack/kubelet-serving-ca.sh # Build RUN make build
$ IMG=quay.io/user-xyz/node-observability-agent make deploy
$ kubectl port-forward svc/node-observability-agent 9000:80
```


To run locally:

```bash
NODE_IP=$NODE_IP./bin/node-observability-agent --tokenFile /var/run/secrets/kubernetes.io/serviceaccount/token --storage /host/tmp/pprofs/
```

