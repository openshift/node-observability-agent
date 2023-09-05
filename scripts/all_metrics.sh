#!/bin/bash

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


DELAY=30
#ITERATIONS=-1
ITERATIONS=5
DEF_SS_OPTS="-noemitaup"
DEF_SS_OPTS_NOP="-noemitau"

## option parsing

REAL_SS_OPTS=${SS_OPTS:-$DEF_SS_OPTS}

while getopts ":d:i:ph" OPT; do
    case "$OPT" in
        "d")
            # something was passed, check it's a positive integer
            if [ "$OPTARG" -eq "$OPTARG" ] 2>/dev/null && [ "$OPTARG" -gt 0 ] 2>/dev/null; then
                DELAY="$OPTARG"
            else
                echo "ERROR: $OPTARG not a valid option for delay. Run 'monitor.sh -h' for help."
                exit 1
            fi
            ;;
        "i")
            # something was passed, check it's a positive integer
            if [ "$OPTARG" -eq "$OPTARG" ] 2>/dev/null && [ "$OPTARG" -gt 0 ] 2>/dev/null; then
                ITERATIONS="$OPTARG"
            else
                echo "ERROR: $OPTARG not a valid option for iterations. Run 'monitor.sh -h' for help."
                exit 1
            fi
            ;;
        "p")
            REAL_SS_OPTS=${SS_OPTS:-$DEF_SS_OPTS_NOP}
            ;;
        "h")
            echo "$USAGE"
            exit 0
            ;;
        ":")
            echo "ERROR: -$OPTARG requires an argument. Run 'monitor.sh -h' for help."
            exit 1
            ;;
        "?")
            echo "ERROR: -$OPTARG is not a valid option. Run 'monitor.sh -h' for help."
            exit 1
            ;;
    esac
done

if [ -z "$SS_OPTS" ] ; then
    if ! ss -S 2>&1 | grep -q "invalid option"; then
        REAL_SS_OPTS+="S"
    fi
fi

## reporting

if [ "$ITERATIONS" -gt 0 ]; then
    echo "Running network monitoring with $DELAY second delay for $ITERATIONS iterations."
else
    echo "Running network monitoring with $DELAY second delay. Press Ctrl+c to stop..."
fi
## one-time commands

MQDEVS=( $(tc qdisc show | awk '/^qdisc mq/{print $(NF-1)}') )

## data collection loop
while [ "$ITERATIONS" != 0 ]; do

    #start timer in background
    eval sleep "$DELAY" &

    now=$(date +%Y_%m_%d_%H)
    then=$(date --date="yesterday" +%Y_%m_%d_%H)
    rm -rf "$HOSTNAME-network_stats_$then"
    mkdir -p "$HOSTNAME-network_stats_$now"

    if ! [ -e "$HOSTNAME-network_stats_$now/version.txt" ]; then
        echo "===== $(date +"%F %T.%N%:z (%Z)") =====" > "$HOSTNAME-network_stats_$now/version.txt"
        echo "This output created with monitor.sh version $VERSION" >> "$HOSTNAME-network_stats_$now/version.txt"
        echo "See https://access.redhat.com/articles/1311173" >> "$HOSTNAME-network_stats_$now/version.txt"
        echo "Delay: $DELAY" >> "$HOSTNAME-network_stats_$now/version.txt"
        echo "Iterations: $ITERATIONS" >> "$HOSTNAME-network_stats_$now/version.txt"
    echo "SS_OPTS: $REAL_SS_OPTS" >> "$HOSTNAME-network_stats_$now/version.txt"
    fi
    if ! [ -e "$HOSTNAME-network_stats_$now/sysctl.txt" ]; then
        echo "===== $(date +"%F %T.%N%:z (%Z)") =====" > "$HOSTNAME-network_stats_$now/sysctl.txt"
        sysctl -a 2>/dev/null >> "$HOSTNAME-network_stats_$now/sysctl.txt"
    fi  
    if ! [ -e "$HOSTNAME-network_stats_$now/ip-address.txt" ]; then
        echo "===== $(date +"%F %T.%N%:z (%Z)") =====" > "$HOSTNAME-network_stats_$now/ip-address.txt"
        ip address list >> "$HOSTNAME-network_stats_$now/ip-address.txt"
    fi
    if ! [ -e "$HOSTNAME-network_stats_$now/ip-route.txt" ]; then
        echo "===== $(date +"%F %T.%N%:z (%Z)") =====" > "$HOSTNAME-network_stats_$now/ip-route.txt"
        ip route show table all >> "$HOSTNAME-network_stats_$now/ip-route.txt"
    fi
    if ! [ -e "$HOSTNAME-network_stats_$now/uname.txt" ]; then
        echo "===== $(date +"%F %T.%N%:z (%Z)") =====" > "$HOSTNAME-network_stats_$now/uname.txt"
        uname -a >> "$HOSTNAME-network_stats_$now/uname.txt"
    fi

    echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/ip_neigh"
    ip neigh show >> "$HOSTNAME-network_stats_$now/ip_neigh"
    echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/tc_qdisc"
    tc -s qdisc >> "$HOSTNAME-network_stats_$now/tc_qdisc"
    if [ "${#MQDEVS[@]}" -gt 0 ]; then
        for MQDEV in "${MQDEVS[@]}"; do
            echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/tc_class_$MQDEV"
            tc -s class show dev "$MQDEV" >> "$HOSTNAME-network_stats_$now/tc_class_$MQDEV"
        done
    fi
    echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/netstat"
    netstat -s >> "$HOSTNAME-network_stats_$now/netstat"
    echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/nstat"
    nstat -az >> "$HOSTNAME-network_stats_$now/nstat"
    echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/ss"
    eval "ss $REAL_SS_OPTS" >> "$HOSTNAME-network_stats_$now/ss"
    echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/interrupts"
    cat /proc/interrupts >> "$HOSTNAME-network_stats_$now/interrupts"
    echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/softnet_stat"
    cat /proc/net/softnet_stat >> "$HOSTNAME-network_stats_$now/softnet_stat"
    echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/vmstat"
    cat /proc/vmstat >> "$HOSTNAME-network_stats_$now/vmstat"
    echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/ps"
    ps -alfe >> "$HOSTNAME-network_stats_$now/ps"
    echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/mpstat"
    eval mpstat -A "$DELAY" 1 2>/dev/null >> "$HOSTNAME-network_stats_$now/mpstat" &
    echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/top"
    top -c -b -n1 >> "$HOSTNAME-network_stats_$now/top"
    echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/numastat"
    numastat 2>/dev/null >> "$HOSTNAME-network_stats_$now/numastat"
    if [ -e /proc/softirqs ]; then
        echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/softirqs"
        cat /proc/softirqs >> "$HOSTNAME-network_stats_$now/softirqs"
    fi
    echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/sockstat"
    cat /proc/net/sockstat >> "$HOSTNAME-network_stats_$now/sockstat"
    if [ -e /proc/net/sockstat6 ]; then
        echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/sockstat6"
        cat /proc/net/sockstat6 >> "$HOSTNAME-network_stats_$now/sockstat6"
    fi
    echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/netdev"
    cat /proc/net/dev >> "$HOSTNAME-network_stats_$now/netdev"
    for DEV in $(ip a l | grep mtu | awk '{print $2}' | awk -F ":" '{print $1}' | awk -F '@' '{print $1}'); do echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/ethtool_$DEV"; ethtool -S "$DEV" >> "$HOSTNAME-network_stats_$now/ethtool_$DEV" 2>/dev/null; done
    for DEV in $(ip a l | grep mtu | awk '{print $2}' | awk -F ":" '{print $1}' | awk -F '@' '{print $1}'); do echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/sys_statistics_$DEV"; find /sys/devices/ -type f | grep "/net/$DEV/statistics" | xargs grep . | awk -F "/" '{print $NF}' >> "$HOSTNAME-network_stats_$now/sys_statistics_$DEV"; done
    if [ -e /proc/net/sctp ]; then
        echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/sctp-assocs"
        cat /proc/net/sctp/assocs >> "$HOSTNAME-network_stats_$now/sctp-assocs"
        echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/sctp-snmp"
        cat /proc/net/sctp/snmp >> "$HOSTNAME-network_stats_$now/sctp-snmp"
    fi
    if [ "$ITERATIONS" -gt 0 ]; then let ITERATIONS-=1; fi
    # Wait till background jobs are finished
    wait
done

tar -czf archives/$HOSTNAME-network_stats_$now.tar.gz  $HOSTNAME-network_stats_$now/
 
status=$?
[ $status -eq 0 ] && echo "Network stats collection completed" || echo "Network stats collection failed"

#TODO this is failing
readarray -t COREDNS_PIDS < <(pgrep coredns)
# check if directory exists
if [ ! -d "$HOSTNAME-network_metrics_$now" ];
then
  mkdir -p "$HOSTNAME-network_metrics_$now"
fi

rm -Rf "$HOSTNAME-network_metrics_$now"/*
cd "$HOSTNAME-network_metrics_$now"
NSENTER_PIDS=()
CONNTRACK_PIDS=()
for COREDNS_PID in "${COREDNS_PIDS[@]}"
do      
   nsenter -n -t $COREDNS_PID bash -c "while true ; do date ; conntrack -L -n ; sleep 5; done" >> conntrack-${COREDNS_PID}.txt &
   CONNTRACK_PIDS+=($!)
   mkdir -p "monitor-${COREDNS_PID}"
   pushd "monitor-${COREDNS_PID}"
   nsenter -n -t $COREDNS_PID bash /monitor/monitor.sh -d 5 -i 120 &
   NSENTER_PIDS+=($!)
   popd
done
wait "${NSENTER_PIDS[@]}"
if [ "${#CONNTRACK_PIDS[@]}" -gt 0 ];
then
  kill "${CONNTRACK_PIDS[@]}"
  tar -czf "archives/$HOSTNAME-network_metrics_$now.tar.gz" "$HOSTNAME-network_metrics_$now/"
fi
 
status=$?
[ $status -eq 0 ] && echo "Network metrics collection completed" || echo "Network metrics collection failed"
