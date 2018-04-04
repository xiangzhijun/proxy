package server

import (
	"fmt"
	"net"
	"net/http"
	"time"

	log "github.com/cihub/seelog"
	"proxy/config"
	msg "proxy/message"
	"proxy/utils"
)

const (
	ReadTimeout time.Duration = 10 * time.Second
)

type Service struct {
	//接受所有客户端的连接
	listener net.Listener

	conf *config.ServerConfig

	//管理所有客户端
	clientManager *ClientManager

	//管理所有代理
	proxyManager *ProxyManager

	//http反向代理
	httpReverseProxy *HttpReverseProxy

	userToken config.UserTokenMap
}

func NewService(conf *config.ServerConfig) (svr *Service, err error) {
	svr = &Service{
		conf:          conf,
		clientManager: NewClientManager(),
		proxyManager:  NewProxyManager(),
		userToken:     make(map[string]string),
	}

	err = svr.userToken.ReadUserTokenMap(conf.UserTokenFile)
	if err != nil {
		return nil, err
	}
	log.Debug(svr.userToken)

	tcp_addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", conf.BindIP, conf.BindPort))
	if err != nil {
		return nil, err
	}
	svr.listener, err = net.ListenTCP("tcp", tcp_addr)
	if err != nil {
		return nil, err
	}

	if conf.HttpProxy.VisitPort > 0 {
		hp := NewHttpReverseProxy()
		svr.httpReverseProxy = hp
		addr := fmt.Sprintf("%s:%d", conf.HttpProxy.VisitIP, conf.HttpProxy.VisitPort)

		var l net.Listener
		l, err = net.Listen("tcp", addr)
		if err != nil {
			log.Error("Creat http reverse proxy error:", err)
			return
		}

		Server := &http.Server{
			Addr:    addr,
			Handler: hp,
		}
		go Server.Serve(l)
		log.Info("http reverse proxy start")
	}

	log.Debug("NewService")
	return
}

func (svr *Service) Run() {
	l := svr.listener
	for {
		conn, err := l.Accept()
		log.Debug("accept client connection")
		if err != nil {
			return
		}

		go func(conn *net.TCPConn) {
			conn.SetReadDeadline(time.Now().Add(ReadTimeout))
			msgType, m, err := msg.ReadMsg(conn)
			log.Debug("receive  msg")
			if err != nil {
				conn.Close()
				return
			}
			conn.SetReadDeadline(time.Time{})

			switch msgType {
			case msg.TypeLogin:
				log.Debug("receive login msg")
				err = svr.RegisterClient(conn, m.(*msg.Login))
				if err != nil {
					log.Error(err)
					loginResp := msg.LoginResp{
						Error: fmt.Sprintf("%v", err),
					}
					msg.WriteMsg(msg.TypeLoginResp, loginResp, conn)
					conn.Close()
					return
				}
				log.Debug("RegisterClient success")

			case msg.TypeNewWorkConn:
				log.Debug("newworkconn")
				c, ok := svr.clientManager.Client[m.(*msg.NewWorkConn).ClientId]
				if ok {
					c.NewWorkConn(conn)
				} else {
					log.Warn("receive work connection,but not found client:", m.(*msg.NewWorkConn).ClientId)
					conn.Close()
				}
			default:
				conn.Close()

			}

		}(conn.(*net.TCPConn))

	}

}

func (svr *Service) RegisterClient(conn *net.TCPConn, loginMsg *msg.Login) (err error) {
	now := time.Now().Unix()
	if svr.conf.AuthTimeout != 0 && now-loginMsg.Timestamp > svr.conf.AuthTimeout {
		err = fmt.Errorf("Authorization Error: Timeout")
		return
	}

	var token string
	var ok bool
	if token, ok = svr.userToken[loginMsg.User]; !ok {
		err = fmt.Errorf("Authorization Error: This user does not exist")
		return
	}
	_, sign := utils.GetMD5([]byte(fmt.Sprintf("%s%d", token, loginMsg.Timestamp)))
	log.Debug(loginMsg.Sign)
	if string(sign) != loginMsg.Sign {
		err = fmt.Errorf("Authorization Error: Token error")
		return
	}

	if loginMsg.ClientId == "" {
		loginMsg.ClientId, err = utils.GetClientId()
		if err != nil {
			return
		}
	}
	if _, ok := svr.clientManager.Client[loginMsg.ClientId]; ok {
		loginMsg.ClientId, err = utils.GetClientId()
		if err != nil {
			return
		}
	}

	clientCtrl := NewClientCtrl(svr, loginMsg, conn, token)
	svr.clientManager.Add(loginMsg.ClientId, clientCtrl)

	clientCtrl.Start()

	return
}
