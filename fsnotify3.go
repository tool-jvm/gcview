package main;

import (
	"github.com/fsnotify/fsnotify"
	"log"
	"fmt"
	"os/exec"
		"strconv"
	"bytes"
		"os"
	"path/filepath"
	"strings"
	"bufio"
	)

const (
	confFilePath = "./";
)



//获取进程ID
func getPid() (int, error) {
	common := `ps -ef | grep gcmain | grep -v grep `
	cmd := exec.Command("/bin/bash", "-c", common)
	//创建获取命令输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error:can not obtain stdout pipe for command:%s\n", err)
	}

	fmt.Println("1")
	//执行命令
	if err := cmd.Start(); err != nil {
		fmt.Println("Error:The command is err,", err)
	}

	fmt.Println("2")
	//使用带缓冲的读取器
	outputBuf := bufio.NewReader(stdout)

	fmt.Println("3")
	var buffer bytes.Buffer


	//一次获取一行,_ 获取当前行是否被读完
	output, _, err := outputBuf.ReadLine()
	if err != nil {

		// 判断是否到文件的结尾了否则出错
		if err.Error() != "EOF" {
			fmt.Println("Error :%s\n", err)
		}
	}
	fmt.Println("4",string(output))
	buffer.WriteString(fmt.Sprintf("%s\n", string(output)))

	info := buffer.String()

	fmt.Println(info)
	fmt.Println("获取进程1：",info)
	info = Substr(info,10,20)
	fmt.Println("获取进程2：",info)
	arr :=strings.Split(info," ")
	fmt.Println("获取进程3：",arr[0])
	info = arr[0]
	return strconv.Atoi(info)
}


//启动进程
func startProcess(common string) error {
	fmt.Println("开始启动进程:",common)
	cmd := exec.Command("/bin/bash", "-c", common)

	//执行命令
	if err := cmd.Start(); err != nil {
		fmt.Println("Error:The command is err,", err)
		return nil
	}

	return nil;
}

func main() {
	//创建一个监控对象
	watch, err := fsnotify.NewWatcher();
	if err != nil {
		log.Fatal(err);
	}
	defer watch.Close();
	//添加要监控的文件
	err = watch.Add(confFilePath);
	if err != nil {
		log.Fatal(err);
	}
	fmt.Println( "开始监控文件");
	//我们另启一个goroutine来处理监控对象的事件
	go func() {
		for {
			select {
			case ev := <-watch.Events:
				{
					//我们只需关心文件的修改
					if ev.Op&fsnotify.Write == fsnotify.Write {
						fmt.Println(ev.Name, "文件写入");
						//查找进程
						pid, err := getPid();
						if pid == 0 {
							return
						}

						fmt.Println("获取进程ID:",pid)
						//获取运行文件的绝对路径
						exePath, _ := filepath.Abs("go build gcmain.go ; ./gcmain ")
						fmt.Println("执行命令:",exePath)
						if err != nil {
							//启动进程
							go startProcess(exePath);
						} else {
							//找到进程，并退出
							process, err := os.FindProcess(pid);
							if err == nil {
								//让进程退出
								process.Kill();
								fmt.Println(exePath, "进程退出");
							}else{
								fmt.Println("未找到:",exePath)
							}
							//启动进程
							go startProcess(exePath);
						}
					}
				}
			case err := <-watch.Errors:
				{
					fmt.Println("error : ", err);
					return;
				}
			}
		}
	}();

	//循环
	select {};
}

//截取字符串 start 起点下标 length 需要截取的长度
func Substr(str string, start int, length int) string {
	rs := []rune(str)
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

//截取字符串 start 起点下标 end 终点下标(不包括)
func Substr2(str string, start int, end int) string {
	rs := []rune(str)
	length := len(rs)

	if start < 0 || start > length {
		panic("start is wrong")
	}

	if end < 0 || end > length {
		panic("end is wrong")
	}

	return string(rs[start:end])
}