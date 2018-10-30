package dao

import (
	"database/sql"
	"encoding/json"
	"strconv"

	"github.com/donnie4w/go-logger/logger"
	_ "github.com/go-sql-driver/mysql"
	. "github.com/jinzhu/gorm"
)

var (
	DAO *Dao
)

//解析为json时对应第字段类型
const (
	NUM = iota
	STR
)

func init() {
	DAO = new(Dao)
}

type Dao struct {
	*DB
	server string
	user   string
	pwd    string
	dbname string
}

func (self *Dao) init(server, user, pwd, dbname string) {
	self.server = server
	self.user = user
	self.pwd = pwd
	self.dbname = dbname
}

/**
1.程序连接数据库会有连接泄漏的情况，需要及时释放连接
2.Go sql包中的Query和QueryRow两个方法的连接不会自动释放连接，只有在遍历完结果或者调用close方法才会关闭连接
3.Go sql中的Ping和Exec方法在调用结束以后就会自动释放连接
4.忽略了函数的某个返回值不代表这个值就不存在了，如果该返回值需要close才会释放资源，直接忽略了就会导致资源的泄漏。
5.有close方法的变量，在使用后要及时调用该方法，释放资源
*/

func (self *Dao) OpenDB(server string, user string, pwd string, dbname string, idle, max int, showsql bool) error {
	self.init(server, user, pwd, dbname)
	//&loc=Local&parseTime=True  如果想将查询的date  datetime timestep 类型的参数返回给 time.Time类型时候设置，默认返回[]byte/string
	dburl := self.user + ":" + self.pwd + "@tcp(" + self.server + ")/" + self.dbname + "?charset=utf8"
	db, err := Open("mysql", dburl)
	self.DB = db
	if err != nil {
		logger.Error("打开数据库异常：", err)
	}

	// Then you could invoke `*sql.DB`'s functions with it
	err = self.DB.DB().Ping()
	if err != nil {
		logger.Error("连接数据库异常：", err)
	}
	//初始连接数
	self.DB.DB().SetMaxIdleConns(idle)
	//最大连接数
	self.DB.DB().SetMaxOpenConns(max)

	// Disable table name's pluralization
	self.SingularTable(true)

	//显示sql
	self.LogMode(showsql)

	return err
}

//查询一条记录 采用回调，在高调用函数的时候最好少采用回调函数的方式
func (self *Dao) QueryOneRowCallback(backfn func(row *sql.Row), sql string, args ...interface{}) {
	row := self.Raw(sql, args...).Row()
	backfn(row)
}

//查询一条记录
func (self *Dao) QueryOneRow(sql string, args ...interface{}) *sql.Row {
	row := self.Raw(sql, args...).Row()
	return row
}

//查询多条记录，在高调用函数的时候最好少采用回调函数的方式
func (self *Dao) QueryRowsCallback(backfn func(rows *sql.Rows), sql string, args ...interface{}) {
	rows, err := self.Raw(sql, args...).Rows()
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		logger.Error("未查询到数据:", err)
	} else {
		for rows.Next() {
			backfn(rows)
		}
	}
}

//查询多条记录 （记得释放rows）
func (self *Dao) QueryRows(sql string, args ...interface{}) (*sql.Rows, error) {
	return self.Raw(sql, args...).Rows()
}

//统计记录数
func (self *Dao) QueryCount(sql string, args ...interface{}) int64 {
	var count int64
	self.Raw(sql, args...).Count(&count)
	return count
}

//查询一个字段
func (self *Dao) QueryOneField(sql string, args ...interface{}) (res interface{}) {
	self.Raw(sql, args...).Row().Scan(&res)
	return
}

//查找返回JSON数组 [{},{},{}]  ，因为mysql库查询返回的大部分类型都是[]BYTE  所以不能通过类型判断，只能由用户指定类型
func (self *Dao) QueryArray(typeFlag []int, sqlstr string, args ...interface{}) ([]interface{}, error) {
	rows, err := self.Raw(sqlstr, args...).Rows()
	defer rows.Close()
	if err == nil {
		columns, err := rows.Columns()
		if err == nil {
			values := make([]sql.RawBytes, len(columns)) //sql.RawBytes
			scanArgs := make([]interface{}, len(values))
			for i := range values {
				scanArgs[i] = &values[i]
			}
			result := make([]interface{}, 0)
			for rows.Next() {
				err = rows.Scan(scanArgs...)
				if err != nil {
					panic(err.Error())
				}
				record := make(map[string]interface{})

				for i, col := range values {
					if col == nil {
						if typeFlag[i] == STR {
							record[columns[i]] = ""
						} else {
							record[columns[i]] = 0
						}
					} else {
						if typeFlag[i] == NUM {
							n, err := strconv.ParseInt(string(col), 10, 64)
							if err != nil {
								n = 0
							}
							record[columns[i]] = n
						} else {
							record[columns[i]] = string(col)
						}
					}
				}

				result = append(result, record)
			}

			return result, nil
		}
	}
	return nil, err
}

//查找返回数组 [a,b,c]
func (self *Dao) QueryArr(typeFlag []int, sqlstr string, args ...interface{}) ([]interface{}, error) {
	rows, err := self.Raw(sqlstr, args...).Rows()
	defer rows.Close()
	if err == nil {
		columns, err := rows.Columns()
		if err == nil {
			values := make([]sql.RawBytes, len(columns)) //sql.RawBytes
			scanArgs := make([]interface{}, len(values))
			for i := range values {
				scanArgs[i] = &values[i]
			}
			result := make([]interface{}, 0)
			for rows.Next() {
				err = rows.Scan(scanArgs...)
				if err != nil {
					panic(err.Error())
				}
				for i, col := range values {
					if col == nil {
						if typeFlag[i] == STR {
							result = append(result, "")
						} else {
							result = append(result, 0)
						}
					} else {
						if typeFlag[i] == NUM {
							n, err := strconv.ParseInt(string(col), 10, 64)
							if err != nil {
								n = 0
							}
							result = append(result, n)
						} else {
							result = append(result, string(col))
						}
					}
				}
			}

			return result, nil
		}
	}
	return nil, err
}

//查询记录并返回json数组字符串
func (self *Dao) QueryJSON(typeFlag []int, sql string, args ...interface{}) ([]byte, error) {
	result, err := self.QueryArray(typeFlag, sql, args...)
	if err == nil {
		if len(result) == 0 {
			return []byte("[]"), nil
		}
		return json.Marshal(result)
	} else {
		return nil, err
	}
}

//执行修改 删除 等操作
func (self *Dao) Execute(sql string, arges ...interface{}) int64 {
	return self.Exec(sql, arges...).RowsAffected
}

//执行新增 ，返回主键id
func (self *Dao) Save(sql string, arges ...interface{}) (int64, error) {
	res, err := self.DB.DB().Exec(sql, arges...)
	if err != nil {
		return -1, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return -1, err
	}
	return id, nil
}
