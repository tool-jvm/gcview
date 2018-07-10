package main

import (
	"fmt"
	"strings"
	"time"
	"net/http"
	"encoding/json"

	"golang.org/x/net/websocket"
	"bufio"
	"os/exec"
	"strconv"
)



//全局信息
var datas Datas
var users map[*websocket.Conn]string

func main() {

	fmt.Println("启动时间")
	fmt.Println(time.Now())

	//初始化
	datas = Datas{}
	users = make(map[*websocket.Conn]string)

	//调用命令
	callshell()

	//绑定效果页面
	http.HandleFunc("/", h_index)
	//绑定socket方法
	http.Handle("/webSocket", websocket.Handler(h_webSocket))
	//开始监听
	http.ListenAndServe(":8880", nil)


}


func h_index(w http.ResponseWriter, r *http.Request) {

	http.ServeFile(w, r, "index.html")
}

func h_webSocket(ws *websocket.Conn) {

	var userMsg UserMsg
	var data string
	for {

		//判断是否重复连接
		if _, ok := users[ws]; !ok {
			users[ws] = "匿名"
		}
		userMsgsLen := len(datas.UserMsgs)
		fmt.Println("UserMsgs", userMsgsLen, "users长度：", len(users))

		//有消息时，全部分发送数据
		if userMsgsLen > 0 {
			b, errMarshl := json.Marshal(datas)
			if errMarshl != nil {
				fmt.Println("全局消息内容异常...")
				break
			}
			for key, _ := range users {
				errMarshl = websocket.Message.Send(key, string(b))
				if errMarshl != nil {
					//移除出错的链接
					delete(users, key)
					fmt.Println("发送出错...")
					break
				}
			}
			datas.UserMsgs = make([]UserMsg, 0)
		}

		fmt.Println("开始解析数据...")
		err := websocket.Message.Receive(ws, &data)
		fmt.Println("data：", data)
		if err != nil {
			//移除出错的链接
			delete(users, ws)
			fmt.Println("接收出错...")
			break
		}

		data = strings.Replace(data, "\n", "", 0)
		err = json.Unmarshal([]byte(data), &userMsg)
		if err != nil {
			fmt.Println("解析数据异常...")
			break
		}
		fmt.Println("请求数据类型：", userMsg.DataType)

		switch userMsg.DataType {
		case "send":
			//赋值对应的昵称到ws
			if _, ok := users[ws]; ok {
				users[ws] = userMsg.UserName

				//清除连接人昵称信息
				datas.UserDatas = make([]UserData, 0)
				//重新加载当前在线连接人
				for _, item := range users {

					userData := UserData{UserName: item}
					datas.UserDatas = append(datas.UserDatas, userData)
				}
			}
			msg := callshellarg(userMsg.Msg)
			pid :=getMainPid(msg)
			msg = callgcview(pid)
			userMsg.Msg = userMsg.Msg+msg
			datas.UserMsgs = append(datas.UserMsgs, userMsg)
		}
	}

}

type UserMsg struct {
	UserName string
	Msg      string
	DataType string
}

type UserData struct {
	UserName string
}

type Datas struct {
	UserMsgs  []UserMsg
	UserDatas []UserData
}


func callshellarg(arg string) string {
	common := `ps -ef | grep -E 'sli|ecg' | grep -v tail | grep -v grep  | grep `+arg
	fmt.Println(common)
	cmd := exec.Command("/bin/bash", "-c", common)
	//创建获取命令输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error:can not obtain stdout pipe for command:%s\n", err)
		return ""
	}

	//执行命令
	if err := cmd.Start(); err != nil {
		fmt.Printf("Error:The command is err,", err)
		return ""
	}
	//使用带缓冲的读取器
	outputBuf := bufio.NewReader(stdout)
	var outstr string

	//一次获取一行,_ 获取当前行是否被读完
	output, _, err := outputBuf.ReadLine()
	if err != nil {
		// 判断是否到文件的结尾了否则出错
		if err.Error() != "EOF" {
			fmt.Printf("Error :%s\n", err)
		}
		return outstr
	}
	outstr = outstr + string(output)
	return outstr
}

func getMainPid(info string) (string) {
	info = Substr(info,10,20)
	arr :=strings.Split(info," ")
	return arr[0]
}

func callgcview(pid string	) string  {
	var outstr string
	common := "jstat -gc "+pid
	cmd := exec.Command("/bin/bash","-c",common)
	//创建获取命令输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error:can not obtain stdout pipe for command:%s\n", err)
		return outstr
	}
	//执行命令
	if err := cmd.Start(); err != nil {
		fmt.Println("Error:The command is err,", err)
		return outstr
	}
	//使用带缓冲的读取器
	outputBuf := bufio.NewReader(stdout)
	outstr = " <br/><table>"

	var arrTitle [17]string
	var arrValue [17] string
	var flag = 0
	for {

		//一次获取一行,_ 获取当前行是否被读完
		output, _, err := outputBuf.ReadLine()
		if err != nil {

			// 判断是否到文件的结尾了否则出错
			if err.Error() != "EOF" {
				fmt.Printf("Error :%s\n", err)
			}
			break
		}
		item := string(output)
		arr := strings.Split(item," ")

		var idx = 0
		if flag == 0 {
			for ar := range arr{
				temp := arr[ar]
				temp = strings.Trim(temp," ")
				if len(temp) > 0 {
					arrTitle[idx] = temp
					idx++
				}
			}
			flag = 1
		}else{
			for ar := range arr{
				temp := arr[ar]
				temp = strings.Trim(temp," ")
				if len(temp) > 0 {
					arrValue[idx] = temp
					idx++
				}
			}
		}


	}

	for idx := range arrTitle{
		outstr = outstr + "<tr>"
		title := arrTitle[idx]

		// 转换title
		title,rat := setTitle(title,arrTitle,arrValue,idx)

		fmt.Println("idx:",idx," title:",title," rat:",rat)
		outstr = outstr + "<td>" + title + "</td>"

		valu := arrValue[idx]
		dat, _ := strconv.ParseFloat(valu,32)
		if dat > 1000 {
			dat = dat / 1000
			outstr = outstr + "<td>" + strconv.FormatFloat(dat,'f',2,64) + "MB</td>"
		}else{
			outstr = outstr + "<td>" + arrValue[idx] + "KB</td>"
		}

		outstr = outstr + "<td>" + strconv.FormatFloat(rat,'f',2,64) + "%</td>"
		outstr = outstr + "</tr>"
	}

	outstr = outstr + "</table>"
	fmt.Printf("%s\n", outstr)
	//wait 方法会一直阻塞到其所属的命令完全运行结束为止
	if err := cmd.Wait(); err != nil {
		fmt.Println("wait:", err.Error())
	}
	return outstr
}

// 转换title
func setTitle(title string,arrTitle [17]string,arrValue [17]string,idx int) (string,float64) {
	if title == "S0C" { //0
		title = "第一个幸存区的大小"
	} else if title == "S1C" { //1
		title = "第二个幸存区的大小"
	} else if title == "S0U" {//2
		title = "第一个幸存区的使用大小"
		da := calRate(arrValue, idx,0)
		return title, da
	} else if title == "S1U" { //3
		title = "第二个幸存区的使用大小"
		da := calRate(arrValue, idx,2)
		return title, da
	} else if title == "EC" { //4
		title = "伊甸园区的大小"
	} else if title == "EU" { //5
		title = "伊甸园区的使用大小"
		da := calRate(arrValue, idx,4)
		return title, da
	} else if title == "OC" { //6
		title = "老年代大小"
	} else if title == "OU" { //7
		title = "老年代使用大小"
		da := calRate(arrValue, idx,6)
		return title, da
	} else if title == "MC" { //8
		title = "方法区大小"
	} else if title == "MU" { //9
		title = "方法区使用大小"
		da := calRate(arrValue, idx,8)
		return title, da
	} else if title == "CCSC" { //10
		title = "压缩类空间大小"
	} else if title == "CCSU" { //11
		title = "压缩类空间使用大小"
		da := calRate(arrValue, idx,10)
		return title, da
	} else if title == "YGC" { //12
		title = "年轻代GC活动的数量"
	} else if title == "YGCT" { //13
		title = "年轻代垃圾收集时间"
	} else if title == "FGC" { //14
		title = "full GC事件的数量"
	} else if title == "FGCT" { //15
		title = "full GC垃圾收集时间。"
	} else if title == "GCT" { //16
		title = "垃圾收集总时间"
	}
	return title, 0
}

func calRate(arrValue [17]string, idx int,start int) float64 {
	soct := arrValue[start]
	socs := arrValue[idx]
	dat, _ := strconv.ParseFloat(soct,32)
	das, _ := strconv.ParseFloat(socs,32)
	da := 0.0
	if dat > 0 {
		da = (das/dat)*100
	}
	fmt.Println(socs," / ",soct,"=",da)
	return da
}


func callshell()  {
	cmd := exec.Command("/bin/bash", "-c", `ps -ef | grep -E 'sli|ecg' | grep -v grep `)

	//创建获取命令输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error:can not obtain stdout pipe for command:%s\n", err)
		return
	}

	//执行命令
	if err := cmd.Start(); err != nil {
		fmt.Println("Error:The command is err,", err)
		return
	}

	//使用带缓冲的读取器
	outputBuf := bufio.NewReader(stdout)

	for {

		//一次获取一行,_ 获取当前行是否被读完
		output, _, err := outputBuf.ReadLine()
		if err != nil {

			// 判断是否到文件的结尾了否则出错
			if err.Error() != "EOF" {
				fmt.Printf("Error :%s\n", err)
			}
			return
		}
		fmt.Printf("%s\n", string(output))

	}

	//wait 方法会一直阻塞到其所属的命令完全运行结束为止
	if err := cmd.Wait(); err != nil {
		fmt.Println("wait:", err.Error())
		return
	}
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