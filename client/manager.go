package client

import (
	"net"
	"sync"

	log "github.com/cihub/seelog"
	"proxy/config"
	msg "proxy/message"
)

const (
	ProxyStatusNew     = 0 //"new"
	ProxyStatusRunning = 1 //"running"
	ProxyStatusClosed  = 2 //"closed"
)

type Manager struct {
	client       *Client
	allProxyConf []*config.ProxyConf
	proxies      map[string]*Proxy
	sendCh       chan (msg.Message)

	closed bool
	mu     sync.RWMutex
}

func NewManager(client *Client, proxy_conf []*config.ProxyConf, sendCh chan (msg.Message)) (m *Manager) {
	m = &Manager{
		client:       client,
		allProxyConf: proxy_conf,
		sendCh:       sendCh,
		proxies:      make(map[string]*Proxy),
		closed:       false,
	}

	for _, cfg := range proxy_conf {
		if _, ok := m.proxies[cfg.Name]; !ok {
			pxy := NewProxy(cfg)
			m.proxies[cfg.Name] = pxy
		}
	}
	return
}

func (m *Manager) sendMsg(m *msg.Message) {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
			m.closed = true
		}

	}()
	m.sendCh <- m
	return
}

func (m *Manager) CheckProxy() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, pxy := range m.proxies {
		if !IsRunning(pxy) {
			if pxy.Type == "extranet" {
				pxy.Run()
				continue
			}
			newProxyMsg := msg.NewProxy{
				ProxyName:  pxy.Name,
				ProxyType:  pxy.Type,
				RemotePort: pxy.RemotePort,
				Encrypt:    pxy.Encryption,
			}

			m, err := msg.Pack(msg.TypeNewProxy, newProxyMsg)
			if err != nil {
				log.Error(err)
				return
			}

			m.sendMsg(&m)
		}
	}
}

func IsRunning(pxy *Proxy) bool {
	if pxy == nil {
		return false
	}

	if pxy.Status == ProxyStatusRunning {
		return true
	} else {
		return false
	}
}

func (m *Manager) StartProxy(name string, remote_port int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		log.Error("start proxy error:Manager is closed")
		return
	}

	pxy, ok := m.proxies[name]
	if !ok {
		log.Error("start proxy error:no found proxy ", name)
		return
	}

	pxy.RemotePort = remote_port
	pxy.Run()
	return
}

func (m *Manager) ProxyWork(name string, conn net.Conn) {
	m.mu.RLock()
	pxy, ok := m.proxies[name]
	m.mu.RUnlock()
	if ok {
		pxy.Work(conn)
	} else {
		conn.Close()
	}
	return

}
