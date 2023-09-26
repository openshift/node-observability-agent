#!/bin/bash
# monitor.sh begins here
# Save this script as monitor.sh
# Allocate read write execute permissions: chmod +rwx monitor.sh
# Help available with: ./monitor.sh -h
# License: Creative Commons Zero - https://creativecommons.org/publicdomain/zero/1.0/

VERSION=52

USAGE=$(cat <<-EOM
Usage: monitor.sh [-d DELAY] [-i ITERATIONS] [-h]

This script collects data relevant to network debugging.
Valid parameters are all optional.

-d DELAY
Specifies a delay between collections. Default is 30 seconds.

    Examples:
    ./monitor.sh -d 10   # 10 seconds
    ./monitor.sh -d 2    # 2 seconds

-i ITERATIONS
Specifies the number of collections. Default is to run forever.

  Examples:
  ./monitor.sh -i 10   # 10 iterations
  ./monitor.sh -i 2    # 2 iterations

-p
Disables process collection in "ss", except when SS_OPTS used.
Default is process collection enabled when SS_OPTS not provided.

  Example:
  ./monitor.sh -p

-h
Displays this help message.

  Example:
  ./monitor.sh -h

Options can be combined.

  Example:
  ./monitor.sh -d 10 -i 360    # run every 10 secs, for an hour

This script recognizes an environment variable SS_OPTS which will
override the script's default command line switches when running
the 'ss' utility.

  Example:
  env SS_OPTS="-pantoemi sport = :22" bash monitor.sh
EOM
)

## defaults

DELAY=30
ITERATIONS=-1
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

#
# Removed default addition of -S for ss options due to
# https://bugzilla.redhat.com/show_bug.cgi?id=1982804
# which causes ss coredump in RHEL8.0 - RHEL8.4. when there
# are active SCTP associations
# 
#if [ -z "$SS_OPTS" ] ; then
#    if ! ss -S 2>&1 | grep -q "invalid option"; then
#        REAL_SS_OPTS+="S"
#    fi
#fi

## reporting

if [ "$ITERATIONS" -gt 0 ]; then
    echo "Running network monitoring with $DELAY second delay for $ITERATIONS iterations."
else
    echo "Running network monitoring with $DELAY second delay. Press Ctrl+c to stop..."
fi

## Check PATH because when run from cron /usr/sbin often isn't in PATH
## ip & ss come from /usr/sbin/

which ss 2>&1 | grep -q ": no ss in"
if [ "$?" -eq 0 ] ; then
    if [ ! -x /usr/sbin/ss ] ; then 
        echo "ss binary not found, check if iproute2 rpm is installed."
    else
        echo $PATH | grep -q ":/usr/sbin:"
        if [ "$?" -gt 0 ] ; then
            PATH=/usr/sbin:$PATH
        fi
    fi
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
    for DEV in $(ip a l | grep mtu | awk '{print $2}' | awk -F "[:@]" '{print $1}'); do echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/ethtool_$DEV"; ethtool -S "$DEV" >> "$HOSTNAME-network_stats_$now/ethtool_$DEV" 2>/dev/null; done

    tmp_file=$(mktemp)
    find /sys/devices -type f | grep -E '/net/.*/statistics' | xargs grep .  > "${tmp_file}"
    for DEV in $(ip a l | grep mtu | awk '{print $2}' | awk -F "[:@]" '{print $1}'); do
          echo "===== $(date +"%F %T.%N%:z (%Z)") =====" >> "$HOSTNAME-network_stats_$now/sys_statistics_$DEV"
          awk -F "/" "/${DEV}/{print \$NF}" >> "$HOSTNAME-network_stats_$now/sys_statistics_$DEV" < "${tmp_file}"
    done
    rm -f "${tmp_file}"

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
