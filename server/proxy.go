package server

import (
	"fmt"
	"io"
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
	realPort := pxy.clientCtrl.svr.portManager.Get(pxy.clientCtrl.clientId, pxy.RemotePort)
	log.Debug("get realport:", realPort)
	if realPort <= 0 {
		log.Error("get realport error")
		return
	}
	pxy.RemotePort = realPort

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", pxy.clientCtrl.svr.conf.BindIP, realPort))
	if err != nil {
		log.Error(err)
		return
	}

	go func(L net.Listener, p *TcpProxy) {
		for {
			conn, err := L.Accept()
			if err != nil {
				log.Info("listener is closed:", err)
				return
			}

			log.Debug("get user connection")
			go TcpHandler(conn, p)
		}
	}(l, pxy)
	log.Debug("tcp proxy is running")

}
func (pxy *TcpProxy) Close() {
	pxy.clientCtrl.svr.portManager.Release(pxy.RemotePort)
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
		return
	}
	log.Debug("HttpProxy is running")
}

func (pxy *HttpProxy) Close() {
	pxy.clientCtrl.svr.httpReverseProxy.Remove(pxy.Domain, pxy.Url)
	log.Debug("httpProxy is Closed")
}

type HttpsProxy struct {
	BaseProxy
	RemotePort int
	Encrypt    bool
}

func (pxy *HttpsProxy) Run() {
	err := pxy.clientCrtl.svr.httpsReverseProxy.Register(pxy.Domain, "/", pxy) //https不支持url路由
	if err != nil {
		log.Error("register https proxy error:", err)
		return
	}
	log.Debug("HttpsProxy is running")

}
func (pxy *HttpsProxy) Close() {
	pxy.clientCrtl.svr.httpsReverseProxy.Remove(pxy.Domain, "/")
	log.Debug("httpProxy is Closed")
}

func TcpHandler(userConn net.Conn, pxy *TcpProxy) {
	defer userConn.Close()

	workConn, err := pxy.GetWorkConn()
	if err != nil {
		log.Error("get work conn error:", err)
	}
	defer workConn.Close()

	log.Debug("bridge connection")
	BridgeConn(userConn, workConn)
	log.Debug("bridge completed")
}

func BridgeConn(conn1, conn2 io.ReadWriteCloser) {
	var wait sync.WaitGroup
	wait.Add(2)

	Copy := func(dst, src io.ReadWriteCloser) {
		defer wait.Done()

		buf := make([]byte, 16*1024)
		io.CopyBuffer(dst, src, buf)
	}

	go Copy(conn2, conn1)
	go Copy(conn1, conn2)
	wait.Wait()

}
