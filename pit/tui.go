package pit

import (
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/margin"
	"github.com/muesli/termenv"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
)

var color = termenv.ColorProfile().Color

const done = 1
const fieldWidth = 20

type tui struct {
	throughput int64
	reqs       int64
	elapsed    int64
	code1xx    int64
	code2xx    int64
	code3xx    int64
	code4xx    int64
	code5xx    int64
	codeOthers int64
	latencies  []int64
	rps        []float64
	mut        sync.Mutex
	errs       map[string]int
	buf        *bytebufferpool.ByteBuffer

	url         string
	count       int
	duration    time.Duration
	connections int
	initCmd     tea.Cmd
	spinner     spinner.Model
	quitting    bool
	done        bool
}

func newTui() *tui {
	s := spinner.NewModel()
	s.Spinner = spinner.Dot

	return &tui{
		errs:    make(map[string]int),
		buf:     bytebufferpool.Get(),
		spinner: s,
	}
}

func (t *tui) start(url string) error {
	t.url = url
	return tea.NewProgram(t).Start()
}

func (t *tui) Init() tea.Cmd {
	return tea.Batch(spinner.Tick, t.initCmd)
}

func (t *tui) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			fallthrough
		case "esc":
			fallthrough
		case "ctrl+c":
			t.quitting = true
			return t, tea.Quit
		default:
			return t, nil
		}

	case int:
		var cmd tea.Cmd
		if msg == done {
			t.done = true
			cmd = tea.Quit
		}
		return t, cmd

	default:
		var cmd tea.Cmd
		t.spinner, cmd = t.spinner.Update(msg)
		return t, cmd
	}

}

func (t *tui) View() string {
	return t.output()
}

func (t *tui) appendCode(code int) {
	switch code / 100 {
	case 1:
		t.code1xx++
	case 2:
		t.code2xx++
	case 3:
		t.code3xx++
	case 4:
		t.code4xx++
	case 5:
		t.code5xx++
	default:
		t.codeOthers++
	}
}

func (t *tui) appendRps(rps float64) {
	t.rps = append(t.rps, rps)
}

func (t *tui) appendLatency(latency time.Duration) {
	t.latencies = append(t.latencies, latency.Microseconds())
}

func (t *tui) appendError(err error) {
	t.mut.Lock()
	t.errs[err.Error()]++
	t.mut.Unlock()
}

func (t *tui) output() string {
	t.buf.Reset()

	t.writeTitle()
	t.writeTotalRequest()
	t.writeElapsed()
	t.writeThroughput()
	t.writeStatistics()
	t.writeCodes()
	t.writeErrors()
	t.writeHint()

	return t.buf.String()
}

func (t *tui) writeTitle() {
	dot := termenv.String(t.spinner.View()).Foreground(color("205")).String()
	_, _ = t.buf.WriteString(dot)
	_, _ = t.buf.WriteString("Benchmarking ")
	_, _ = t.buf.WriteString(t.url)
	_, _ = t.buf.WriteString(" with ")
	t.writeInt(t.connections)
	_, _ = t.buf.WriteString(" connections\n")
}

func (t *tui) writeTotalRequest() {
	_, _ = t.buf.WriteString("Requests:  ")
	t.writeInt(int(atomic.LoadInt64(&t.reqs)))
	if t.count != 0 {
		_ = t.buf.WriteByte('/')
		t.writeInt(t.count)
	}
	_, _ = t.buf.WriteString("  ")
}

func (t *tui) writeElapsed() {
	elapsed := time.Duration(atomic.LoadInt64(&t.elapsed))
	_, _ = t.buf.WriteString("Elapsed:  ")
	t.writeFloat(elapsed.Seconds())
	if t.count == 0 {
		_ = t.buf.WriteByte('/')
		t.writeFloat(t.duration.Seconds())
	}
	_, _ = t.buf.WriteString("s  ")
}

func (t *tui) writeThroughput() {
	_, _ = t.buf.WriteString("Throughput:  ")
	elapsed := time.Duration(atomic.LoadInt64(&t.elapsed))
	if seconds := elapsed.Seconds(); seconds != 0 {
		throughput, unit := formatThroughput(float64(t.throughput) / seconds)
		t.writeFloat(throughput)
		_ = t.buf.WriteByte(' ')
		_, _ = t.buf.WriteString(unit)
	} else {
		_, _ = t.buf.WriteString("0 B/s")
	}
	_ = t.buf.WriteByte('\n')
}

func (t *tui) writeStatistics() {
	_, _ = t.buf.Write([]byte("Statistics"))
	_, _ = t.buf.Write(margin.Bytes([]byte("Avg"), fieldWidth, 8))
	_, _ = t.buf.Write(margin.Bytes([]byte("Stdev"), fieldWidth, 7))
	_, _ = t.buf.Write(margin.Bytes([]byte("Max\n"), fieldWidth, 8))

	rpsAvg, rpsStdev, rpsMax := rpsResult(t.rps)
	_, _ = t.buf.Write([]byte("  Reqs/sec"))
	t.writeRps(rpsAvg)
	t.writeRps(rpsStdev)
	t.writeRps(rpsMax)
	_ = t.buf.WriteByte('\n')

	latencyAvg, latencyStdev, latencyMax := latencyResult(t.latencies)
	_, _ = t.buf.Write([]byte("  Latency "))
	t.writeLatency(latencyAvg)
	t.writeLatency(latencyStdev)
	t.writeLatency(latencyMax)
	_ = t.buf.WriteByte('\n')
}

func (t *tui) writeRps(rps float64) {
	b := strconv.AppendFloat(nil, rps, 'f', 2, 64)
	_, _ = t.buf.Write(margin.Bytes(b, fieldWidth, (fieldWidth-uint(len(b)))/2))
}

func (t *tui) writeLatency(latency float64) {
	b := strconv.AppendFloat(nil, latency, 'f', 2, 64)
	b = append(b, 'm', 's')
	_, _ = t.buf.Write(margin.Bytes(b, fieldWidth, (fieldWidth-uint(len(b)))/2))
}

func (t *tui) writeCodes() {
	_, _ = t.buf.WriteString("HTTP codes:\n  ")

	_, _ = t.buf.WriteString(termenv.String("1xx").Foreground(color("#ffaf00")).String())
	_, _ = t.buf.WriteString(" - ")
	t.writeInt(int(atomic.LoadInt64(&t.code1xx)))
	_, _ = t.buf.WriteString(", ")

	_, _ = t.buf.WriteString(termenv.String("2xx").Foreground(color("#00ff00")).String())
	_, _ = t.buf.WriteString(" - ")
	t.writeInt(int(atomic.LoadInt64(&t.code2xx)))
	_, _ = t.buf.WriteString(", ")

	_, _ = t.buf.WriteString(termenv.String("3xx").Foreground(color("#ffff00")).String())
	_, _ = t.buf.WriteString(" - ")
	t.writeInt(int(atomic.LoadInt64(&t.code3xx)))
	_, _ = t.buf.WriteString(", ")

	_, _ = t.buf.WriteString(termenv.String("4xx").Foreground(color("#ff8700")).String())
	_, _ = t.buf.WriteString(" - ")
	t.writeInt(int(atomic.LoadInt64(&t.code4xx)))
	_, _ = t.buf.WriteString(", ")

	_, _ = t.buf.WriteString(termenv.String("5xx").Foreground(color("#ff0000")).String())
	_, _ = t.buf.WriteString(" - ")
	t.writeInt(int(atomic.LoadInt64(&t.code5xx)))
	_, _ = t.buf.WriteString("\n  ")

	_, _ = t.buf.WriteString(termenv.String("Others").Foreground(color("#808080")).String())
	_, _ = t.buf.WriteString(" - ")
	t.writeInt(int(atomic.LoadInt64(&t.codeOthers)))
	_, _ = t.buf.WriteString("\n")
}

func (t *tui) writeErrors() {
	t.mut.Lock()
	defer t.mut.Unlock()

	if len(t.errs) == 0 {
		return
	}
	_, _ = t.buf.WriteString("Errors:\n")
	for err, count := range t.errs {
		_, _ = t.buf.WriteString("  ")
		_, _ = t.buf.WriteString(termenv.String(err).Underline().String())
		_, _ = t.buf.WriteString(": ")
		t.writeInt(count)
		_ = t.buf.WriteByte('\n')
	}
}

func (t *tui) writeHint() {
	if t.done {
		_, _ = t.buf.WriteString(termenv.String(" Done! \n").Background(color("#008700")).String())
	} else if t.quitting {
		_, _ = t.buf.WriteString(termenv.String(" Terminated! \n").Background(color("#870000")).String())
	} else {
		_, _ = t.buf.WriteString(termenv.String(" press q/esc/ctrl+c to quit \n").Background(color("#444")).String())
	}
}

func (t *tui) writeInt(i int) {
	t.buf.B = fasthttp.AppendUint(t.buf.B, i)
}

func (t *tui) writeFloat(f float64) {
	t.buf.B = strconv.AppendFloat(t.buf.B, f, 'f', 2, 64)
}

func rpsResult(rps []float64) (avg float64, stdev float64, max float64) {
	l := len(rps)
	if l == 0 {
		return
	}

	var sum, sum2 float64
	for _, r := range rps {
		sum += r
		if r > max {
			max = r
		}
	}

	avg = sum / float64(l)

	var diff float64
	for _, r := range rps {
		diff = avg - r
		sum2 += diff * diff
	}

	stdev = math.Sqrt(sum2 / float64(l-1))

	return
}

func latencyResult(latencies []int64) (avg float64, stdev float64, max float64) {
	l := len(latencies)
	if l == 0 {
		return
	}

	var sum float64
	for _, latency := range latencies {
		// us -> ms
		r := float64(latency) / 1000
		sum += r
		if r > max {
			max = r
		}
	}

	avg = sum / float64(l)

	var diff, sum2 float64
	for _, latency := range latencies {
		// us -> ms
		r := float64(latency) / 1000
		diff = avg - r
		sum2 += diff * diff
	}

	stdev = math.Sqrt(sum2 / float64(l-1))

	return
}

func formatThroughput(throughput float64) (float64, string) {
	switch {
	case throughput < 1e3:
		return throughput, "B/s"
	case throughput < 1e6:
		return throughput / 1e3, "KB/s"
	case throughput < 1e9:
		return throughput / 1e6, "MB/s"
	default:
		return throughput / 1e12, "GB/s"
	}
}
