package util

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"unicode/utf8"
	"strings"

)

//unicode 解码
func DecodeUnicode(data []byte) string {
	if data == nil {
		return ""
	}
	length := len(data)
	if length == 0 {
		return ""
	}
	all := make([]rune, 0)
	var s, t int = 0, 0
	var r rune
	for s < length {
		r, t = utf8.DecodeRune(data[s:])
		s += t
		all = append(all, r)
	}
	return string(all)
}

//检查是否错误
func CheckError(errs ...error) bool {
	for _, er := range errs {
		if er != nil {
			fmt.Println("参数错误！", er)
			return true
		}
	}
	return false
}

//MD5加密
func MD5Str(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

//获取GUID
func GetGuid() string {
	b := make([]byte, 48)

	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return MD5Str(base64.URLEncoding.EncodeToString(b))
}

type String string

//字符串截取
func (s *String) subStr(start, length int) string {
	rs := []rune(string(*s))
	rl := len(rs)
	end := 0
	if start < 0 {
		start = rl - 1 + start
	}
	end = start + length
	if start > end {
		start, end = end, start
	}
	if start < 0 {
		start = 0
	}
	if start > rl {
		start = rl
	}
	if end < 0 {
		end = 0
	}
	if end > rl {
		end = rl
	}
	return string(rs[start:end])
}


//获取命令行的第一个参数
func GetFirstArg() string {
	arg_num := len(os.Args)
	if arg_num > 1 && os.Args[1] != SUB_COMMAND {
		return os.Args[1]
	}
	return ""
}

//子任务进程标示
const SUB_COMMAND = "-childproc"

//开启守护进程 ，父进程监护子进程 , 适用linux系统
func Fork() bool {
	if strings.ToLower(runtime.GOOS) != "windows" {
		//将当前进程修改为守护进程 让其被1号进程接管  则 父进程ID为1
		//父进程不是1且又不是任务子进程，则是首次启动进程，启动监控进程（监控进程pid为1），退出当前进程
		if os.Getppid() != 1 && os.Args[len(os.Args)-1] != SUB_COMMAND {

			//将命令行参数中执行文件路径转换成可用路径
			filePath, _ := filepath.Abs(os.Args[0])
			cmd := exec.Command(filePath, os.Args[1:]...)
			//将其他命令传入生成出的进程
			//给新进程设置文件描述符，可以重定向到文件中
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			//开始执行新进程，不等待新进程退出
			cmd.Start()
			fmt.Println("Keeper started !")
			return false
		}

		//监护进程，监护进程的目的是放在子进程意外挂掉，子进程挂掉后由监护进程再次启动
		if os.Getppid() == 1 {
			//父进程
			//将命令行参数中执行文件路径转换成可用路径
			filePath, _ := filepath.Abs(os.Args[0])
			for {
				cmd := exec.Command(filePath, append(os.Args[1:], SUB_COMMAND)...)
				//开始执行新进程
				cmd.Start()
				//等待新的子进程退出
				err := cmd.Wait()
				if err != nil {

				} else {

					return false
				}
			}
		}

	}
	return true
}
