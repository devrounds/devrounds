package portmap

import (
	"sort"

	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// Entry is one listening TCP port and the process holding it.
type Entry struct {
	Port    int    `json:"port"`
	PID     int32  `json:"pid"`
	Process string `json:"process"`
	Folder  string `json:"folder"`
}

// Listening returns every listening TCP port with its process and start folder.
func Listening() ([]Entry, error) {
	conns, err := gnet.Connections("tcp")
	if err != nil {
		return nil, err
	}
	type key struct {
		port int
		pid  int32
	}
	seen := map[key]bool{}
	info := map[int32]Entry{}
	var out []Entry
	for _, c := range conns {
		if c.Status != "LISTEN" {
			continue
		}
		k := key{int(c.Laddr.Port), c.Pid}
		if seen[k] {
			continue
		}
		seen[k] = true
		e := Entry{Port: k.port, PID: c.Pid}
		if c.Pid > 0 {
			meta, ok := info[c.Pid]
			if !ok {
				if proc, err := process.NewProcess(c.Pid); err == nil {
					meta.Process, _ = proc.Name()
					meta.Folder, _ = proc.Cwd()
				}
				info[c.Pid] = meta
			}
			e.Process = meta.Process
			e.Folder = meta.Folder
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Port != out[j].Port {
			return out[i].Port < out[j].Port
		}
		return out[i].PID < out[j].PID
	})
	return out, nil
}
