package scan

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

const sampleProcNetUDP = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode ref pointer drops
   0: 0100007F:14E9 00000000:0000 07 00000000:00000000 00:00000000 00000000  1000        0 22222 2 0000000000000000 0
   1: 00000000:0035 00000000:0000 07 00000000:00000000 00:00000000 00000000     0        0 33333 2 0000000000000000 0
   2: 0100007F:14E9 0100007F:14E9 01 00000000:00000000 00:00000000 00000000  1000        0 44444 2 0000000000000000 0
   3: garbage line
`

const sampleProcNetUDP6 = `  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode ref pointer drops
   0: 00000000000000000000000000000000:14E9 00000000000000000000000000000000:0000 07 00000000:00000000 00:00000000 00000000  1000        0 55555 2 0000000000000000 0
`

func TestParseProcNet(t *testing.T) {
	owners := map[string]owner{"12345": {"4242", "node"}}
	got := parseProcNet(sampleProcNet, owners, stateListenTCP, ProtoTCP)
	want := []Listener{
		{Addr: "127.0.0.1", Port: 8000, Proto: ProtoTCP, Process: "node", PID: "4242"},
		{Addr: "0.0.0.0", Port: 22, Proto: ProtoTCP, Process: "-", PID: "-"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestParseProcNetStateFilter(t *testing.T) {
	tests := []struct {
		name  string
		data  string
		state string
		proto string
		want  []Listener
	}{
		{
			name:  "tcp listens",
			data:  sampleProcNet,
			state: stateListenTCP,
			proto: ProtoTCP,
			want: []Listener{
				{Addr: "127.0.0.1", Port: 8000, Proto: ProtoTCP, Process: "-", PID: "-"},
				{Addr: "0.0.0.0", Port: 22, Proto: ProtoTCP, Process: "-", PID: "-"},
			},
		},
		{
			name:  "tcp data with udp state",
			data:  sampleProcNet,
			state: stateListenUDP,
			proto: ProtoUDP,
			want:  nil,
		},
		{
			name:  "udp listens",
			data:  sampleProcNetUDP,
			state: stateListenUDP,
			proto: ProtoUDP,
			want: []Listener{
				{Addr: "127.0.0.1", Port: 5353, Proto: ProtoUDP, Process: "-", PID: "-"},
				{Addr: "0.0.0.0", Port: 53, Proto: ProtoUDP, Process: "-", PID: "-"},
			},
		},
		{
			name:  "udp data with tcp state",
			data:  sampleProcNetUDP,
			state: stateListenTCP,
			proto: ProtoTCP,
			want:  nil,
		},
		{
			name:  "udp6 listens",
			data:  sampleProcNetUDP6,
			state: stateListenUDP,
			proto: ProtoUDP,
			want: []Listener{
				{Addr: "::", Port: 5353, Proto: ProtoUDP, Process: "-", PID: "-"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseProcNet(tt.data, nil, tt.state, tt.proto)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestParseProcNetOwners(t *testing.T) {
	owners := map[string]owner{"22222": {"777", "dnsmasq"}}
	got := parseProcNet(sampleProcNetUDP, owners, stateListenUDP, ProtoUDP)
	if len(got) == 0 {
		t.Fatal("expected listeners")
	}
	if got[0].Process != "dnsmasq" || got[0].PID != "777" {
		t.Errorf("owner not applied: %+v", got[0])
	}
	if got[1].Process != "-" || got[1].PID != "-" {
		t.Errorf("unknown owner should be a dash: %+v", got[1])
	}
}

func TestParseProcNetEmpty(t *testing.T) {
	if got := parseProcNet("", nil, stateListenTCP, ProtoTCP); len(got) != 0 {
		t.Errorf("expected empty, got %+v", got)
	}
}

func TestParseProto(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"default", "", ProtoTCP, false},
		{"tcp", "tcp", ProtoTCP, false},
		{"udp", "udp", ProtoUDP, false},
		{"all", "all", ProtoAll, false},
		{"uppercase", "TCP", "", true},
		{"unknown", "sctp", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseProto(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListInvalidProto(t *testing.T) {
	if _, err := List("sctp"); err == nil {
		t.Error("expected an error for an unknown proto")
	}
}

func TestSSArgs(t *testing.T) {
	tests := []struct {
		proto string
		want  []string
	}{
		{ProtoTCP, []string{"-tlnpH"}},
		{ProtoUDP, []string{"-ulnpH"}},
		{ProtoAll, []string{"-tulnpH"}},
	}
	for _, tt := range tests {
		if got := ssArgs(tt.proto); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.proto, got, tt.want)
		}
	}
}

const sampleSS = `LISTEN 0      511        127.0.0.1:59406 0.0.0.0:* users:(("code",pid=134347,fd=72))
LISTEN 0      32           10.0.3.1:53    0.0.0.0:*
LISTEN 0      128               [::]:22       [::]:*
garbage
LISTEN 0      1`

const sampleSSTU = `tcp LISTEN 0      511         127.0.0.1:59406 0.0.0.0:* users:(("code",pid=134347,fd=72))
udp UNCONN 0      0            10.0.3.1:53    0.0.0.0:*
udp UNCONN 0      0       127.0.0.53%lo:53    0.0.0.0:* users:(("dnsmasq",pid=900,fd=5))
`

func TestParseSS(t *testing.T) {
	tests := []struct {
		name  string
		data  string
		proto string
		want  []Listener
	}{
		{
			name:  "tcp only layout",
			data:  sampleSS,
			proto: ProtoTCP,
			want: []Listener{
				{Addr: "127.0.0.1", Port: 59406, Proto: ProtoTCP, Process: "code", PID: "134347"},
				{Addr: "10.0.3.1", Port: 53, Proto: ProtoTCP, Process: "-", PID: "-"},
				{Addr: "[::]", Port: 22, Proto: ProtoTCP, Process: "-", PID: "-"},
			},
		},
		{
			name:  "udp only layout",
			data:  sampleSS,
			proto: ProtoUDP,
			want: []Listener{
				{Addr: "127.0.0.1", Port: 59406, Proto: ProtoUDP, Process: "code", PID: "134347"},
				{Addr: "10.0.3.1", Port: 53, Proto: ProtoUDP, Process: "-", PID: "-"},
				{Addr: "[::]", Port: 22, Proto: ProtoUDP, Process: "-", PID: "-"},
			},
		},
		{
			name:  "combined layout has netid",
			data:  sampleSSTU,
			proto: ProtoAll,
			want: []Listener{
				{Addr: "127.0.0.1", Port: 59406, Proto: ProtoTCP, Process: "code", PID: "134347"},
				{Addr: "10.0.3.1", Port: 53, Proto: ProtoUDP, Process: "-", PID: "-"},
				{Addr: "127.0.0.53%lo", Port: 53, Proto: ProtoUDP, Process: "dnsmasq", PID: "900"},
			},
		},
		{
			name:  "empty",
			data:  "",
			proto: ProtoAll,
			want:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSS(tt.data, tt.proto)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v\nwant %+v", got, tt.want)
			}
		})
	}
}
