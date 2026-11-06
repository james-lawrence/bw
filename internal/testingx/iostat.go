package testingx

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

// IOCounters is a snapshot of the process wide I/O counters of the kernel (/proc/self/io). only available on linux,
// the zero value is reported elsewhere.
type IOCounters struct {
	ReadChars  uint64 // rchar: bytes passed to read-like syscalls.
	WriteChars uint64 // wchar: bytes passed to write-like syscalls.
	ReadCalls  uint64 // syscr: read syscalls.
	WriteCalls uint64 // syscw: write syscalls.
	WriteBytes uint64 // write_bytes: bytes sent to the storage layer.
}

// ReadIOCounters reads the current counters.
func ReadIOCounters() (c IOCounters) {
	f, err := os.Open("/proc/self/io")
	if err != nil {
		return c
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		k, v, ok := strings.Cut(s.Text(), ":")
		if !ok {
			continue
		}

		n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
		if err != nil {
			continue
		}

		switch k {
		case "rchar":
			c.ReadChars = n
		case "wchar":
			c.WriteChars = n
		case "syscr":
			c.ReadCalls = n
		case "syscw":
			c.WriteCalls = n
		case "write_bytes":
			c.WriteBytes = n
		}
	}

	return c
}

// Sub returns the difference between two snapshots.
func (t IOCounters) Sub(o IOCounters) IOCounters {
	return IOCounters{
		ReadChars:  t.ReadChars - o.ReadChars,
		WriteChars: t.WriteChars - o.WriteChars,
		ReadCalls:  t.ReadCalls - o.ReadCalls,
		WriteCalls: t.WriteCalls - o.WriteCalls,
		WriteBytes: t.WriteBytes - o.WriteBytes,
	}
}

// BenchIO measures the I/O of the benchmark, call the returned function when done to report the counters
// per unit of work (b.N unless n is given). the counters are for the whole process, so benchmarks using this
// should not run other work concurrently.
func BenchIO(b *testing.B) (report func(units int)) {
	b.Helper()

	start := ReadIOCounters()

	return func(units int) {
		b.Helper()

		if units <= 0 {
			units = b.N
		}

		d := ReadIOCounters().Sub(start)
		u := float64(units)
		b.ReportMetric(float64(d.WriteCalls)/u, "write-syscalls/op")
		b.ReportMetric(float64(d.WriteChars)/u, "wchar-B/op")
		b.ReportMetric(float64(d.WriteBytes)/u, "storage-write-B/op")
		b.ReportMetric(float64(d.ReadCalls)/u, "read-syscalls/op")
		b.ReportMetric(float64(d.ReadChars)/u, "rchar-B/op")
	}
}
