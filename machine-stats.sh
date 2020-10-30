#!/bin/bash
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
cpu_usage_percent=$(grep 'cpu ' /proc/stat | awk '{usage=($2+$4)*100/($2+$4+$5)} END {print usage "%"}')
free_memory_mb=$(awk '/Mem:/ {print $4}' <(free -m))

echo $timestamp $cpu_usage_percent $free_memory_mb >> ./machine_stats.log
