package util

import (
	"fmt"
	"github.com/streadway/amqp"
	"time"
)

//--------------生产--只负责生产-------------
type RabbitmqPublish struct {
	ch           *amqp.Channel //连接通道
	cachedata    [][]byte      //连接异常后缓存的客户端发送的数据buf
	reConnecting bool          //正在重连
	exName       string
	serverurl    string
	durable      bool
	autodel      bool
}

func NewRabbitmqPublish() *RabbitmqPublish {
	conn := &RabbitmqPublish{ch: nil, reConnecting: false, cachedata: make([][]byte, 0)}
	return conn
}

//重连函数
func (self *RabbitmqPublish) reconnect() {
	self.reConnecting = true
	self.ch = nil
	go func() {
		c := time.Tick(time.Second * 6)
		for d := range c {
			fmt.Println(d)
			isOk := self.connect()
			if isOk {
				self.reConnecting = false
				break
			}
		}
	}()
}

/**
*	serverurl服务器URL 例如："amqp://imuser:lwl-zdq@10.0.10.163:5672/im"
*	durable  是否持久化  要做到持久化，路由和队列这个都得设置为true
*	exName   路由名称
*	autodel  无连接时是否自动销毁
**/
func (self *RabbitmqPublish) Connect(serverurl, exName string, durable, autodel bool) {
	self.exName = exName
	self.serverurl = serverurl
	self.durable = durable
	self.autodel = autodel
	go func() {
		self.connect()
	}()
}
func (self *RabbitmqPublish) connect() bool {
	conn, err := amqp.Dial(self.serverurl)
	if err != nil {
		fmt.Println("错误:", err)
		if !self.reConnecting {
			self.reconnect()
		}
		return false
	}
	tmpch, err := conn.Channel()
	if err != nil {
		fmt.Printf("打开Channel错误：", err)
		if !self.reConnecting {
			self.reconnect()
		}
		return false
	}
	err = tmpch.ExchangeDeclare(
		self.exName,  // name
		"direct",     // type
		self.durable, // durable 是否持久，交换机和队列同时设置为持久的，就会持久化消息
		self.autodel, // auto-deleted 如果是TRUE，则没有一个连接绑定到该路由的时候，该路由就会自动删除
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		fmt.Println(err)
		if !self.reConnecting {
			self.reconnect()
		}
		return false
	}
	self.ch = tmpch
	return true
}
func (self *RabbitmqPublish) ISOK() bool {
	return self.ch != nil
}
func (self *RabbitmqPublish) SendMsg(key string, msg []byte) {
	if self.ch != nil {
		err := self.ch.Publish(
			self.exName, // exchange
			key,         // routing key
			false,       // mandatory
			false,       // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        msg,
			})
		if err != nil {
			fmt.Println("发送失败:", err)
			//cachedata=append(cachedata,msg)
			if !self.reConnecting {
				self.reconnect()
			}
		}
	} else {
		//cachedata=append(cachedata,msg)
	}
}

//------------消费--还能生产-------------------
type RabbitMQConsume struct {
	ch         *amqp.Channel //连接通道
	handleback func([]byte)
	exName     string
	queueName  string
	key        string
	serverurl  string
	durable    bool
	autodel    bool
}

func (self *RabbitMQConsume) reConnect() {
	self.ch = nil
	time.Sleep(time.Second * 2)
	self.Connect()
}

/**
*	serverurl	 服务器URL 例如："amqp://imuser:lwl-zdq@10.0.10.163:5672/im"
*	exName  	 路由名称
*	queueName	 传空字符串，会自动产生队列名字
*   key     	 路由和队列绑定的KEY,这里的封装只能使用direct模式
*	durable 	 是否持久化  要做到持久化，路由和队列这个都得设置为true
*	autodel  	 无连接时是否自动销毁
*	handleback 	 处理回调函数
**/
func (self *RabbitMQConsume) Start(serverurl, exName, queueName, key string, durable, autodel bool, handleback func([]byte)) {
	if handleback != nil {
		self.handleback = handleback
	}
	self.exName = exName
	self.queueName = queueName
	self.key = key
	self.durable = durable
	self.autodel = autodel
	self.serverurl = serverurl
	go self.Connect()
}

//连接兔子，准备消费
func (self *RabbitMQConsume) Connect() bool {
	fmt.Println("Rabbitmq consum connect start ...")
	conn, err := amqp.Dial(self.serverurl)
	if err != nil {
		fmt.Println("创建连接错误:", err)
		self.reConnect()
		return false
	}
	//defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		fmt.Printf("打开Channel错误：", err)
		conn.Close()
		self.reConnect()
		return false
	}
	err = ch.ExchangeDeclare(
		self.exName,  // name
		"direct",     // type
		self.durable, // durable 是否持久，交换机和队列同时设置为持久的，就会持久化消息
		self.autodel, // auto-deleted 如果是TRUE，则没有一个连接绑定到该路由的时候，该路由就会自动删除
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		fmt.Println("创建路由错误：", err)
		ch.Close()
		conn.Close()
		self.reConnect()
		return false
	}
	q, err := ch.QueueDeclare(
		self.queueName, // name
		self.durable,   // durable 是否持久，交换机和队列同时设置为持久的，就会持久化消息
		self.autodel,   // delete when usused 如果是TRUE，则没有一个连接的时候，该队列就会自动删除
		false,          // exclusive  排他性，只有创建该队列的连接可以访问，该连接关闭后队列自动删除
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		fmt.Println("创建消息队列错误：", err)
		ch.Close()
		conn.Close()
		self.reConnect()
		return false
	}

	err = ch.QueueBind(q.Name, self.key, self.exName, false, nil)
	if err != nil {
		fmt.Println("绑定队列错误：", err)
		ch.Close()
		conn.Close()
		self.reConnect()
		return false
	}
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto ack
		false,  // exclusive
		false,  // no local
		false,  // no wait
		nil,    // args
	)
	if err != nil {
		fmt.Println("开始消费错误：", err)
		ch.Close()
		conn.Close()
		self.reConnect()
		return false
	}
	self.ch = ch
	go self.MsgLoop(msgs)
	return true
}
func (self *RabbitMQConsume) MsgLoop(msgs <-chan amqp.Delivery) {
	fmt.Println("连接成功，开始处理消息...")
	if self.handleback == nil {
		return
	}
	for {
		if d, ok := <-msgs; ok {
			self.handle(d.Body)
		} else {
			fmt.Printf("Rabbit server stop")
			self.reConnect()
			break
		}
	}
}

//严重错误捕获
func ErrorHandler() {
	if err := recover(); err != nil {
		fmt.Println("严重错误：", err)
	}
}

//在这里捕获painc异常，不影响上层函数的for循环
func (self *RabbitMQConsume) handle(data []byte) {
	defer ErrorHandler()
	self.handleback(data)
}

func (self *RabbitMQConsume) SendMsg(key string, msg []byte) {
	if self.ch != nil {
		err := self.ch.Publish(
			self.exName, // exchange
			key,         // routing key
			false,       // mandatory
			false,       // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        msg,
			})
		if err != nil {
			fmt.Println("发送失败:", err)
			//cachedata=append(cachedata,msg)
		}
	} else {
		//cachedata=append(cachedata,msg)
	}
}
