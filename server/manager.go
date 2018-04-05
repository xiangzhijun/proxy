package server

import (
	"fmt"
	"net"
	"sync"
)

type ClientManager struct {
	//map[clientID]client
	Client map[string]*ClientCtrl
}

func NewClientManager() (cm *ClientManager) {
	cm = &ClientManager{
		Client: make(map[string]*ClientCtrl),
	}
	return

}

func (cm *ClientManager) Add(clientId string, clientCtrl *ClientCtrl) {
	cm.Client[clientId] = clientCtrl
}

type ProxyManager struct {
	proxies map[string]Proxy
}

func NewProxyManager() (pm *ProxyManager) {
	pm = &ProxyManager{}
	return

}

type PortManager struct {
	usedPorts map[int]string
	freePorts map[int]string
	Min       int
	Max       int
	BindIP    string
	mu        sync.Mutex
}

func NewPortManager(min, max int, bind_ip string) *PortManager {
	if min > max || min < 3000 {
		min = 10000
		max = 11000
	}
	free_ports := make(map[int]string, max-min+1)
	for i := min; i <= max; i++ {
		free_ports[i] = "f"
	}

	pm := &PortManager{
		usedPorts: make(map[int]string),
		freePorts: free_ports,
		Min:       min,
		Max:       max,
		BindIP:    bind_ip,
	}
	return pm
}

func (pm *PortManager) Get(name string, remote_port int) int {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if remote_port >= pm.Min && remote_port <= pm.Max {
		if _, ok := pm.freePorts[remote_port]; ok {
			if pm.PortTest(remote_port) {
				pm.usedPorts[remote_port] = name
				delete(pm.freePorts, remote_port)
				return remote_port
			}
		}
	}

	tryTimes := 5
	count := 0
	for p, _ := range pm.freePorts {
		if count > tryTimes {
			break
		}
		count++
		if pm.PortTest(p) {
			pm.usedPorts[p] = name
			delete(pm.freePorts, p)
			return p
		}

	}
	return 0

}

func (pm *PortManager) PortTest(port int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", pm.BindIP, port))
	if err != nil {
		return false
	}
	l.Close()
	return true
}

func (pm *PortManager) Release(port int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if _, ok := pm.usedPorts[port]; ok {
		delete(pm.usedPorts, port)
		pm.freePorts[port] = "free"
	}
}
