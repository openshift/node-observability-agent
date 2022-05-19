#!/bin/bash
kubectl get cm kubelet-serving-ca -n openshift-config-managed -o yaml | yq '.data."ca-bundle.crt"' | awk '{gsub(/\\n/,"\n")}1' > test_resources/default/ca-bundle.crt
