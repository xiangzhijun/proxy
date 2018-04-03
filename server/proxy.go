package server

import (
	"net"
	"sync"

	log "github.com/cihub/seelog"
	msg "proxy/message"
	"proxy/utils"
)

type Proxy interface {
	Run()
	Close()

	GetWorkConn() (conn net.Conn, err error)

	GetName() string
	GetType() string
	GetClient() *ClientCtrl
	GetMsg() msg.NewProxy
}

func NewProxy(c *ClientCtrl, m msg.NewProxy) (pxy Proxy) {
	baseProxy := BaseProxy{
		Name:       m.ProxyName,
		Type:       m.ProxyType,
		clientCtrl: c,
		Msg:        m,
	}

	switch m.ProxyType {
	case "tcp":
		pxy = &TcpProxy{
			BaseProxy:  baseProxy,
			RemotePort: m.RemotePort,
			Encrypt:    m.Encrypt,
		}

	case "http":
		pxy = &HttpProxy{
			BaseProxy:  baseProxy,
			RemotePort: m.RemotePort,
			Encrypt:    m.Encrypt,
			Host:       m.Host,
			Domain:     m.Domain,
			Url:        m.Url,
		}

	case "https":
		pxy = &HttpsProxy{
			BaseProxy:  baseProxy,
			RemotePort: m.RemotePort,
			Encrypt:    m.Encrypt,
		}

	}

	return

}

type BaseProxy struct {
	Name string
	Type string

	clientCtrl *ClientCtrl
	Msg        msg.NewProxy

	mu sync.RWMutex
}

func (b *BaseProxy) GetName() string {
	return b.Name
}
func (b *BaseProxy) GetType() string {
	return b.Type
}
func (b *BaseProxy) GetClient() *ClientCtrl {
	return b.clientCtrl
}
func (b *BaseProxy) GetMsg() msg.NewProxy {
	return b.Msg
}

func (pxy *BaseProxy) GetWorkConn() (conn net.Conn, err error) {
	c := pxy.clientCtrl

	for i := 0; i < c.loginMsg.ConnPoolCount+1; i++ {
		if conn, err = c.GetWorkConn(); err != nil {
			log.Error("get workConn from pool error:", err)
			return
		}

		m := msg.StartWork{
			ProxyName: pxy.Name,
		}
		err = msg.WriteMsg(msg.TypeStartWork, m, conn)

		if err != nil {
			log.Error("workConn send startProxy msg error:", err)
			conn.Close()
		} else {
			break
		}
	}

	if pxy.Msg.Encrypt {
		conn, err = utils.Encryption(conn, []byte(pxy.clientCtrl.token))
	}
	return
}

type TcpProxy struct {
	BaseProxy
	RemotePort int
	Encrypt    bool
}

func (pxy *TcpProxy) Run() {

}
func (pxy *TcpProxy) Close() {
}

type HttpProxy struct {
	BaseProxy
	RemotePort int
	Encrypt    bool
	Host       string
	Domain     string
	Url        string
}

func (pxy *HttpProxy) Run() {
	err := pxy.clientCtrl.svr.httpReverseProxy.Register(pxy.Domain, pxy.Url, pxy)
	if err != nil {
		log.Error("register http proxy error:", err)
		pxy.Close()
	}
}

func (pxy *HttpProxy) Close() {
	pxy.clientCtrl.svr.httpReverseProxy.Remove(pxy.Domain, pxy.Url)
}

type HttpsProxy struct {
	BaseProxy
	RemotePort int
	Encrypt    bool
}

func (pxy *HttpsProxy) Run() {

}
func (pxy *HttpsProxy) Close() {
}
