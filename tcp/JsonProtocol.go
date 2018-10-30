package tcp

import (
	"bytes"
	"fmt"
)

const (
	JSON_RECV_BUF_LEN       = 10 * 1024
	JSON_MAX_BUF      int64 = 1024 * 1024 * 1024 * 10
	JSON_CLIENT_BUF         = 2 * bytes.MinRead
)

//json分拆协议包以 \r\n 分隔
type JsonProto struct {
	buf     *bytes.Buffer
	OnError func(conn *Client, err error)
	OnData  func(conn *Client, data []byte)
	OnClose func(conn *Client)
}

func NewJsonProto(OnData func(conn *Client, data []byte), OnClose func(conn *Client), OnError func(conn *Client, err error)) (proto *JsonProto) {
	proto = new(JsonProto)
	proto.buf = bytes.NewBuffer(make([]byte, 0, JSON_CLIENT_BUF))
	proto.OnError = OnError
	proto.OnData = OnData
	proto.OnClose = OnClose
	return
}

//读取数据
func (self *JsonProto) Read(client *Client) {
	control := true
	readbuf := make([]byte, JSON_RECV_BUF_LEN)
	//clientAddr := client.conn.RemoteAddr()
	for control {
		n, err := client.conn.Read(readbuf)
		if err != nil {
			control = false
			if !client.isClosed {
				//fmt.Println(clientAddr, "连接异常!", err)
				//关闭并释放资源，否则服务器会有CLOSE_WAIT出现，客户端会员 FIN_WAIT2
				client.Close()
				if err.Error() == "EOF" {
					self.OnClose(client)
				} else {
					self.OnError(client, err)
				}
			}
			break
		}
		if n <= 0 {
			control = false
			fmt.Println("客户端关闭连接")
			if !client.isClosed {
				client.Close()
				self.OnClose(client)
			}
			break
		}
		self.SplitPackage(client, readbuf[:n])
	}
}

//拆包
func (self *JsonProto) SplitPackage(client *Client, buf []byte) {
	k := 0
	for i, byteval := range buf {
		//1: i > 0 && buf[i-1] == 13
		//2: i==0 && client.Allbuf.Len() > 0 && client.Allbuf.Bytes()[client.Allbuf.Len()-1] == 13
		if byteval == 10 {
			if i > 0 && buf[i-1] == 13 {
				//读取到包结束符号
				self.buf.Write(buf[k : i-1])
				self.OnData(client, self.buf.Bytes())
				self.buf.Truncate(0)
				k = i + 1
			} else if i == 0 && self.buf.Len() > 0 && self.buf.Bytes()[self.buf.Len()-1] == 13 {
				//读取到包结束符号
				self.OnData(client, self.buf.Bytes()[:self.buf.Len()-1])
				self.buf.Truncate(0)
				k = i + 1
			}
		}
	}
	//检测是否还有半截包
	if k < len(buf) {
		self.buf.Write(buf[k:])
		//包过大
		if int64(self.buf.Len()) > JSON_MAX_BUF {
			client.Close()
			self.OnError(client, TO_LAGER)
		}
		//fmt.Println("半截包内容:", string(client.Allbuf.Bytes()))
	}
}
