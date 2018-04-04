package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	log "github.com/cihub/seelog"
)

const (
	responseHeaderTimeout = time.Duration(30) * time.Second
)

type HttpReverseProxy struct {
	router    *Routers
	Transport http.RoundTripper
}

func NewHttpReverseProxy() (rp *HttpReverseProxy) {
	rp = &HttpReverseProxy{
		router: NewRouters(),
	}
	Transport := &http.Transport{
		ResponseHeaderTimeout: responseHeaderTimeout,
		DisableKeepAlives:     true,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			url := ctx.Value("url").(string)
			host := getHostFromAddr(ctx.Value("host").(string))
			return rp.GetConn(host, url)
		},
	}

	rp.Transport = Transport

	return

}

func (hp *HttpReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	log.Debug("receive request from user")
	ctx := req.Context()

	if c, ok := rw.(http.CloseNotifier); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
		closedCh := c.CloseNotify()
		go func() {
			select {
			case <-closedCh:
				cancel()
			case <-ctx.Done():
			}
		}()
	}

	clientReq := req.WithContext(ctx)
	clientReq.Header = cloneHeader(req.Header)

	clientReq = clientReq.WithContext(context.WithValue(clientReq.Context(), "url", req.URL.Path))
	clientReq = clientReq.WithContext(context.WithValue(clientReq.Context(), "host", req.Host))

	if req.ContentLength == 0 {
		clientReq.Body = nil
	}

	log.Debug("72")
	clientReq.URL.Scheme = "http"
	url := clientReq.Context().Value("url").(string)
	host := getHostFromAddr(clientReq.Context().Value("host").(string))
	log.Debug("76[", host, ":", url, "]")
	host = hp.GetRealHost(host, url)
	log.Debug("real_host:", host)
	if host != "" {
		clientReq.Host = host
	}
	clientReq.URL.Host = host

	clientReq.Close = false

	if con := clientReq.Header.Get("Connection"); con != "" {
		for _, h := range strings.Split(con, ",") {
			if h = strings.TrimSpace(h); h != "" {
				clientReq.Header.Del(h)
			}
		}
	}

	for _, h := range hopHeaders {
		if clientReq.Header.Get(h); h != "" {
			clientReq.Header.Del(h)
		}
	}

	//转发请求
	transport := hp.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	res, err := transport.RoundTrip(clientReq)
	if err != nil {
		log.Error("http proxy error:", err)
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte("Not Found"))
		return
	}

	if con := res.Header.Get("Connection"); con != "" {
		for _, h := range strings.Split(con, ",") {
			res.Header.Del(h)
		}
	}

	for _, h := range hopHeaders {
		res.Header.Del(h)
	}

	copyHeader(rw.Header(), res.Header)

	TrailerLens := len(res.Trailer)
	if TrailerLens > 0 {
		trailer := make([]string, 0, TrailerLens)
		for k := range res.Trailer {
			trailer = append(trailer, k)
		}
		rw.Header().Add("Trailer", strings.Join(trailer, ","))
	}

	rw.WriteHeader(res.StatusCode)
	copyResponse(rw, res.Body)
	res.Body.Close()

	if TrailerLens == len(res.Trailer) {
		copyHeader(rw.Header(), res.Trailer)
		return
	}

	for k, vv := range res.Trailer {
		k = http.TrailerPrefix + k
		for _, v := range vv {
			rw.Header().Add(k, v)
		}
	}
}

func (hp *HttpReverseProxy) GetRealHost(host, url string) (rhost string) {
	r := hp.router.Get(host, url)
	if r == nil {
		return ""
	}
	return r.pxy.GetMsg().Host
}

func (hp *HttpReverseProxy) Register(domain, url string, pxy Proxy) error {
	r := hp.router.Find(domain, url)
	if r != nil {
		err := fmt.Errorf("Register error:router is existed")
		return err
	} else {
		hp.router.Add(domain, url, pxy)
		log.Debug("add router  ", domain, ":", url)
		return nil
	}
}
func (hp *HttpReverseProxy) Remove(domain, url string) {
	hp.router.Del(domain, url)
}

func (hp *HttpReverseProxy) GetConn(domain, url string) (net.Conn, error) {
	r := hp.router.Get(domain, url)
	if r == nil {
		return nil, fmt.Errorf("conn not found")
	}
	return r.pxy.GetWorkConn()
}

var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

func copyResponse(dst io.Writer, src io.Reader) {
	buf := make([]byte, 32*1024)

	for {
		nr, err1 := src.Read(buf)
		if nr > 0 {
			nw, err2 := dst.Write(buf[:nr])
			if err2 != nil {
				log.Error("write error:", err2)
				return
			}
			if nw != nr {
				log.Error("write error:", io.ErrShortWrite)
				return
			}

		}

		if err1 != nil {
			if err1 != io.EOF && err1 != context.Canceled {
				log.Error("read error:", err1)
			}
			return
		}

	}

}
func getHostFromAddr(addr string) (host string) {
	s := strings.Split(addr, ":")
	if len(s) > 1 {
		host = s[0]
	} else {
		host = addr
	}
	return
}

func cloneHeader(h http.Header) http.Header {
	newHeader := make(http.Header, len(h))

	for k, v := range h {
		v2 := make([]string, len(v))
		copy(v2, v)
		newHeader[k] = v2
	}
	return newHeader
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}

}
