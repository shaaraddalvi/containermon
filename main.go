package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var (
	containerName = flag.String("container", "", "Name or ID of the container to monitor")
	outputFormat  = flag.String("output-format", "json", "Output format: json | csv")
	interval      = flag.Int("interval", 10, "collection interval (in seconds)")
	file          = flag.String("file", "/tmp/containerlog", "Path to file for output, if not provided defaults to stdout")

	previousTotalUsage uint64
	previousTime       time.Time
)

func main() {
	flag.Parse()
	if *containerName == "" {
		fmt.Println("--container flag is required")
		flag.Usage()
		return
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	// Crate output writer
	var writer *bufio.Writer

	// If file is provided then open it for writing
	if *file != "" {
		f, err := os.Create(*file)
		if err != nil {
			panic(err)
		}

		writer = bufio.NewWriter(f)
	} else {
		writer = bufio.NewWriter(os.Stdout)
	}

	stats := getStats(cli)
	start := time.Now()
	startUsage := stats.CPUStats.CPUUsage.TotalUsage
	previousTotalUsage = stats.CPUStats.CPUUsage.TotalUsage
	previousTime = start
	ticker := time.NewTicker(time.Duration(*interval) * time.Second)
	if *outputFormat == "csv" {
		_, err = fmt.Fprintf(writer, "ts,timeElapsed,cpuTimeElapsed,percentCPUSinceStart,percentCPUThisInterval,memoryUsageMiB,percentMemoryUsage")

		if err != nil {
			panic(err)
		}
		writer.Flush()
	}
	for range ticker.C {
		stats = getStats(cli)
		now := time.Now()
		elapsed := now.Sub(start)
		intervalElapsed := now.Sub(previousTime)
		printStats(writer, stats, now, elapsed, intervalElapsed, startUsage)
		previousTotalUsage = stats.CPUStats.CPUUsage.TotalUsage
		previousTime = now
	}
}

func getStats(cli *client.Client) *types.StatsJSON {
	resp, err := cli.ContainerStats(context.TODO(), *containerName, false)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	stats := &types.StatsJSON{}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(buf, stats)
	if err != nil {
		panic(err)
	}

	return stats
}

func printStats(writer *bufio.Writer, stats *types.StatsJSON, now time.Time, elapsed time.Duration, intervalElapsed time.Duration, startUsage uint64) {
	ts := now.UTC().Format(time.RFC3339)
	timeElapsed := elapsed.Seconds()
	// cpu time in seconds
	cpuTimeElapsed := float64(stats.CPUStats.CPUUsage.TotalUsage-startUsage) / 1000000000
	percentCPUSinceStart := float64(stats.CPUStats.CPUUsage.TotalUsage-startUsage) / float64(elapsed.Nanoseconds()) * 100
	percentCPUThisInterval := float64(stats.CPUStats.CPUUsage.TotalUsage-previousTotalUsage) / float64(intervalElapsed.Nanoseconds()) * 100
	percentMemoryUsage := (float64(stats.MemoryStats.Usage) / 1024 / 1024) / (float64(stats.MemoryStats.Limit) / 1024 / 1024) * 100

	if *outputFormat == "csv" {
		// csv
		// ts,timeElapsed,cpuTimeElapsed,percentCPUSinceStart,percentCPUThisInterval,memoryUsageMiB,percentMemoryUsage
		_, err := fmt.Fprintf(writer, "%s,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f\n",
			ts,
			timeElapsed,
			cpuTimeElapsed,
			percentCPUSinceStart,
			percentCPUThisInterval,
			float64(stats.MemoryStats.Usage)/1024/1024,
			percentMemoryUsage)

		if err != nil {
			panic(err)
		}

		writer.Flush()
	} else {
		// json
		_, err := fmt.Fprintf(writer, `{"ts":"%s","timeElapsed":%.2f,"cpuTimeElapsed":%.2f,"percentCPUSinceStart":%.2f,"percentCPUThisInterval":%.2f,"memoryUsageMiB":%.2f,"memoryUsagePercentage":%.2f}`,
			ts,
			timeElapsed,
			cpuTimeElapsed,
			percentCPUSinceStart,
			percentCPUThisInterval,
			float64(stats.MemoryStats.Usage)/1024/1024,
			percentMemoryUsage)

		if err != nil {
			panic(err)
		}

		fmt.Fprintf(writer, "\n")
		writer.Flush()
	}
}
