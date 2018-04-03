package message

//import "fmt"

const (
	TypeLogin        = '1'
	TypeLoginResp    = 'a'
	TypeNewProxy     = '2'
	TypeNewProxyResp = 'b'
	TypeNewWorkConn  = '3'
	TypeReqWorkConn  = 'c'
	TypePing         = '4'
	TypePong         = 'd'
	TypeStartWork    = 'e'
)

//var AllType = [...]string{TypeLogin, TypeLoginResp, TypeNewProxy, TypeNewProxyResp, TypePing, TypePong}

type Message struct {
	Type    byte   `json:"type"`
	MesData string `json:"mes_data"`
}

//客户端启动时，会向服务器发生Login消息
type Login struct {
	Hostname      string `json:"hostname"`
	User          string `json:"user"`
	Sign          string `json:"sign"` //key+timestamp生成的MD5值
	ClientId      string `json:"client_id"`
	ConnPoolCount int    `json:"conn_pool_count"`
	Timestamp     int64  `json:"timestamp"`
}

//服务器收到客户端的Login消息后，会返回LoginResp消息
type LoginResp struct {
	ClientId string `json:"client_id"`
	Status   int    `json:"status"`
	Error    string `json:"error"`
}

type NewProxy struct {
	ProxyName  string `json:"proxy_name"`
	ProxyType  string `json:"proxy_type"`
	RemotePort int    `json:"remote_port"` //指定服务器向外的代理接口
	Encrypt    bool   `json:"encrypt"`     //传输是否加密

	Host   string `json:"host"`
	Domain string `json:"domain"`
	Url    string `json:"url"`
}

type NewProxyResp struct {
	ProxyName  string `json:"proxy_name"`
	RemotePort int    `json:"remote_port"`
	Error      string `json:"error"`
}

type ReqWorkConn struct {
}

type NewWorkConn struct {
	ClientId string `json:"client_id"`
}

//客户端定期向服务器发送Ping消息，若在一定时间内没有收到服务器回复Pong，则重新登录
type Ping struct {
}
type Pong struct {
}

type StartWork struct {
	ProxyName string `json:"proxy_name"`
}
