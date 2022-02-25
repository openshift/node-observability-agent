# Node Observability Agent

The agent exposes port 9000 by default, unless a `--port` parameter is passed to it. 

List of requrired parameters to be passed to the agent:
- node : IP address of the node on which to perform the profiling
- storageFolder : folder to which the pprof files are saved
- tokenFile : file containing token to be used for kubelet profiling http request
- crioSocket : file referring to the unix socket to be used for CRIO profiling

It accepts requests for the following endpoints:

- Kubelet + CRIO Profiling: `/pprof`
- Status update: `/status`

## Run the agent

## Running

The agent can be run locally but is best run in a pod on a Kubernetes cluster.


E.g.:

```bash
./node-observability-agent --tokenFile /var/run/secrets/kubernetes.io/serviceaccount/token --storage /host/tmp/pprofs/ --node $NODE_IP
```

