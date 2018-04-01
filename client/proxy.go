package client

import (
	"net"
	"proxy/config"
)

type Proxy interface {
	Work(conn net.Conn)
	Run() error
	Close()
}

func NewProxy(cfg *config.ProxyConf) (pxy *Proxy) {
	baseProxy := BaseProxy{
		Name:       cfg.Name,
		Type:       cfg.Type,
		RemotePort: cfg.RemotePort,
	}
	switch cfg.Type {
	case "tcp":
		pxy = &TcpProxy{
			BaseProxy: baseProxy,
			cfg:       cfg,
		}

	case "http":
		pxy = &HttpProxy{
			BaseProxy: baseProxy,
			cfg:       cfg,
		}

	case "https":
		pxy = &HttpsProxy{
			BaseProxy: baseProxy,
			cfg:       cfg,
		}
	case "extranet":
		pxy = &ExtranetProxy{
			BaseProxy: baseProxy,
			cfg:       cfg,
		}
	}
	return
}

type BaseProxy struct {
	Name       string
	Type       string
	RemotePort int
	Status     int
}

type HttpProxy struct {
	BaseProxy
	closed bool
	cfg    *config.ProxyConf
}

func (pxy *HttpProxy) Run() error {

	return nil
}

func (pxy *HttpProxy) Work(conn net.Conn) {
}

func (pxy *HttpProxy) Close() {

}

type HttpsProxy struct {
	BaseProxy
	closed bool
	cfg    *config.ProxyConf
}

func (pxy *HttpsProxy) Run() error {
	return nil
}

func (pxy *HttpsProxy) Work(conn net.Conn) {
}

func (pxy *HttpsProxy) Close() {

}

type TcpProxy struct {
	BaseProxy
	closed bool
	cfg    *config.ProxyConf
}

func (pxy *TcpProxy) Run() error {
	return nil
}

func (pxy *TcpProxy) Work(conn net.Conn) {
}

func (pxy *TcpProxy) Close() {

}

type ExtranetProxy struct {
	BaseProxy
	closed bool
	cfg    *config.ProxyConf
}

func (pxy *ExtranetProxy) Run() error {
	return nil

}

func (pxy *ExtranetProxy) Work(conn net.Conn) {

}

func (pxy *ExtranetProxy) Close() {

}
