# Node Observability Agent

The agent exposes port 9000 by default, unless a `--port` parameter is passed to it. 

List of required parameters to be passed to the agent:
- `NODE_IP` environment variable: IP address of the node on which to perform the profiling
- `--storage` flag : folder to which the pprof files are saved
- `--tokenFile` flag : file containing token to be used for kubelet profiling http request
- `--mode=profiling` flag: 'profiling' default or 'scripting': used to enable profiling or executing metric type bash scripts

It accepts requests for the following endpoints:

- Kubelet + CRIO Profiling: `/node-observability-pprof`
- Scripting: `node-observability-scripting`
- Status update: `/node-observability-status`

The agent doesn't accept concurrent requests: only one profiling request can run at a time. 
Therefore, `/node-observability-status` as well as `/node-observability-pprof` or `/node-observability-scripting` will return a 409 error if the agent is already running a profiling request. 
In case of error, `/node-observability-status` and `/node-observability-pprof` or `/node-observability-scripting` will return a 500 error. The agent will remain in error until an admin has cleared the `agent.err` file that is stored in the `storageFolder`. 

## Interaction with Node Observability Operator

Please refer to the [node-observability-operator(https://github.com/openshift/node-observability-operator) for details on how to use the agent

The project will highlight how to change the agent image (in a daemonset) as well as how to create a 'run' CRD (to execute profiling/scripts etc on each node)

## Run the agent

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

## Run using scripting functionality (local dev)

First build a container using the Dockerfile.dev file

For now the script support is limited to 2 scripts (refer to scripts/metrics directory)

- metrics.sh
- network-metrics.sh (uses monitor.sh)

These scripts will be copied to the /tmp/scripts folder in the container

```bash
podman build -t quay.io/<user-name>/node-observability-scripts:dev -f Dockerfile.dev

```

```bash
# create results directory
mkdir -p /tmp/results

podman run --privileged --cap-add=NET_ADMIN -e NODE_IP=127.0.0.1 -e EXECUTE_SCRIPT=/tmp/scripts/metrics.sh -p 9000:9000 \ 
-it quay.io/<user-name>/node-observability-scripts:dev ./node-observability-agent --mode scripting --storage /tmp/results --loglevel debug

```

## Execute the script

Use status end point to check 

```bash
curl  -v http://127.0.0.1:9000/node-observability-status

*   Trying 127.0.0.1:9000...
* Connected to 127.0.0.1 (127.0.0.1) port 9000 (#0)
> GET /node-observability-status HTTP/1.1
> Host: 127.0.0.1:9000
> User-Agent: curl/8.0.1
> Accept: */*
> 
< HTTP/1.1 200 OK
< Date: Fri, 08 Sep 2023 10:55:32 GMT
< Content-Length: 16
< Content-Type: text/plain; charset=utf-8
< 
* Connection #0 to host 127.0.0.1 left intact
Service is ready

```

Call the start script end point

```bash
curl  -v http://127.0.0.1:9000/node-observability-scripting

*   Trying 127.0.0.1:9000...
* Connected to 127.0.0.1 (127.0.0.1) port 9000 (#0)
> GET /node-observability-scripting HTTP/1.1
> Host: 127.0.0.1:9000
> User-Agent: curl/8.0.1
> Accept: */*
> 
< HTTP/1.1 200 OK
< Content-Type: application/json
< Date: Fri, 08 Sep 2023 10:55:41 GMT
< Content-Length: 66
< 
* Connection #0 to host 127.0.0.1 left intact
{"ID":"e669a8cb-b9b0-4f21-a57d-9a2228e8bc9b","ExecutionRuns":null}
```

Check the output log of the running podman instance

```bash
podman run --privileged --cap-add=NET_ADMIN -e NODE_IP=127.0.0.1 -e EXECUTE_SCRIPT=/tmp/scripts/metrics.sh -p 9000:9000 \ 
-it quay.io/<user-name>/node-observability-scripts:dev ./node-observability-agent --mode scripting --storage /tmp/results --loglevel debug

WARN[0000] Error validating CNI config file /home/lzuccarelli/.config/cni/net.d/kind.conflist: [failed to find plugin "dnsname" in path [/usr/local/libexec/cni /usr/libexec/cni /usr/local/lib/cni /usr/lib/cni /opt/cni/bin]] 
INFO[0000] Starting node-observability-agent version: "v0.0.0-unknown", commit: "00801d1", build date: "2023-08-29T11:26:31Z", go version: "go1.19.10", GOOS: "linux", GOARCH: "amd64" at log level trace 
INFO[0000] Start listening on tcp://0.0.0.0:9000         module=server
INFO[0000] Targeting node 127.0.0.1                      module=server
INFO[0006] start handling status request                 module=handler
INFO[0006] agent is ready                                module=handler
INFO[0018] start handling status request                 module=handler
INFO[0018] previous profiling is still ongoing, runID: e669a8cb-b9b0-4f21-a57d-9a2228e8bc9b  module=handler
INFO[0025] Metrics collection completed                  module=handler
INFO[0025] successfully finished running Scripting - e669a8cb-b9b0-4f21-a57d-9a2228e8bc9b: 2023-09-08 10:55:41.63103853 +0000 UTC m=+15.849803234 -> 2023-09-08 10:55:51.726152976 +0000 UTC m=+25.944917673   module=handler

```
