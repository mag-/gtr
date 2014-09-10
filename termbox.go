package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/nsf/termbox-go"
)

// Print a string on termbox
func Print(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x++
	}
}

// Printf a string on termbox
func Printf(x, y int, fg, bg termbox.Attribute, format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	Print(x, y, fg, bg, s)
}

// TraceRow containst statistics about a host
type TraceRow struct {
	Host    string
	MaxTime time.Duration
	MinTime time.Duration
	AvgTime time.Duration
	TTL     int
	count   int
	sync.RWMutex
}

func newTraceRow(host string, t time.Duration, ttl int) *TraceRow {
	return &TraceRow{
		Host:    host,
		MaxTime: t,
		MinTime: t,
		AvgTime: t,
		TTL:     ttl,
		count:   1,
	}
}

// Print row on termbox
func (row *TraceRow) Print() {
	row.RLock()
	Printf(3, 2, termbox.ColorDefault, termbox.ColorDefault, "%-3v %-60v %-12v %-12v %-12v",
		"TTL", "Host", "Avg", "Min", "Max")
	Printf(3, 2+row.TTL, termbox.ColorDefault, termbox.ColorDefault, "%-3d %-60v %-12v %-12v %-12v",
		row.TTL, row.Host, row.AvgTime, row.MinTime, row.MaxTime)
	row.RUnlock()
	termbox.Flush()
}

// Update row with new data
func (row *TraceRow) Update(host string, t time.Duration, ttl int) {
	row.Lock()
	row.Host = host
	if t > row.MaxTime {
		row.MaxTime = t
	}
	if t < row.MinTime {
		row.MinTime = t
	}
	row.AvgTime = (row.AvgTime*time.Duration(row.count) + t) / time.Duration(row.count+1)
	row.count = row.count + 1
	row.Unlock()
}
