package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"

	"github.com/nsf/termbox-go"
)

func printUpdate(update TraceUpdate, rows map[int]*TraceRow) {
	addr := fmt.Sprintf("%v.%v.%v.%v", update.Address[0], update.Address[1], update.Address[2], update.Address[3])
	hostOrAddr := addr
	if update.Host != "" {
		hostOrAddr = update.Host
	}
	row, ok := rows[update.TTL]
	if !ok {
		row = newTraceRow(hostOrAddr, update.ElapsedTime, update.TTL)
		rows[update.TTL] = row
	} else {
		row.Update(hostOrAddr, update.ElapsedTime, update.TTL)
	}

	if update.Success {
		row.Print()
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	options := NewTracerouteOptions()
	flag.IntVar(&options.MaxTTL, "ttl", 64, "Set max ttl")
	flag.Parse()
	host := flag.Arg(0)

	ipAddr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		log.Fatal("Failed to resolve host: ", host)
	}

	// initialize termbox
	err = termbox.Init()
	if err != nil {
		log.Fatal("Could not start termbox, termbox.Init() gave an error:\n%s\n", err)
		os.Exit(1)
	}
	termbox.HideCursor()
	termbox.Clear(termbox.ColorBlack, termbox.ColorBlack)
	defer termbox.Close()

	c := make(chan TraceUpdate, 0)
	go func() {
		rows := make(map[int]*TraceRow)
		for {
			update, ok := <-c
			if !ok {
				fmt.Println()
				return
			}
			printUpdate(update, rows)
		}
	}()
	go func() {
		err = Traceroute(ipAddr, options, c)
		if err != nil {
			log.Fatal("Error: ", err)
		}
	}()

	// make chan for termbox events and run poller to send events on chan
	eventChan := make(chan termbox.Event)
	go func() {
		for {
			event := termbox.PollEvent()
			eventChan <- event
		}
	}()
	// register signals to channel
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, os.Kill)
	// handle termbox events and unix signals
	func() {
		for {
			// select for either event or signal
			select {
			case event := <-eventChan:
				switch event.Type {
				case termbox.EventKey: // actions depend on key
					switch event.Key {
					case termbox.KeyCtrlZ, termbox.KeyCtrlC:
						return
					}
				case termbox.EventError: // quit
					log.Fatalf("Quitting because of termbox error: \n%s\n", event.Err)
				}
			case signal := <-sigChan:
				log.Printf("Got signal:", signal)
				os.Exit(0)
			}
		}
	}()
}
