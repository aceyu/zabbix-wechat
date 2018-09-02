package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/astaxie/beego/cache"
)

type MsgInfo struct {
	//消息属性和内容
	Touser, Toparty, Corpid, Corpsecret, Title, Msg, Url, Log string
	Agentid                                                   int
}

var msgInfo MsgInfo
var filecache cache.Cache

type TextMsg struct {
	Content string `json:"content"`
}

type TextcardMsg struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Url         string `json:"url"`
	Btntxt      string `json:"btntxt"`
}

type WechatMsg struct {
	Touser      string       `json:"touser"`
	Toparty     string       `json:"toparty"`
	Totag       string       `json:"totag"`
	Msgtype     string       `json:"msgtype"`
	Agentid     int          `json:"agentid"`
	TextcardMsg *TextcardMsg `json:"textcard"`
	Safe        int          `json:"safe"`
}

func init() {
	flag.StringVar(&msgInfo.Touser, "touser", "", "消息的接收人，可以在微信后台查看，可空。")
	flag.StringVar(&msgInfo.Toparty, "toparty", "", "消息的接收组，可以在微信后台查看，可空。")
	flag.IntVar(&msgInfo.Agentid, "agentid", 1, "AgentID，可以在微信后台查看，不可空。")
	flag.StringVar(&msgInfo.Corpid, "corpid", "", "CorpID，可以在微信后台查看，不可空。")
	flag.StringVar(&msgInfo.Corpsecret, "corpsecret", "", "CorpSecret，可以在微信后台查看，不可空。")
	flag.StringVar(&msgInfo.Title, "title", "监控报警", "消息标题, 不可空。")
	flag.StringVar(&msgInfo.Msg, "msg", "", "消息体, 不可空。")
	flag.StringVar(&msgInfo.Url, "url", "http://10.0.1.11/zabbix", "消息内容点击后跳转到的URL，可空。")
	flag.StringVar(&msgInfo.Log, "log", "/tmp/wechat.log", "日志路径，可空。")
	flag.Parse()
	filecache, _ = cache.NewCache("file", `{}`)
	logFile, logErr := os.OpenFile(msgInfo.Log, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if logErr != nil {
		fmt.Println("Fail to find", *logFile, "Start Failed")
		os.Exit(1)
	}
	log.SetOutput(logFile)
	log.Println("初始化完成。")
}

type ResAccessToken struct {
	ErrCode     int64  `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

func getToken(corpid, corpsecret string, agentid int) (accessToken string) { //根据id和secret获取AccessToken
	fmt.Println(corpid, corpsecret)
	accessTokenCacheKey := fmt.Sprintf("access_token_%d", agentid)
	val := filecache.Get(accessTokenCacheKey)
	if val != nil && val.(string) != "" {
		accessToken = val.(string)
		return
	}

	//从微信服务器获取
	var resAccessToken ResAccessToken
	url := fmt.Sprintf("%s?corpid=%s&corpsecret=%s", "https://qyapi.weixin.qq.com/cgi-bin/gettoken", corpid, corpsecret)
	var body []byte
	response, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		log.Printf("http get error : uri=%v , statusCode=%v", url, response.StatusCode)
		return
	}
	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return
	}
	err = json.Unmarshal(body, &resAccessToken)
	if err != nil {
		log.Println(err)
		return
	}
	if resAccessToken.ErrMsg != "ok" {
		log.Printf("get access_token error : errcode=%v , errormsg=%v", resAccessToken.ErrCode, resAccessToken.ErrMsg)
		return
	}
	expires := resAccessToken.ExpiresIn - 1500
	err = filecache.Put(accessTokenCacheKey, resAccessToken.AccessToken, time.Duration(expires)*time.Second)
	accessToken = resAccessToken.AccessToken
	return
}
func sendMsg(token string, msg []byte) (status bool) {
	log.Printf("需要POST的内容：%s\r\n", msg)
	body := bytes.NewBuffer(msg)
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)
	res, err := http.Post(url, "application/json;charset=utf-8", body)
	if err != nil {
		log.Fatal(err)
		return
	}
	result, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Printf("微信接口返回消息：%s\r\n", result)

	return
}

func main() {
	wm := &WechatMsg{
		Touser:  msgInfo.Touser,
		Toparty: msgInfo.Toparty,
		Msgtype: "textcard",
		Agentid: msgInfo.Agentid,
		TextcardMsg: &TextcardMsg{
			Title:       msgInfo.Title,
			Description: msgInfo.Msg,
			Url:         msgInfo.Url,
			Btntxt:      "更多",
		},
	}
	msg, err := json.Marshal(wm)
	if err != nil {
		log.Println(err)
		return
	}
	token := getToken(msgInfo.Corpid, msgInfo.Corpsecret, msgInfo.Agentid)
	sendMsg(token, msg)
}
