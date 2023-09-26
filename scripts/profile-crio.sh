#!/bin/bash
date
echo "Running CRIO profiling on node $NODE_IP"
curl --unix-socket /var/run/crio/crio.sock http://localhost/debug/pprof/profile --output "/host/tmp/pprofs/crio-${NODE_IP}_$(date +"%F-%T.%N").pb.gz"
status=$?
[ $status -eq 0 ] && echo "CRIO profiling file saved to /host/tmp/pprofs/" || echo "CRIO profiling failed"
