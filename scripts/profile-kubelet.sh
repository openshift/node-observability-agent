#!/bin/bash
date
echo "Running kubelet profiling on node $NODE_IP"
curl --insecure -H "Authorization: Bearer $(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" https://${NODE_IP}:10250/debug/pprof/profile --output "/host/tmp/pprofs/kubelet-${NODE_IP}_$(date +"%F-%T.%N").pb.gz"
status=$?
[ $status -eq 0 ] && echo "Kubelet profiling file saved to /host/tmp/pprofs/" || echo "Kubelet profiling failed"