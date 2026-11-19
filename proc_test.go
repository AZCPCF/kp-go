package main

import (
	"reflect"
	"testing"
)

func TestParseAddr(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		wantIP   string
		wantPort int
		wantErr  bool
	}{
		{"ipv4 loopback", "0100007F:1F40", "127.0.0.1", 8000, false},
		{"ipv4 any", "00000000:0050", "0.0.0.0", 80, false},
		{"ipv6 loopback", "00000000000000000000000001000000:1F90", "::1", 8080, false},
		{"ipv6 any", "00000000000000000000000000000000:0016", "::", 22, false},
		{"no colon", "0100007F", "", 0, true},
		{"bad hex ip", "ZZZZZZZZ:1F40", "", 0, true},
		{"bad port", "0100007F:ZZZZ", "", 0, true},
		{"port overflow", "0100007F:10000", "", 0, true},
		{"bad ip length", "0100:1F40", "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, port, err := parseAddr(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if ip != tt.wantIP || port != tt.wantPort {
				t.Errorf("got %s:%d, want %s:%d", ip, port, tt.wantIP, tt.wantPort)
			}
		})
	}
}

const sampleProcNet = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1F40 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 12345 1 0000000000000000 100 0 0 10 0
   1: 0100007F:0050 0100007F:9C40 01 00000000:00000000 00:00000000 00000000  1000        0 99999 1 0000000000000000 100 0 0 10 0
   2: 00000000:0016 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 55555 1 0000000000000000 100 0 0 10 0
   3: garbage line
`

func TestParseProcNet(t *testing.T) {
	owners := map[string]owner{"12345": {"4242", "node"}}
	got := parseProcNet(sampleProcNet, owners)
	want := []Listener{
		{Addr: "127.0.0.1", Port: 8000, Process: "node", PID: "4242"},
		{Addr: "0.0.0.0", Port: 22, Process: "-", PID: "-"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestParseProcNetEmpty(t *testing.T) {
	if got := parseProcNet("", nil); len(got) != 0 {
		t.Errorf("expected empty, got %+v", got)
	}
}
