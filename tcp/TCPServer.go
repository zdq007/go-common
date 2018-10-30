package tcp

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

//服务器结构
type TCPServer struct {
	listener  *net.TCPListener
	addr      *net.TCPAddr
	OnStart   func(port int)
	OnConnect func(conn *Client)
	OnError   func(conn *Client, err error)
	OnData    func(conn *Client, data []byte)
	OnClose   func(conn *Client)
}

//创建服务器
func NewTCPServer() (self *TCPServer) {
	self = new(TCPServer)
	self.OnStart = func(port int) {}
	self.OnConnect = func(conn *Client) {}
	self.OnError = func(conn *Client, err error) {}
	self.OnData = func(conn *Client, data []byte) {}
	self.OnClose = func(conn *Client) {}
	return
}

//设置事件监听
func (self *TCPServer) On(key string, backfn interface{}) {
	if backfn == nil {
		return
	}
	switch strings.ToLower(key) {
	case "start":
		if fn, ok := backfn.(func(port int)); ok {
			self.OnStart = fn
		}
	case "connect":
		if fn, ok := backfn.(func(conn *Client)); ok {
			self.OnConnect = fn
		}
	case "error":
		if fn, ok := backfn.(func(conn *Client, err error)); ok {
			self.OnError = fn
		}
	case "data":
		if fn, ok := backfn.(func(conn *Client, data []byte)); ok {
			self.OnData = fn
		}
	case "close":
		if fn, ok := backfn.(func(conn *Client)); ok {
			self.OnClose = fn
		}
	}
}

func (ser *TCPServer) Listen(addr *net.TCPAddr) bool {
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return false
	}
	ser.listener = listener
	ser.addr = listener.Addr().(*net.TCPAddr)
	ser.OnStart(ser.addr.Port)
	ser.loop()
	return true
}

func (ser *TCPServer) GetPort() (port int) {
	if ser.addr != nil {
		port = ser.addr.Port
	}
	return
}

func (ser *TCPServer) loop() {
	for {
		conn, err := ser.listener.AcceptTCP()
		if err != nil {
			fmt.Println("accept err", err)
			continue
		}
		client := newClient(conn, ser)
		ser.OnConnect(client)
		go client.readLoop()
	}
}

//协议接口
type Protocoler interface {
	//读方法
	Read(client *Client)
	//拆分方法
	SplitPackage(client *Client, readbuf []byte)
}

//客户端类
type Client struct {
	conn     *net.TCPConn           //连接对象
	proto    Protocoler             //协议（数据读取和拆分和包装）
	isClosed bool                   //是否调用关闭
	attrs    map[string]interface{} //绑定属性
	date     int64                  //连接时间
	server   *TCPServer
}

func (client *Client) readLoop() {
	client.proto.Read(client)
}

//创建新客户端
func newClient(conn *net.TCPConn, server *TCPServer) (client *Client) {
	client = new(Client)
	conn.SetNoDelay(false)
	client.conn = conn
	client.server = server
	//这里采用ByteProto
	client.proto = NewJsonProto(server.OnData, server.OnClose, server.OnError)
	client.attrs = make(map[string]interface{}, 2)
	client.date = time.Now().Unix()
	return
}
func (self *Client) Set(key string, val interface{}) {
	self.attrs[key] = val
}
func (self *Client) Get(key string) interface{} {
	return self.attrs[key]
}
func (self *Client) GetString(key string) string {
	v := self.attrs[key]
	if str, ok := v.(string); ok {
		return str
	} else {
		return ""
	}
}
func (self *Client) GetInt(key string) int {
	v := self.attrs[key]
	if val, ok := v.(int); ok {
		return val
	} else {
		return 0
	}
}
func (self *Client) GetInt64(key string) int64 {
	v := self.attrs[key]
	if val, ok := v.(int64); ok {
		return val
	} else {
		return 0
	}
}
func (self *Client) GetUint64(key string) uint64 {
	v := self.attrs[key]
	if val, ok := v.(uint64); ok {
		return val
	} else {
		return 0
	}
}
func (self *Client) Del(key string) {
	delete(self.attrs, key)
}
func (self *Client) RemoteAddr() string {
	return self.conn.RemoteAddr().String()
}
func (self *Client) IP() string {
	arr := strings.Split(self.conn.RemoteAddr().String(), ":")
	return arr[0]
}
func (self *Client) SetTimeout(sec int32) {
	self.conn.SetReadDeadline(time.Now().Add(time.Duration(sec) * time.Second))
}
func (self *Client) Close() {
	self.isClosed = true
	self.conn.Close()
}
func (self *Client) Write(data []byte) (n int, err error) {
	n, err = self.conn.Write(data)
	return
}

//包过大
var TO_LAGER = errors.New("Package is to lagger!")
