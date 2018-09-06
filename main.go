package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
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
	Touser, Toparty, Totag, Corpid, Corpsecret, Msg, Log, CachePath string
	Agentid                                                         int
}

var msgInfo MsgInfo
var filecache cache.Cache

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
	flag.StringVar(&msgInfo.Totag, "totag", "", "消息的接收组，可以在微信后台查看，可空。")
	flag.IntVar(&msgInfo.Agentid, "agentid", 1, "AgentID，可以在微信后台查看，不可空。")
	flag.StringVar(&msgInfo.Corpid, "corpid", "", "CorpID，可以在微信后台查看，不可空。")
	flag.StringVar(&msgInfo.Corpsecret, "corpsecret", "", "CorpSecret，可以在微信后台查看，不可空。")
	flag.StringVar(&msgInfo.Msg, "msg", "", "消息体, 不可空。")
	flag.StringVar(&msgInfo.Log, "log", "/tmp/wechat.log", "日志路径，可空。")
	flag.StringVar(&msgInfo.CachePath, "cachepath", "/tmp/cache", "缓存路径，可空。")
	flag.Parse()
	filecache, _ = cache.NewCache("file", `{"CachePath":"`+msgInfo.CachePath+`","FileSuffix":".bin","DirectoryLevel":2,"EmbedExpiry":0}`)
	logFile, logErr := os.OpenFile(msgInfo.Log, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if logErr != nil {
		fmt.Println("Fail to find ", msgInfo.Log, " Start Failed")
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
		return false
	}
	result, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return false
	}
	log.Printf("微信接口返回消息：%s\r\n", result)
	err = checkRespError(result)
	if err == errUnmarshall {
		return true
	}
	if err != nil {
		return false
	}
	return true
}

type wechatError struct {
	ErrCode int64  `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

var errUnmarshall = errors.New("Json Unmarshal Error")

func checkRespError(jsonData []byte) error {
	var errmsg wechatError
	if err := json.Unmarshal(jsonData, &errmsg); err != nil {
		return errUnmarshall
	}
	if errmsg.ErrCode != 0 {
		return fmt.Errorf("Error , errcode=%d , errmsg=%s", errmsg.ErrCode, errmsg.ErrMsg)
	}

	return nil
}

func main() {
	log.Printf("来源消息为：%s", msgInfo.Msg)
	textcardMsg := parseXmlMsg()
	wm := &WechatMsg{
		Touser:      msgInfo.Touser,
		Toparty:     msgInfo.Toparty,
		Totag:       msgInfo.Totag,
		Msgtype:     "textcard",
		Agentid:     msgInfo.Agentid,
		TextcardMsg: textcardMsg,
	}
	msg, err := json.Marshal(wm)
	if err != nil {
		log.Println(err)
		return
	}
	retry := 1
Do:
	token := getToken(msgInfo.Corpid, msgInfo.Corpsecret, msgInfo.Agentid)
	sendMsg(token, msg)
	if err != nil {
		if retry > 0 {
			retry--
			accessTokenCacheKey := fmt.Sprintf("access_token_%d", msgInfo.Agentid)
			filecache.Delete(accessTokenCacheKey)
			goto Do
		}
		return
	}
}

type XmlMsg struct {
	From                   string `xml:"from"`
	Time                   string `xml:"time"`
	Level                  string `xml:"level"`
	Name                   string `xml:"name"`
	Key                    string `xml:"key"`
	Value                  string `xml:"value"`
	Now                    string `xml:"now"`
	Id                     string `xml:"id"`
	Ip                     string `xml:"ip"`
	Url                    string `xml:"url"`
	Age                    string `xml:"age"`
	RecoveryTime           string `xml:"recoveryTime"`
	Status                 string `xml:"status"`
	Acknowledgement        string `xml:"acknowledgement"`
	Acknowledgementhistory string `xml:"acknowledgementhistory"`
}

func parseXmlMsg() *TextcardMsg {
	var xmlMsg XmlMsg
	err := xml.Unmarshal([]byte(msgInfo.Msg), &xmlMsg)
	if err != nil {
		log.Println(err)
		return nil
	}
	var description string
	if xmlMsg.RecoveryTime == "" || xmlMsg.Status == "PROBLEM" {
		description = "<div class=\"gray\">告警级别：%s</div>" +
			"<div class=\"gray\">故障时间：%s</div><div class=\"gray\">故障时长：%s</div>" +
			"<div class=\"gray\">IP地址：%s</div>" +
			"<div class=\"gray\">检测项：%s</div>" +
			"<div class=\"highlight\">%s</div>" +
			"<div class=\"gray\">[%s 故障 (%s)]</div>"
		description = fmt.Sprintf(description, xmlMsg.Level, xmlMsg.Time, xmlMsg.Age, xmlMsg.Ip, xmlMsg.Key, xmlMsg.Now, xmlMsg.From, xmlMsg.Id)

	} else if xmlMsg.RecoveryTime != "" || xmlMsg.Status == "OK" {
		description = "<div class=\"gray\">告警级别：%s</div>" +
			"<div class=\"gray\">故障时间：%s</div>" +
			"<div class=\"gray\">恢复时间：%s</div><div class=\"gray\">故障时长：%s</div>" +
			"<div class=\"gray\">IP地址：%s</div>" +
			"<div class=\"gray\">检测项：%s</div>" +
			"<div class=\"highlight\">%s</div>" +
			"<div class=\"gray\">[%s 恢复 (%s)]</div>"
		description = fmt.Sprintf(description, xmlMsg.Level, xmlMsg.Time, xmlMsg.RecoveryTime, xmlMsg.Age, xmlMsg.Ip, xmlMsg.Key, xmlMsg.Now, xmlMsg.From, xmlMsg.Id)

	}
	return &TextcardMsg{
		Title:       xmlMsg.Name,
		Url:         xmlMsg.Url,
		Btntxt:      "更多",
		Description: description,
	}
}
