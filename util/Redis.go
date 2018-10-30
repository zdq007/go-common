package util

import (
	"fmt"
	"github.com/gomodule/redigo"
	"time"
)

type RedisPool struct {
	Pool *redis.Pool
}

var conn redis.Conn

// 重写生成连接池方法
func NewRedisPool(server, password *string) *RedisPool {
	pool := &RedisPool{
		Pool: &redis.Pool{
			MaxIdle:     6,                 //最大空闲活动数
			MaxActive:   30,                //最大允许连接数，连接不够的时候最多允许创建的连接数
			Wait:        true,              //获取不到连接堵塞线程，不堵塞线程将会获取到空值，注意！！
			IdleTimeout: 240 * time.Second, //连接失效检测时间
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", *server)
				if err != nil {
					return nil, err
				}
				if password != nil {
					if _, err := c.Do("AUTH", password); err != nil {
						c.Close()
						return nil, err
					}
				}
				return c, err
			},
			//			TestOnBorrow: func(c redis.Conn, t time.Time) error { //检测命令
			//				_,err:=c.Do("PING")
			//				return err
			//			},
		},
	}
	conn = pool.Pool.Get()
	return pool
}

//设置
func (self *RedisPool) SET(key string, val interface{}) {
	c := self.Pool.Get()
	defer c.Close()
	c.Send("SET", key, val)
}

//批量设置
func (self *RedisPool) MSET(params ...interface{}) {
	c := self.Pool.Get()
	defer c.Close()
	c.Send("MSET", params...)
}

//hset 设置
func (self *RedisPool) HSET(key string, filed, val interface{}) error {
	c := self.Pool.Get()
	defer c.Close()
	err := c.Send("HSET", key, filed, val)
	return err
}
func (self *RedisPool) HMSET(params ...interface{}) error {
	c := self.Pool.Get()
	defer c.Close()
	err := c.Send("HMSET", params...)
	return err
}

//获取
func (self *RedisPool) GET(key string) []byte {
	c := self.Pool.Get()
	defer c.Close()
	data, err := c.Do("GET", key)
	if err != nil {
		fmt.Println(err)
		return nil
	} else {
		if data == nil {
			return nil
		}
		return data.([]byte)
	}
}

//hset 获取
func (self *RedisPool) HGET(key, filed string) []byte {
	c := self.Pool.Get()
	defer c.Close()
	data, err := c.Do("HGET", key, filed)
	if err != nil {
		return nil
	} else {
		if data == nil {
			return nil
		}
		return data.([]byte)
	}
}

//HMGET
func (self *RedisPool) HMGET(params ...interface{}) []interface{}{
	c := self.Pool.Get()
	defer c.Close()
	data, err := c.Do("HMGET", params...)
	if err != nil {
		return nil
	} else {
		if data == nil {
			return nil
		}
		return data.([]interface{})
	}
}

//hset 获取所有
func (self *RedisPool) HGETALL(key string) map[string][]byte {
	c := self.Pool.Get()
	defer c.Close()
	data, err := c.Do("HGETALL", key)
	if err != nil {
		return nil
	} else {
		arrdata := data.([]interface{})
		datamap := make(map[string][]byte, len(arrdata)/2)
		var filed string
		for i, val := range arrdata {
			v := val.([]byte)
			if i%2 == 0 { //偶数 为filed
				filed = string(v)
			} else { //奇数 为值
				datamap[filed] = v
			}
		}
		return datamap
	}
}

//删除KEY
func (self *RedisPool) DEL(keys ...interface{}) {
	c := self.Pool.Get()
	defer c.Close()
	c.Send("DEL", keys...)
}

//hset 删除
func (self *RedisPool) HDEL(key string, filed interface{}) {
	c := self.Pool.Get()
	defer c.Close()
	c.Send("HDEL", key, filed)
}

//设置超时 dtime 秒
func (self *RedisPool) EXPIRE(key string, dtime int32) {
	c := self.Pool.Get()
	defer c.Close()
	c.Send("EXPIRE", key, dtime)
}

//获取所有的KEY
func (self *RedisPool) KEYS(key string) []interface{} {
	c := self.Pool.Get()
	defer c.Close()
	d, err := c.Do("KEYS", key)
	if err != nil {
		return nil
	} else {
		return d.([]interface{})
	}
}

//检测是否存在
func (self *RedisPool) EXISTS(key string) interface{} {
	c := self.Pool.Get()
	defer c.Close()
	d, err := c.Do("EXISTS", key)
	if err != nil {
		return nil
	} else {
		return d
	}
}

//订阅
func (self *RedisPool) SUBSCRIBE(chanl string, handel func(data []byte)) {
	go func() {
		c := self.Pool.Get()
		c.Send("SUBSCRIBE", chanl)
		c.Flush()
		for {
			reply, err := c.Receive()
			if err == nil {
				rdata := reply.([]interface{})
				switch string(rdata[0].([]byte)) {
				case "message":
					//fmt.Println("接收到：%s", rdata[2].([]byte))
					handel(rdata[2].([]byte))
				case "subscribe":
					//fmt.Println("subscribe")
				}
			} else {
				c.Close()
				fmt.Println("err:", err)
				//time.Sleep(time.Second * 6)
				self.SUBSCRIBE(chanl, handel)
				return
			}
		}
	}()
}

//订阅
func (self *RedisPool) Subscribe(chanl string, handel func(data []byte)) {
	go func() {
		c := self.Pool.Get()
		psc := redis.PubSubConn{c}
		psc.Subscribe(chanl)
		for {
			switch v := psc.Receive().(type) {
			case redis.Message:
				//fmt.Printf("%s: message: %s\n", v.Channel, v.Data)
				handel(v.Data)
			case redis.Subscription:
				//fmt.Printf("%s: %s %d\n", v.Channel, v.Kind, v.Count)
			case error:
				c.Close()
				fmt.Println("err:", v)
				time.Sleep(time.Second * 6)
				self.Subscribe(chanl, handel)
				return
			}
		}
	}()
}

//发布
func (self *RedisPool) PUBLISH(chanl string, value interface{}) {
	c := self.Pool.Get()
	defer c.Close()
	c.Send("PUBLISH", chanl, value)
}

func redisServer(c redis.Conn) {
	startTime := time.Now()
	// 从连接池里面获得一个连接
	fmt.Println(c)
	// 连接完关闭，其实没有关闭，是放回池里，也就是队列里面，等待下一个重用
	if c != nil {
		defer c.Close()
		dbkey := "netgame:info"
		fmt.Println(c.Do("KEYS", "*"))
		if ok, err := redis.Bool(c.Do("LPUSH", dbkey, "yangzetao")); ok {
		} else {
			fmt.Println(err)
		}
		msg := fmt.Sprintf("用时：%s", time.Now().Sub(startTime))
		fmt.Println(msg)
	}
}
