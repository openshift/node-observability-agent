#!/usr/bin/env bash
IMAGE_TAG=$(git rev-parse --short HEAD)
TEMPDIR=$(mktemp -d tmp.k8s.XXXXX)
delete_temp_dir() {
    if [ -d "$TEMPDIR" ]; then
        rm -r "$TEMPDIR"
    fi
}
trap delete_temp_dir EXIT
(
    cd "$TEMPDIR"
    kustomize create --resources ../test_resources/default/
    kustomize edit set image \
        "node-observability-agent=$IMG:$IMAGE_TAG"
    kustomize build
)
