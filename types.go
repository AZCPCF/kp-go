package main

type Listener struct {
	Addr    string
	Port    int
	Process string
	PID     string
}

type Group struct {
	Process   string
	PID       string
	Listeners []Listener
}
