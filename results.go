package main

type EchoResult string

func (r EchoResult) LastOne() bool { return true }

type StreamResult struct {
	Value int
	Last  bool
}

func (r StreamResult) LastOne() bool { return r.Last }
