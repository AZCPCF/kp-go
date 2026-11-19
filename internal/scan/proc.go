package scan

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Socket states in /proc/net/{tcp,udp}: TCP LISTEN is 0A, an unconnected UDP
// socket (the listening equivalent) is 07.
const (
	stateListenTCP = "0A"
	stateListenUDP = "07"
)

type owner struct {
	pid  string
	comm string
}

type procFile struct {
	path  string
	state string
	proto string
}

var procFiles = []procFile{
	{"/proc/net/tcp", stateListenTCP, ProtoTCP},
	{"/proc/net/tcp6", stateListenTCP, ProtoTCP},
	{"/proc/net/udp", stateListenUDP, ProtoUDP},
	{"/proc/net/udp6", stateListenUDP, ProtoUDP},
}

func parseAddr(s string) (string, int, error) {
	i := strings.IndexByte(s, ':')
	if i < 0 {
		return "", 0, fmt.Errorf("bad address %q", s)
	}
	port, err := strconv.ParseUint(s[i+1:], 16, 16)
	if err != nil {
		return "", 0, err
	}
	raw, err := hex.DecodeString(s[:i])
	if err != nil {
		return "", 0, err
	}
	if len(raw) != 4 && len(raw) != 16 {
		return "", 0, fmt.Errorf("bad ip length %d", len(raw))
	}
	for off := 0; off < len(raw); off += 4 {
		raw[off], raw[off+1], raw[off+2], raw[off+3] = raw[off+3], raw[off+2], raw[off+1], raw[off]
	}
	return net.IP(raw).String(), int(port), nil
}

func socketOwners() map[string]owner {
	res := map[string]owner{}
	comms := map[string]string{}
	paths, _ := filepath.Glob("/proc/[0-9]*/fd/*")
	for _, p := range paths {
		link, err := os.Readlink(p)
		if err != nil || !strings.HasPrefix(link, "socket:[") {
			continue
		}
		inode := strings.TrimSuffix(strings.TrimPrefix(link, "socket:["), "]")
		if _, ok := res[inode]; ok {
			continue
		}
		pid := strings.Split(p, "/")[2]
		comm, ok := comms[pid]
		if !ok {
			b, err := os.ReadFile("/proc/" + pid + "/comm")
			comm = "-"
			if err == nil {
				comm = strings.TrimSpace(string(b))
			}
			comms[pid] = comm
		}
		res[inode] = owner{pid, comm}
	}
	return res
}

func listListenersProc(proto string) ([]Listener, error) {
	owners := socketOwners()
	var res []Listener
	for _, f := range procFiles {
		if proto != ProtoAll && proto != f.proto {
			continue
		}
		data, err := os.ReadFile(f.path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}
		res = append(res, parseProcNet(string(data), owners, f.state, f.proto)...)
	}
	return res, nil
}

func parseProcNet(data string, owners map[string]owner, state, proto string) []Listener {
	var res []Listener
	lines := strings.Split(data, "\n")
	for _, line := range lines[1:] {
		f := strings.Fields(line)
		if len(f) < 10 || f[3] != state {
			continue
		}
		addr, port, err := parseAddr(f[1])
		if err != nil {
			continue
		}
		l := Listener{Addr: addr, Port: port, Proto: proto, Process: "-", PID: "-"}
		if o, ok := owners[f[9]]; ok {
			l.Process, l.PID = o.comm, o.pid
		}
		res = append(res, l)
	}
	return res
}
