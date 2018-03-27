package client

import (
	"sync"

	"proxy/config"
	msg "proxy/message"
)

type Manager struct {
	client       *Client
	allProxyConf []*config.ProxyConf
	proxies      map[string]*proxy
	sendCh       chan (msg.Message)

	mu sync.RWMutex
}

func NewManager(client *Client, proxy_conf []*config.ProxyConf, sendCh chan (msg.Message)) (m *Manager) {
	m = &Manager{
		client:       client,
		allProxyConf: proxy_conf,
		sendCh:       sendCh,
		proxies:      make(map[string]*proxy),
	}

	for cfg, _ := range proxy_conf {
		if _, ok := m.proxies[cfg.Name]; !ok {
			pxy := NewProxy(cfg)
			m.proxies[cfg.Name] = pxy
		}
	}
	return
}
