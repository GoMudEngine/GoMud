package util

import (
	"fmt"
	"math"
	"reflect"
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

func sizeOf(v reflect.Value) uintptr {
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return 0
		}
		return sizeOf(v.Elem())

	case reflect.Slice:
		if v.IsNil() {
			return 0
		}
		length := v.Len()
		elemSize := sizeOf(reflect.New(v.Type().Elem()).Elem())
		return uintptr(length) * elemSize

	case reflect.Struct:
		var size uintptr
		for i := 0; i < v.NumField(); i++ {
			size += sizeOf(v.Field(i))
		}
		return size

	case reflect.Array:
		length := v.Len()
		elemSize := sizeOf(reflect.New(v.Type().Elem()).Elem())
		return uintptr(length) * elemSize

	case reflect.String:
		return uintptr(len(v.String()))

	case reflect.Map:
		var size uintptr
		keys := v.MapKeys()
		for _, key := range keys {
			size += sizeOf(key) + sizeOf(v.MapIndex(key))
		}
		return size

	default:
		return v.Type().Size()
	}
}

func MemoryUsage(i interface{}) uint64 {
	return uint64(sizeOf(reflect.ValueOf(i)))
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
