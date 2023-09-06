#!/bin/bash
#echo "Gathering metrics ..."
mkdir network-metrics
rm -Rf network-metrics/*
#echo "Gathering monitor metrics ..."
cd network-metrics
bash -c "while true ; do date ; conntrack -L -n ; sleep 5; done" >> conntrack.txt &
CONNTRACK=$!
bash /tmp/scripts/monitor.sh -d 5 -i 120
kill $CONNTRACK
tar -czf /network-metrics.tar.gz /network-metrics
#echo "Done with network metrics collection."
