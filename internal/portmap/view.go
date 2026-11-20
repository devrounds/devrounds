package portmap

// Row is one listening port, with the container publishing it, if any.
type Row struct {
	Entry
	Container string `json:"container,omitempty"`
}

// Report is everything map knows.
type Report struct {
	Ports      []Row        `json:"ports"`
	Containers []Container  `json:"containers"`
	Docker     bool         `json:"docker_available"`
	Project    *ProjectInfo `json:"project,omitempty"`
}

// Build joins listening ports with the containers that publish them.
func Build(entries []Entry, containers []Container, docker bool) Report {
	byPort := map[int]string{}
	for _, c := range containers {
		for _, p := range c.HostPorts {
			if _, ok := byPort[p]; !ok {
				byPort[p] = c.Name
			}
		}
	}
	rep := Report{Ports: []Row{}, Containers: []Container{}, Docker: docker}
	for _, e := range entries {
		rep.Ports = append(rep.Ports, Row{Entry: e, Container: byPort[e.Port]})
	}
	if containers != nil {
		rep.Containers = containers
	}
	return rep
}
