package tcp

import (
	"encoding/binary"
	"github.com/donnie4w/go-logger/logger"
)

const (
	RECV_BUF uint32 = 100 * 1024
	MAX_BUF  uint32 = 90 * 1024
	MIN_BUF  uint32 = 10 * 1024
)

const (
	HEAD_LEN           uint32 = 12 //包头长度
	HEAD_VERSION_POS   uint32 = 0  //版本字节的开始位置
	HEAD_MSGTYPE_POS   uint32 = 1  //消息类型字节的开始位置
	HEAD_PACKETLEN_POS uint32 = 2  //消息长度字节的开始位置
	HEAD_TARGETID_POS  uint32 = 4  //目标字节的开始位置
)

//自定义协议 包由包长度决定
type ByteProto struct {
	readbuf []byte //数据读取缓冲BUF
	dl      uint32 //数据开始的位置
	rl      uint32 //数据结束的位置
	OnError func(conn *Client, err error)
	OnData  func(conn *Client, data []byte)
	OnClose func(conn *Client)
}

func NewByteProto(OnData func(conn *Client, data []byte), OnClose func(conn *Client), OnError func(conn *Client, err error)) (proto *ByteProto) {
	proto = new(ByteProto)
	proto.readbuf = make([]byte, RECV_BUF)
	proto.OnError = OnError
	proto.OnData = OnData
	proto.OnClose = OnClose
	return
}

//协议头
type PacketHead struct {
	Version  byte   //版本号       1字节
	Msgtype  byte   //消息类型   1字节
	Datalen  uint16 //实体长度   2字节
	Targetid uint64 //目标ID   8字节
}

//判断是否需要移动剩余缓冲字节到buf的开始位置
func (self *ByteProto) CheckReadBuffer() error {
	if (RECV_BUF - self.rl) < MIN_BUF {
		copy(self.readbuf, self.readbuf[self.dl:self.rl])
		self.rl = self.rl - self.dl
		self.dl = 0
	}
	if (self.rl - self.dl) > MAX_BUF {
		logger.Error("包过大,包长：", self.rl-self.dl)
		return TO_LAGER
	}

	return nil
}

//读取数据
func (self *ByteProto) Read(client *Client) {
	control := true
	clientAddr := client.conn.RemoteAddr()
	count := 0
	conn := client.conn
	readbuf := self.readbuf
	//now:=time.Now()
	for control {
		if err := self.CheckReadBuffer(); err != nil {
			client.Close()
		}
		n, err := conn.Read(readbuf[self.rl:])
		if err != nil {
			control = false
			if !client.isClosed {
				logger.Debug(clientAddr, "连接异常!", err)
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
			logger.Debug("客户端关闭连接")
			if !client.isClosed {
				client.Close()
				self.OnClose(client)
			}
			break
		}
		self.rl += uint32(n)
		count += n
		//logger.Debug("接收到数据长度:", n)
		//fmt.Println("接收到时数据：", n, " 总数据：", count, "时间：", time.Since(now))
		self.SplitPackage(client, readbuf)
	}
}

//拆包
func (self *ByteProto) SplitPackage(client *Client, readbuf []byte) {
	logger.Debug("数据:", readbuf[self.dl:self.rl])
	for {
		//当前消息包的长度
		if self.rl-self.dl < HEAD_LEN {
			break
		}
		packlen := binary.BigEndian.Uint16(readbuf[self.dl+HEAD_PACKETLEN_POS:])
		//计算出 完整包的结束游标
		completelen := self.dl + HEAD_LEN + uint32(packlen)
		if self.rl < completelen {
			break
		}
		packet := readbuf[self.dl:completelen]
		self.dl = completelen
		self.OnData(client, packet)
	}
}

func NewPacketHead2(ver, msgtype byte, datalen uint16, targetid uint64) (ph *PacketHead) {
	ph = new(PacketHead)
	ph.Version = ver
	ph.Msgtype = msgtype
	ph.Datalen = datalen
	ph.Targetid = targetid
	return
}
func NewPacketHead(data []byte) (ph *PacketHead) {
	packlen := binary.BigEndian.Uint16(data[HEAD_PACKETLEN_POS:])
	ph = NewPacketHead2(data[HEAD_VERSION_POS], data[HEAD_MSGTYPE_POS], packlen, binary.BigEndian.Uint64(data[HEAD_TARGETID_POS:]))
	return
}

func (self *PacketHead) ToByte() []byte {
	data := make([]byte, self.Datalen+uint16(HEAD_LEN))
	data[HEAD_VERSION_POS] = self.Version
	data[HEAD_MSGTYPE_POS] = self.Msgtype
	binary.BigEndian.PutUint16(data[HEAD_PACKETLEN_POS:], self.Datalen)
	binary.BigEndian.PutUint64(data[HEAD_TARGETID_POS:], self.Targetid)
	return data
}
func (self *PacketHead) ToString() {
	logger.Debug("版本号：", self.Version, "消息类型：", self.Msgtype, "包长度：", self.Datalen, "目标ID：", self.Targetid)
}

func WarpData(msgtype byte, data []byte) []byte {
	dl := 0
	if data != nil {
		dl = len(data)
	}
	ph := NewPacketHead2(1, msgtype, uint16(dl), 0)
	rdata := ph.ToByte()
	if data != nil && dl > 0 {
		copy(rdata[HEAD_LEN:], data)
	}
	return rdata
}

//连接
//1|1|0|目标ID|

//退出
//1|2|0|0
