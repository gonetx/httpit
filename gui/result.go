package gui

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/valyala/bytebufferpool"
)

type Result struct {
	throughput int64
	reqs       int64
	elapsed    time.Duration
	code1xx    int64
	code2xx    int64
	code3xx    int64
	code4xx    int64
	code5xx    int64
	codeOthers int64
	latencies  []int64
	rps        []float64
	errs       map[string]int
	output     io.Writer
	buf        *bytebufferpool.ByteBuffer
}

func NewResult(output io.Writer) *Result {
	return &Result{
		errs:   make(map[string]int),
		output: output,
		buf:    bytebufferpool.Get(),
	}
}

func (r *Result) Throughput() *int64 {
	return &r.throughput
}

func (r *Result) TotalReqs() int64 {
	return r.reqs
}

func (r *Result) Elapsed() time.Duration {
	return r.elapsed
}

func (r *Result) AppendCode(code int) {
	switch code / 100 {
	case 1:
		r.code1xx++
	case 2:
		r.code2xx++
	case 3:
		r.code3xx++
	case 4:
		r.code4xx++
	case 5:
		r.code5xx++
	default:
		r.codeOthers++
	}
}

func (r *Result) IncreaseReq() {
	r.reqs++
}

func (r *Result) AddElapsed(elapsed time.Duration) {
	r.elapsed += elapsed
}

func (r *Result) AppendRps(rps float64) {
	r.rps = append(r.rps, rps)
}

func (r *Result) AppendLatency(latency time.Duration) {
	r.latencies = append(r.latencies, latency.Microseconds())
}

func (r *Result) AppendError(err error) {
	r.errs[err.Error()]++
}

func (r *Result) Print() {
	r.writeTotalRequest()
	r.writeElapsed()
	r.writeStatistics()
	r.writeCodes()
	r.writeThroughput()
	r.writeErrors()

	_, _ = r.buf.WriteTo(r.output)
}

func (r *Result) writeTotalRequest() {
	_, _ = r.buf.WriteString(fmt.Sprintf("Total requests: %d\n", r.reqs))
}

func (r *Result) writeElapsed() {
	_, _ = r.buf.WriteString(fmt.Sprintf("Elapsed: %.2fs\n", r.elapsed.Seconds()))
}

func (r *Result) writeStatistics() {
	_, _ = r.buf.WriteString("Statistics        Avg        Stdev        Max\n")

	rpsAvg, rpsStdev, rpsMax := rpsResult(r.rps)
	_, _ = r.buf.WriteString(fmt.Sprintf("    Reqs/sec    %.2f     %.2f     %.2f\n", rpsAvg, rpsStdev, rpsMax))
	latencyAvg, latencyStdev, latencyMax := latencyResult(r.latencies)
	_, _ = r.buf.WriteString(fmt.Sprintf("    Latency    %.2fms        %.2fms        %.2fms\n", latencyAvg, latencyStdev, latencyMax))
}

func (r *Result) writeCodes() {
	_, _ = r.buf.WriteString("HTTP codes:\n")
	_, _ = r.buf.WriteString(fmt.Sprintf("    1xx - %d, ", r.code1xx))
	_, _ = r.buf.WriteString(fmt.Sprintf("2xx - %d, ", r.code2xx))
	_, _ = r.buf.WriteString(fmt.Sprintf("3xx - %d, ", r.code3xx))
	_, _ = r.buf.WriteString(fmt.Sprintf("4xx - %d, ", r.code4xx))
	_, _ = r.buf.WriteString(fmt.Sprintf("5xx - %d\n", r.code5xx))
	_, _ = r.buf.WriteString(fmt.Sprintf("    Others - %d\n", r.code5xx))
}

func (r *Result) writeThroughput() {
	_, _ = r.buf.WriteString("Throughput: ")
	_, _ = r.buf.WriteString(formatThroughput(float64(r.throughput) / r.elapsed.Seconds()))
	_, _ = r.buf.WriteString("\n")
}

func (r *Result) writeErrors() {
	if len(r.errs) == 0 {
		return
	}
	_, _ = r.buf.WriteString("Errors:\n")
	for err, count := range r.errs {
		_, _ = r.buf.WriteString(fmt.Sprintf("\t%s: %d\n", err, count))
	}
}

func rpsResult(rps []float64) (avg float64, stdev float64, max float64) {
	var sum, sum2 float64
	for _, r := range rps {
		sum += r
		if r > max {
			max = r
		}
	}

	avg = sum / float64(len(rps))

	var diff float64
	for _, r := range rps {
		diff = avg - r
		sum2 += diff * diff
	}

	stdev = math.Sqrt(sum2 / float64(len(rps)-1))

	return
}

func latencyResult(latencies []int64) (avg float64, stdev float64, max float64) {
	var sum float64
	for _, latency := range latencies {
		// us -> ms
		r := float64(latency) / 1000
		sum += r
		if r > max {
			max = r
		}
	}

	avg = sum / float64(len(latencies))

	var diff, sum2 float64
	for _, latency := range latencies {
		// us -> ms
		r := float64(latency) / 1000
		diff = avg - r
		sum2 += diff * diff
	}

	stdev = math.Sqrt(sum2 / float64(len(latencies)-1))

	return
}

func formatThroughput(throughput float64) string {
	switch {
	case throughput < 1e3:
		return fmt.Sprintf("%.2f B/s", throughput)
	case throughput < 1e6:
		return fmt.Sprintf("%.2f KB/s", throughput/1e3)
	case throughput < 1e9:
		return fmt.Sprintf("%.2f MB/s", throughput/1e6)
	default:
		return fmt.Sprintf("%.2f GB/s", throughput/1e12)
	}
}
