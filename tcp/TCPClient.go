package tcp

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

var (
	nilConn = errors.New("nil connect")
)

type TCPClient struct {
	conn          *net.TCPConn
	OnConnect     func()
	OnClose       func()
	OnError       func(err error)
	OnData        func(data []byte)
	buf           *bytes.Buffer
	seraddr       *net.TCPAddr
	heartDuration int    //心跳间隔时间
	heartPage     []byte //心跳数据包
	isClosed      bool
	status        int //1心跳  2重连  0关闭
}

func NewTCPClient() (self *TCPClient) {
	self = new(TCPClient)
	self.OnConnect = func() {}
	self.OnClose = func() {}
	self.OnError = func(err error) {}
	self.OnData = func(data []byte) {}
	self.heartPage = make([]byte, 0)
	self.heartDuration = 20
	return
}

//心跳和重连共用的活动管理器线程，由status状态开关来作重连或发生心跳包
//连接关闭后状态置2开始重连，连接恢复后状态置1发送心跳
func (self *TCPClient) aliveManager() {
	for !self.isClosed {
		if self.status == 1 {
			if self.conn != nil {
				_, err := self.Write(self.heartPage)
				if err != nil {
					fmt.Println("发送心跳包异常:", err)
				}
			}
			time.Sleep(time.Duration(self.heartDuration) * time.Second)

		} else if self.status == 2 {
			fmt.Println("连接异常，尝试重新连接")
			err := self.Connect(nil)
			if err == nil {
				self.status = 1
			}
		}
		time.Sleep(time.Second * 2)
	}
}

//设置了保持客户端活动才支持重连和自动心跳发送机制，参数为发送间隔时间和心跳包内容
func (self *TCPClient) SetKeepAlive(second int, data []byte) {
	self.heartDuration = second
	self.heartPage = data
	self.status = 1
}

//设置为重连状态
func (self *TCPClient) reConnect() {
	self.status = 2
}

//手动关闭连接，手动关闭的不会重连
func (self *TCPClient) Close() {
	fmt.Printf("关闭 %p", self.conn)
	self.status = 0
	self.isClosed = true
	if self.conn != nil {
		self.conn.Close()
	}
}

//开始连接
func (self *TCPClient) Connect(addr *net.TCPAddr) error {
	if addr != nil {
		self.seraddr = addr
	}
	con, err := net.DialTCP("tcp", nil, self.seraddr)
	if err != nil {
		fmt.Println(err)
		if addr != nil && self.status == 1 {
			self.reConnect()
			go self.aliveManager()
		}
		return err
	} else {
		if addr != nil && self.status == 1 {
			go self.aliveManager()
		}
	}
	con.SetNoDelay(false)
	self.isClosed = false
	self.buf = bytes.NewBuffer(make([]byte, 0, JSON_CLIENT_BUF))
	self.conn = con
	self.OnConnect()
	go self.readData()
	return nil
}

//设置事件回调
func (self *TCPClient) On(key string, backfn interface{}) {
	if backfn == nil {
		return
	}
	switch strings.ToLower(key) {
	case "connect":
		if fn, ok := backfn.(func()); ok {
			self.OnConnect = fn
		}
	case "close":
		if fn, ok := backfn.(func()); ok {
			self.OnClose = fn
		}
	case "error":
		if fn, ok := backfn.(func(err error)); ok {
			self.OnError = fn
		}
	case "data":
		if fn, ok := backfn.(func(data []byte)); ok {
			self.OnData = fn
		}
	}
}

//写数据
func (self *TCPClient) Write(data []byte) (n int, err error) {
	if self.conn != nil {
		n, err = self.conn.Write(data)
	} else {
		return 0, nilConn
	}
	return
}

//自动读数据，读到数据后回调给客户线程
func (self *TCPClient) readData() {
	control := true
	readbuf := make([]byte, JSON_RECV_BUF_LEN)
	for control {
		n, err := self.conn.Read(readbuf)
		if err != nil {
			control = false
			if !self.isClosed {
				if err.Error() == "EOF" {
					self.OnClose()
				} else {
					self.OnError(err)
				}
				self.reConnect()
			}
			break
		}
		if n <= 0 {
			control = false
			if !self.isClosed {
				self.OnClose()
				self.reConnect()
			}
			break
		}
		self.PackageSplit(readbuf[:n])
	}
}

//拆分包 /r/n->13 10
func (self *TCPClient) PackageSplit(buf []byte) {
	k := 0
	for i, byteval := range buf {
		if byteval == 10 {
			if i > 0 && buf[i-1] == 13 {
				//读取到包结束符号
				self.buf.Write(buf[k : i-1])
				self.OnData(self.buf.Bytes())
				self.buf.Truncate(0)
				k = i + 1
			} else if i == 0 && self.buf.Len() > 0 && self.buf.Bytes()[self.buf.Len()-1] == 13 {
				//读取到包结束符号
				self.OnData(self.buf.Bytes()[:self.buf.Len()-1])
				self.buf.Truncate(0)
				k = i + 1
			}
		}
	}
	//检测是否还有半截包
	if k < len(buf) {
		self.buf.Write(buf[k:])
	}
}
