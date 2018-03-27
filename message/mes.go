package message

//import "fmt"

const (
	TypeLogin        = '1'
	TypeLoginResp    = 'a'
	TypeNewProxy     = '2'
	TypeNewProxyResp = 'b'
	TypePing         = '3'
	TypePong         = 'c'
)

//var AllType = [...]string{TypeLogin, TypeLoginResp, TypeNewProxy, TypeNewProxyResp, TypePing, TypePong}

type Message struct {
	Type    byte   `json:"type"`
	MesData string `json:"mes_data"`
}

//客户端启动时，会向服务器发生Login消息
type Login struct {
	Hostname  string `json:"hostname"`
	User      string `json:"user"`
	Sign      string `json:"sign"` //key+timestamp生成的MD5值
	ClientId  string `json:"client_id"`
	Timestamp string `json:"timestamp"`
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
}

type NewProxyResp struct {
}

//客户端定期向服务器发送Ping消息，若在一定时间内没有收到服务器回复Pong，则重新登录
type Ping struct {
}
type Pong struct {
}
