package client

import (
	"net"
	"net/http"

	"proxy/config"
)

type ExtranetProxy struct {
	cfg    *config.ExtranetConf
	l      net.Listener
	closed bool
}

func NewExtranetProxy(conf *config.ExtranetConf) *ExtranetProxy {
	ep := &ExtranetProxy{
		cfg:    conf,
		closed: false,
	}
	return ep
}

func (pxy *ExtranetProxy) Run() error {
	addr := fmt.Sprintf("%s:%d", pxy.cfg.BindIP, pxy.cfg.BindPort)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("listen port err:", err)
		return err
	}

	pxy.l = l

	svr := http.Server{
		Addr:    addr,
		Handler: pxy,
	}

	go svr.Serve(l)
	log.Debug("Extranet proxy is running")
	pxy.Status = ProxyStatusRunning
	return nil

}

func (pxy *ExtranetProxy) Close() {
	pxy.l.Close()
	pxy.closed = true
	log.Info("extrnet proxy is closed")
}

func (pxy *ExtranetProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	if req.Method == http.MethodConnect {
		hijacker, ok := rw.(http.Hijacker)
		if !ok {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		user, _, err := hijacker.Hijack()
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		remote, err := net.Dial("tcp", req.URL.Host)
		if err != nil {
			http.Error(rw, "Failed", http.StatusBadRequest)
			user.Close()
			return
		}

		user.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		go BridgeConn(user, remote)
	} else {
		removeHopHeader(req)

		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		for k, vs := range resp.Header {
			for _, v := range vs {
				rw.Header().Add(k, v)
			}
		}

		rw.WriteHeader(resp.StatusCode)

		_, err = io.Copy(rw, resp.Body)
		if err != nil && err != io.EOF {
			log.Error("copy resp error:", err)
		}
	}

}

func removeHopHeader(req *http.Request) {
	req.RequestURI = ""
	req.Header.Del("Proxy-Connection")
	req.Header.Del("Connection")
	req.Header.Del("Proxy-Authenticate")
	req.Header.Del("Proxy-Authorization")
	req.Header.Del("TE")
	req.Header.Del("Trailers")
	req.Header.Del("Transfer-Encoding")
	req.Header.Del("Upgrade")

}
