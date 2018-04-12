package client

import (
	"fmt"
	"io"
	"net"
	"sync"

	log "github.com/cihub/seelog"
	"proxy/config"
	"proxy/utils"
)

type Proxy interface {
	Work(conn net.Conn)
	Run() error

	GetName() string
	GetType() string
	GetRemotePort() int
	GetToken() string
	GetStatus() int
	GetConfig() *config.ProxyConf

	Close()
}

func NewProxy(cfg *config.ProxyConf, token string) (pxy Proxy) {
	baseProxy := BaseProxy{
		Name:       cfg.Name,
		Type:       cfg.Type,
		RemotePort: cfg.RemotePort,
		Token:      token,
		cfg:        cfg,
	}
	switch cfg.Type {
	case "tcp":
		pxy = &TcpProxy{
			BaseProxy: baseProxy,
		}

	case "http":
		pxy = &HttpProxy{
			BaseProxy: baseProxy,
		}

	case "https":
		pxy = &HttpsProxy{
			BaseProxy: baseProxy,
		}
	}
	return
}

type BaseProxy struct {
	Name       string
	Type       string
	RemotePort int
	Token      string
	Status     int

	cfg *config.ProxyConf
}

func (b *BaseProxy) GetName() string {
	return b.Name
}
func (b *BaseProxy) GetType() string {
	return b.Type
}
func (b *BaseProxy) GetRemotePort() int {
	return b.RemotePort
}
func (b *BaseProxy) GetToken() string {
	return b.Token
}
func (b *BaseProxy) GetStatus() int {
	return b.Status
}
func (b *BaseProxy) GetConfig() *config.ProxyConf {
	return b.cfg
}

type HttpProxy struct {
	BaseProxy
	closed bool
}

func (pxy *HttpProxy) Run() error {
	pxy.Status = ProxyStatusRunning
	log.Debug(pxy.Name, " is running[tcp]")
	return nil
}

func (pxy *HttpProxy) Work(conn net.Conn) {
	Handler(pxy.cfg, conn, pxy.Token)
	log.Debug(pxy.Name, " is working[http]")
}

func (pxy *HttpProxy) Close() {
	pxy.Status = ProxyStatusClosed
	log.Debug(pxy.Name, " is closed[https]")
}

type HttpsProxy struct {
	BaseProxy
	closed bool
}

func (pxy *HttpsProxy) Run() error {
	pxy.Status = ProxyStatusRunning
	log.Debug(pxy.Name, " is running[tcp]")
	return nil
}

func (pxy *HttpsProxy) Work(conn net.Conn) {
	Handler(pxy.cfg, conn, pxy.Token)
	log.Debug(pxy.Name, " is working[https]")
}

func (pxy *HttpsProxy) Close() {
	pxy.Status = ProxyStatusClosed
	log.Debug(pxy.Name, " is closed[https]")
}

type TcpProxy struct {
	BaseProxy
	closed bool
}

func (pxy *TcpProxy) Run() error {
	log.Debug(pxy.Name, " is running[tcp]")
	pxy.Status = ProxyStatusRunning
	return nil
}

func (pxy *TcpProxy) Work(conn net.Conn) {
	Handler(pxy.cfg, conn, pxy.Token)
	log.Debug(pxy.Name, " is working[https]")
}

func (pxy *TcpProxy) Close() {
	pxy.Status = ProxyStatusClosed
	log.Debug(pxy.Name, " is closed[https]")
}

func Handler(cfg *config.ProxyConf, conn net.Conn, token string) {
	defer conn.Close()
	var err error
	var remote io.ReadWriteCloser
	remote = conn
	if cfg.Encryption {
		remote, err = utils.Encryption(conn, []byte(token))
		if err != nil {
			log.Error("proxy handler error:", err)
			return
		}
	}

	local_server_addr := fmt.Sprintf("%s:%d", cfg.LocalIP, cfg.LocalPort)
	tcp_addr, err1 := net.ResolveTCPAddr("tcp", local_server_addr)
	if err != nil {
		log.Error("ResolveT addr error:", err1)
		return
	}

	localConn, err2 := net.DialTCP("tcp", nil, tcp_addr)
	if err2 != nil {
		log.Error("connect to local server ", local_server_addr, " error:", err2)
	}
	defer localConn.Close()

	BridgeConn(remote, localConn)
	log.Debug("bridgeconn over")
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
func copyBuffer(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	if buf == nil {
		buf = make([]byte, 32*1024)
	}
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}
