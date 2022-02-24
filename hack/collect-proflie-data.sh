#!/bin/bash

# assumes oc cli is insyalled and current user is logged in

#set some envars
NAMESPACE=node-observability-operator
# used to filter pods
PODFILTER=nodeobservability
# used for profile data downloads
LOCALDIR=$HOME/profiledata
# pod mount 
PODDIR=/mnt

mkdir -p $LOCALDIR

oc project $NAMEPSACE

PODS=$(oc get pods | grep $PODFILTER | awk '{print $1}')

echo $PODS

for pod in $PODS
do
  oc rsync $pod:$PODDIR $LOCALDIR
done

