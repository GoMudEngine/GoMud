package util

import (
	"fmt"
	"math"
	"runtime"
)

type MemReport func() map[string]MemoryResult

type MemoryResult struct {
	Memory uint64
	Count  int
}

var (
	memoryTrackerNames []string
	memoryTrackers     []MemReport
)

func AddMemoryReporter(name string, reporter MemReport) {
	memoryTrackerNames = append(memoryTrackerNames, name)
	memoryTrackers = append(memoryTrackers, reporter)
}

func GetMemoryReport() (names []string, trackedResults []map[string]MemoryResult) {

	names = append([]string{}, memoryTrackerNames...)
	trackedResults = []map[string]MemoryResult{}

	for _, reporter := range memoryTrackers {
		trackedResults = append(trackedResults, reporter())
	}

	return names, trackedResults
}

func ServerGetMemoryUsage() map[string]MemoryResult {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	ret := map[string]MemoryResult{}
	ret[`HeapAlloc (!Freed)`] = MemoryResult{m.HeapAlloc, 0}                   // Everything that hasn't been garbage collected
	ret[`HeapSys (!Reclaimed)`] = MemoryResult{m.HeapSys, 0}                   // Everything that the OS hasn't reclaimed, even if it was freed by the GC
	ret[`StackSys (Reserved)`] = MemoryResult{m.StackSys, 0}                   // Ho wmuch stack memory is allocated
	ret[`StackInuse (In Use)`] = MemoryResult{m.StackInuse, 0}                 // How much stack memory is being used
	ret[`Sys (Everything)`] = MemoryResult{m.Sys, 0}                           // heap, stacks, and other internal data structures
	ret[`GC Count`] = MemoryResult{uint64(m.NumGC), 0}                         // How many times the GC has been run
	ret[`Maximum Processors`] = MemoryResult{uint64(runtime.GOMAXPROCS(0)), 0} // How many processors are available for goroutines
	ret[`Goroutines Count`] = MemoryResult{uint64(runtime.NumGoroutine()), 0}  // How many goroutines are currently running

	return ret
}

func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit { // For bytes to display as KB
		return fmt.Sprintf("%5.1f KB", float64(bytes)/1024)
	}

	exp := int(math.Log(float64(bytes)) / math.Log(unit))
	prefixes := "KMGTPE"
	prefix := prefixes[exp-1]
	return fmt.Sprintf("%5.1f %cB", float64(bytes)/math.Pow(unit, float64(exp)), prefix)
}

func init() {
	AddMemoryReporter(`Go`, ServerGetMemoryUsage)
}
