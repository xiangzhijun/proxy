package message

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
)

func Pack(mes_type byte, msg interface{}) (m Message, err error) {
	msg_data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	m = Message{
		Type:    mes_type,
		MesData: string(msg_data),
	}

	return
}

func UnPack(m Message) (msg_type byte, msg interface{}, err error) {
	switch m.Type {
	case TypeLogin:
		msg = new(Login)

	case TypeLoginResp:
		msg = new(LoginResp)
	case TypePing:
		msg = new(Ping)
	case TypePong:
		msg = new(Pong)
	case TypeNewProxy:
		msg = new(NewProxy)
	case TypeNewProxyResp:
		msg = new(NewProxyResp)
	case TypeNewWorkConn:
		msg = new(NewWorkConn)
	case TypeReqWorkCOnn:
		msg = new(ReqWorkCOnn)
	case TypeStartWork:
		msg = new(StartWork)
	}
	err = json.Unmarshal([]byte(m.MesData), msg)
	msg_type = m.Type

	return
}

func PackMsg(msg Message) (data []byte, err error) {
	data, err = json.Marshal(msg)
	return
}

func UnPackMsg(data []byte) (msg Message, err error) {
	err = json.Unmarshal(data, &msg)
	return
}

func WriteMsg(mes_type byte, msg interface{}, c io.Writer) error {
	m, err := Pack(mes_type, msg)
	if err != nil {
		return err
	}

	data, err := PackMsg(m)
	if err != nil {
		return err
	}

	buffer := bytes.NewBuffer(nil)
	binary.Write(buffer, binary.BigEndian, int64(len(data)))
	buffer.Write(data)

	_, err = c.Write(buffer.Bytes())
	return err
}

func ReadMsg(c io.Reader) (byte, interface{}, error) {
	var length int64
	err := binary.Read(c, binary.BigEndian, &length)
	if err != nil {
		return 0, nil, err
	}

	buff := make([]byte, length)
	_, err = io.ReadFull(c, buff)
	if err != nil {
		return 0, nil, err
	}

	m, err := UnPackMsg(buff)
	if err != nil {
		return 0, nil, err
	}

	return UnPack(m)

}

func ReadRawMsg(c io.Reader) (*msg.Message, error) {
	var length int64
	err := binary.Read(c, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}

	buff := make([]byte, length)
	_, err = io.ReadFull(c, buff)
	if err != nil {
		return nil, err
	}

	m, err := UnPackMsg(buff)
	return &m, err

}

func WriteRawMsg(m *msg.Message, c io.Writer) error {

	data, err := PackMsg(m)
	if err != nil {
		return err
	}

	buffer := bytes.NewBuffer(nil)
	binary.Write(buffer, binary.BigEndian, int64(len(data)))
	buffer.Write(data)

	_, err = c.Write(buffer.Bytes())
	return err
}
