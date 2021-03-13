package pit

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stretchr/testify/assert"
)

func Test_tui_appendCode(t *testing.T) {
	t.Parallel()

	tt := newTui()
	tt.appendCode(101)
	tt.appendCode(201)
	tt.appendCode(301)
	tt.appendCode(401)
	tt.appendCode(501)
	tt.appendCode(601)

	assert.Equal(t, int64(1), tt.code1xx)
	assert.Equal(t, int64(1), tt.code2xx)
	assert.Equal(t, int64(1), tt.code3xx)
	assert.Equal(t, int64(1), tt.code4xx)
	assert.Equal(t, int64(1), tt.code5xx)
	assert.Equal(t, int64(1), tt.codeOthers)
}

func Test_tui_writeProcessBar(t *testing.T) {
	t.Parallel()

	tt := newTui()
	tt.count = 1
	tt.reqs = 2
	tt.writeProcessBar()
	assert.Contains(t, tt.buf.String(), "100%")
}

func Test_tui_writeTotalRequest(t *testing.T) {
	t.Parallel()

	tt := newTui()
	tt.count = 3
	tt.reqs = 2
	tt.writeTotalRequest()
	assert.Contains(t, tt.buf.String(), "2/3")
}

func Test_tui_writeElapsed(t *testing.T) {
	t.Parallel()

	tt := newTui()
	tt.duration = time.Second
	tt.elapsed = int64(time.Second * 2)
	tt.writeElapsed()
	assert.Contains(t, tt.buf.String(), "1.00/1.00")
}

func Test_tui_writeThroughput(t *testing.T) {
	t.Parallel()

	var throughput int64 = 1001
	tt := newTui()
	tt.throughput = &throughput
	tt.elapsed = int64(time.Second)
	tt.writeThroughput()
	assert.Contains(t, tt.buf.String(), "1.00 KB/s")
}

func Test_tui_writeErrors(t *testing.T) {
	t.Parallel()

	tt := newTui()
	tt.errs["custom-error"] = 1
	tt.writeErrors()
	assert.Contains(t, tt.buf.String(), "custom-error")
	assert.Contains(t, tt.buf.String(), "1")
}

//func Test_tui_writeHint(t *testing.T) {
//	t.Parallel()
//
//	t.Run("done", func(t *testing.T) {
//		tt := newTui()
//		tt.done = true
//		tt.writeHint()
//		assert.Contains(t, tt.buf.String(), "Done")
//	})
//
//	t.Run("terminate", func(t *testing.T) {
//		tt := newTui()
//		tt.quitting = true
//		tt.writeHint()
//		assert.Contains(t, tt.buf.String(), "Terminated")
//	})
//}

func Test_tui_writeInt(t *testing.T) {
	t.Parallel()

	tt := newTui()
	tt.writeInt(12, "#444")
	assert.Contains(t, tt.buf.String(), "12")
}

func Test_rpsResult(t *testing.T) {
	t.Parallel()

	avg, stdev, max := rpsResult([]float64{1, 6, 5, 7, 9, 8})
	assert.Equal(t, 6.0, avg)
	assert.Equal(t, 2.8284271247461903, stdev)
	assert.Equal(t, 9.0, max)
}

func Test_latencyResult(t *testing.T) {
	t.Parallel()

	avg, stdev, max := latencyResult([]int64{1e6, 6e6, 5e6, 7e6, 9e6, 8e6})
	assert.Equal(t, 6.0e3, avg)
	assert.Equal(t, 2828.42712474619, stdev)
	assert.Equal(t, 9.0e3, max)
}

func Test_formatThroughput(t *testing.T) {
	t.Parallel()

	v, u := formatThroughput(100)
	assert.Equal(t, 100.0, v)
	assert.Equal(t, "B/s", u)

	v, u = formatThroughput(1001)
	assert.Equal(t, 1.001, v)
	assert.Equal(t, "KB/s", u)

	v, u = formatThroughput(1111111)
	assert.Equal(t, 1.111111, v)
	assert.Equal(t, "MB/s", u)

	v, u = formatThroughput(1111111111)
	assert.Equal(t, 1.111111111, v)
	assert.Equal(t, "GB/s", u)
}

func Test_tui_Update(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		initCmd tea.Cmd
	}{
		{"press esc", func() tea.Msg { return tea.KeyMsg{Type: tea.KeyEsc} }},
		{"press q", func() tea.Msg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}} }},
		{"press ctrl + c", func() tea.Msg { return tea.KeyMsg{Type: tea.KeyCtrlC} }},
		{"done", func() tea.Msg { return done }},
		{"skip normal key", tea.Batch(
			func() tea.Msg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}} },
			func() tea.Msg { return done },
		)},
		{"windows size msg", tea.Batch(
			func() tea.Msg { return tea.WindowSizeMsg{Height: 10, Width: 100} },
			func() tea.Msg { return done },
		)},
		//{"tick fps", tea.Batch(
		//	func() tea.Msg { return tick() },
		//	func() tea.Msg {
		//		time.Sleep(time.Millisecond * 100)
		//		return done
		//	},
		//)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tt := newTui()
			tt.count = 1
			tt.initCmd = tc.initCmd

			err := tea.NewProgram(tt, tea.WithInput(os.Stdin), tea.WithOutput(ioutil.Discard)).Start()
			assert.Nil(t, err)
		})
	}
}
