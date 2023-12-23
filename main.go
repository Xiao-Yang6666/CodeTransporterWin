package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/atotto/clipboard"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/getlantern/systray"
	"github.com/go-toast/toast"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
)

// SmsMessage 定义一个结构体来映射JSON数据
type SmsMessage struct {
	Sender      string `json:"sender"`
	SmsCode     string `json:"smsCode"`
	PhoneNumber string `json:"phoneNumber"`
	SmsMsg      string `json:"smsMsg"`
}

// Config 定义配置文件结构
type Config struct {
	Broker string `yaml:"broker"`
	Topic  string `yaml:"topic"`
}

var cfg Config
var logger *log.Logger

func init() {
	// 初始化日志
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("打开日志文件失败:", err)
	}
	logger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	// 读取配置文件
	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		logger.Fatalf("读取配置文件失败: %v", err)
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		logger.Fatalf("解析配置文件失败: %v", err)
	}
}

// 当收到消息时的处理函数
var messagePubHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	var smsMsg SmsMessage
	err := json.Unmarshal(msg.Payload(), &smsMsg)
	if err != nil {
		logger.Printf("消息解析错误: %v\n", err)
		return
	}

	if smsMsg.SmsCode != "" {
		logger.Printf("接收到消息: 发送者: %s, 验证码: %s, 手机号: %s\n", smsMsg.Sender, smsMsg.SmsCode, smsMsg.PhoneNumber)
		_ = clipboard.WriteAll(smsMsg.SmsCode)
		notification := toast.Notification{
			AppID:   "手机消息接收器",
			Title:   fmt.Sprintf("来自: %s", smsMsg.PhoneNumber),
			Message: fmt.Sprintf("验证码: %s\n发送者: %s", smsMsg.SmsCode, smsMsg.Sender),
		}
		_ = notification.Push()
	} else {
		logger.Printf("接收到消息: 短信内容: %s, 手机号: %s\n", smsMsg.SmsMsg, smsMsg.PhoneNumber)
		notification := toast.Notification{
			AppID:   "手机消息接收器",
			Title:   fmt.Sprintf("来自: %s", smsMsg.PhoneNumber),
			Message: fmt.Sprintf("内容: %s", smsMsg.SmsMsg),
		}
		_ = notification.Push()
	}
}

var connectHandler MQTT.OnConnectHandler = func(client MQTT.Client) {
	logger.Println("已连接")
}

var connectLostHandler MQTT.ConnectionLostHandler = func(client MQTT.Client, err error) {
	logger.Printf("连接丢失: %v", err)
}

func generateRandomClientID(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// 处理错误
		logger.Fatalf("生成随机客户端ID失败: %v", err)
	}
	return hex.EncodeToString(bytes)
}

func main() {
	// MQTT订阅
	opts := MQTT.NewClientOptions().AddBroker(cfg.Broker)
	randomClientID := generateRandomClientID(10) // 生成10字节长度的随机ID
	opts.SetClientID(randomClientID)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	subscribe(client, cfg.Topic)

	// 运行systray
	systray.Run(onReady, onExit)
}

func subscribe(client MQTT.Client, topic string) {
	token := client.Subscribe(topic, 1, nil)
	token.Wait()
	logger.Printf("已订阅主题: %s\n", topic)
}

func onReady() {
	//systray.SetIcon(IconData) // 设置托盘图标, iconData应该是图标的字节数组
	systray.SetTitle("手机消息接收器")
	systray.SetTooltip("手机消息接收器")

	// 添加菜单项
	mQuit := systray.AddMenuItem("关闭", "关闭程序")

	// 菜单事件循环
	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				systray.Quit() // 退出程序
				log.Println("退出程序")
				return
			}
		}
	}()
}

func onExit() {
	// 清理工作
}
