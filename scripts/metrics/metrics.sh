#!/bin/bash
#
# This script is completly based on the KCS article https://access.redhat.com/solutions/5343671

# Resolution in seconds
RESOLUTION=5 
# Duration in seconds (or whatever is acceptable as an argument to "sleep")
DURATION=10

now=$(date +%Y_%m_%d_%H)
rm -rf "archives/"
mkdir -p "archives"
rm -rf results/*
mkdir -p results

#echo "Gathering metrics ..."
if [ ! -d "$HOSTNAME-metrics_$now" ];
then
  mkdir $HOSTNAME-metrics_$now
fi
rm -Rf $HOSTNAME-metrics_$now/*
pidstat -p ALL -T ALL -I -l -r  -t  -u -w ${RESOLUTION} > "$HOSTNAME-metrics_$now/pidstat.txt" &
PIDSTAT=$!
sar -A ${RESOLUTION} > "$HOSTNAME-metrics_$now/sar.txt" &
SAR=$!
bash -c "while true; do date ; ps aux | sort -nrk 3,3 | head -n 20 ; sleep ${RESOLUTION} ; done" > "$HOSTNAME-metrics_$now/ps.txt" &
PS=$!
bash -c "while true ; do date ; free -m ; sleep ${RESOLUTION} ; done" > "$HOSTNAME-metrics_$now/free.txt" &
FREE=$!
bash -c "while true ; do date ; cat /proc/softirqs; sleep ${RESOLUTION}; done" > "$HOSTNAME-metrics_$now/softirqs.txt" &
SOFTIRQS=$!
bash -c "while true ; do date ; cat /proc/interrupts; sleep ${RESOLUTION}; done" > "$HOSTNAME-metrics_$now/interrupts.txt" &
INTERRUPTS=$!
iotop -Pobt > "$HOSTNAME-metrics_$now/iotop.txt" &
IOTOP=$!
#echo "Metrics gathering started. Please wait for completion..."
sleep "${DURATION}"
kill $PIDSTAT
kill $SAR
kill $PS
kill $FREE
kill $SOFTIRQS
kill $INTERRUPTS
kill $IOTOP
 
tar -czf archives/$HOSTNAME-metrics-$now.tar.gz  $HOSTNAME-metrics_$now/
 
status=$?
[ $status -eq 0 ] && echo "Metrics collection completed" || echo "Metrics collection failed"
