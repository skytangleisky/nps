package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/gorilla/websocket"
	"net/http"
	"regexp"
	"strconv"
	"unsafe"
)

/*
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)

var upgrader = websocket.Upgrader{}

type Message struct {
	Message string `json:"message"`
}

func main() {
	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", fs)

	http.HandleFunc("/debug", handleConnections)

	go handleMessages()

	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

//注册成为 websocket
func handleConnections(w http.ResponseWriter, r *http.Request) {

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	clients[ws] = true

	//不断的从页面上获取数据 然后广播发送出去
	for {
		//将从页面上接收数据改为不接收 直接发送
		//var msg Message
		//err := ws.ReadJSON(&msg)
		//if err != nil {
		//  log.Printf("error: %v", err)
		//  delete(clients, ws)
		//  break
		//}

		//目前存在问题 定时效果不好 需要在业务代码替换时改为beego toolbox中的定时器
		time.Sleep(time.Second * 3)
		msg := Message{Message: "这是向页面发送的数据 " + time.Now().Format("2006-01-02 15:04:05")}
		broadcast <- msg
	}
}

//广播发送至页面
func handleMessages() {
	for {
		msg := <-broadcast
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("client.WriteJSON error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
*/

type DebugController struct {
	beego.Controller
}

// 连接的客户端,把每个客户端都放进来
var clients = make(map[*websocket.Conn]bool)

// 广播频道(通道)
var broadcast = make(chan []MyMessage)

// 配置升级程序(升级为websocket)
var upgrader = websocket.Upgrader{}

// 定义我们的消息对象
type Message struct {
	Data interface{} `json:"data"`
}

type MyMessage struct {
	Background string `json:"background,omitempty"`
	Color      string `json:"color"`
	Message    string `json:"message"`
}

func (c *DebugController) Debug() {
	// 解决跨域问题(微信小程序)
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	//升级将HTTP服务器连接升级到WebSocket协议。
	//responseHeader包含在对客户端升级的响应中
	//请求。使用responseHeader指定Cookie（设置Cookie）和
	//应用程序协商的子目录（Sec WebSocket协议）。
	//如果升级失败，则升级将向客户端答复一个HTTP错误
	ws, err := upgrader.Upgrade(c.Ctx.ResponseWriter, c.Ctx.Request, nil)
	if err != nil {
		logs.Error(err)
		return
	}
	//defer ws.Close()

	//将当前客户端放入map中
	clients[ws] = true
	c.EnableRender = false //Beego不启用渲染
	//go func() {
	//	logs.Debug("\u001B[1;32mCONNECTED\u001B[0m", len(clients))
	//	for {
	//		_, d, err := ws.ReadMessage()
	//		if err != nil {
	//			//logs.Error(err.Error())
	//			ws.Close()
	//			delete(clients, ws) //删除map中的客户端
	//			logs.Debug("\u001B[1;31mDISCONNECTED\u001B[0m",len(clients))
	//			break //结束循环
	//		} else {
	//			logs.Debug("\u001B[1;34m\n"+bytes2str(d)+"\u001B[0m")//接受消息 业务逻辑
	//		}
	//	}
	//}()
	var myMessages = make([]MyMessage, 0)
	myMessages = append(myMessages, MyMessage{"", "00ff00", "CONNECTED "})
	myMessages = append(myMessages, MyMessage{"", "bbbbbb", strconv.Itoa(len(clients)) + "\n"})
	broadcast <- myMessages
	go func() {
		for {
			_, d, err := ws.ReadMessage()
			if err != nil {
				//logs.Error(err.Error())
				ws.Close()
				delete(clients, ws) //删除map中的客户端
				var myMessages = make([]MyMessage, 0)
				myMessages = append(myMessages, MyMessage{"", "ff0000", "DISCONNECTED "})
				myMessages = append(myMessages, MyMessage{"", "bbbbbb", strconv.Itoa(len(clients)) + "\n"})
				broadcast <- myMessages
				break //结束循环
			} else {
				if bytes2str(d) == "HEART" {
					continue
				}
				var myMessages = make([]MyMessage, 0)
				myMessages = append(myMessages, MyMessage{"", "00ff00", bytes2str(d) + "\n"})
				broadcast <- myMessages
			}
		}
	}()

}
func PrintMessage(b []byte) {
	str := bytes2str(b)
	myMessages := decodeANSI(str)
	broadcast <- myMessages
}

func decodeANSI(strANSI string) []MyMessage {
	var myMessages = make([]MyMessage, 0)
	ss := "\u001B[0m" + strANSI
	arrStr := regexp.MustCompile("\u001B\\[([0-9]?[0-9][;])?[1-9]?[0-9]m").Split(ss, -1)
	colorStr := regexp.MustCompile("\u001B\\[([0-9]?[0-9][;])?[1-9]?[0-9]m").FindAllString(ss, -1)
	for i, value := range colorStr {
		message := MyMessage{"", "2B2B2B", arrStr[i+1]}
		//fmt.Println(value[1:])
		switch value {
		case "\u001B[0m":
			message.Color = "BBBBBB"
			break
		case "\u001B[1;34m":
			message.Color = "1FB0FF"
			break
		case "\u001B[1;37m":
			message.Color = "FFFFFF"
			break
		case "\u001B[1;36m":
			message.Color = "00E5E5"
			break
		case "\u001B[1;35m":
			message.Color = "ED7EED"
			break
		case "\u001B[1;31m":
			message.Color = "FF4050"
			break
		case "\u001B[1;33m":
			message.Color = "E5BF00"
			break
		case "\u001B[1;32m":
			message.Color = "4FC414"
			break
		case "\u001B[1;44m":
			message.Color = "FFFFFF"
			message.Background = "1778BD"
			break
		case "\u001B[41m":
			message.Color = "FF4050"
			message.Background = "772E2C"
			break
		case "\u001B[42m":
			message.Color = "4FC414"
			message.Background = "458500"
			break
		case "\u001B[44m":
			message.Color = "1FB0FF"
			message.Background = "1778BD"
			break
		case "\u001B[47m":
			message.Color = "808080"
			message.Background = "616161"
			break
		case "\u001B[46m":
			message.Color = "00E5E5"
			message.Background = "006E6E"
			break
		case "\u001B[43m":
			message.Color = "E5BF00"
			message.Background = "A87B00"
			break
		case "\u001B[45m":
			message.Color = "ED7EED"
			message.Background = "458500"
			break
		case "\u001B[97;42m":
			message.Color = "FFFFFF"
			message.Background = "39511F"
			break
		case "\u001B[97;43m":
			message.Color = "FFFFFF"
			message.Background = "5C4F17"
			break
		case "\u001B[97;44m":
			message.Color = "FFFFFF"
			message.Background = "245980"
			break
		case "\u001B[97;46m":
			message.Color = "FFFFFF"
			message.Background = "154F4F"
			break
		case "\u001B[90;47m":
			message.Color = "808080"
			message.Background = "616161"
			break
		default:
			logs.Warn("未定义颜色：\\u001B" + value[1:])
		}
		myMessages = append(myMessages, message)
	}
	return myMessages
}

var l []MyMessage

func init() {
	go func() {
		for {
			//读取通道中的消息
			msg := <-broadcast
			for _, v := range msg {
				l = append(l, v)
			}
			if len(l) > 500 {
				l = l[len(l)-500:]
			}
			for client := range clients {
				//把通道中的消息发送给客户端
				//fmt.Println("tanglei=", MyMessage{"FFff0000","abcdef\n"}.Color)
				err := client.WriteJSON(l)
				//err:= client.WriteMessage(websocket.TextMessage,str2bytes(`[{"color":"ffff0000","message":"\n123456789\n"}]`))
				if err != nil {
					logs.Warn("client.WriteJSON error: %v", err)
					client.Close()          //关闭
					delete(clients, client) //删除map中的客户端
				}
			}
			if len(clients) > 0 {
				l = l[:0:0]
			}
		}
	}()
}

/*普通string<->byte*/
//var b = []byte("hello boy")
//var str = string(str)

/*优化后的string<->byte*/
func str2bytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

func bytes2str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
