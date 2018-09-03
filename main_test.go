package main_test

import (
	"encoding/xml"
	"fmt"
	"log"
	"testing"
)

func TestParseXmlMsg(t *testing.T) {
	msg := `<?xml version="1.0" encoding="UTF-8" ?>
	<root>
	  <from>10.1.10.69</from>
	  <time>2018.09.03 09:31:47</time>
	  <level>Warning</level>
	  <name>Free disk space is less than 20% on volume /</name>
	  <key>vfs.fs.size[/,pfree]</key>
	  <value>20 %</value>
	  <now>20 %</now>
	  <id>3980794</id>
	  <ip>10.1.10.69</ip>
	  <url>http://10.0.1.11/zabbix/</url>
	  <age>0m</age>
	  <status>PROBLEM</status>
	<acknowledgement> No </acknowledgement>
	<acknowledgementhistory> </acknowledgementhistory>
	</root>`
	xmlMsg := parseXmlMsg(msg)
	fmt.Println(xmlMsg)
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
	Status                 string `xml:"status"`
	Acknowledgement        string `xml:"acknowledgement"`
	Acknowledgementhistory string `xml:"acknowledgementhistory"`
}

type TextcardMsg struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Url         string `json:"url"`
	Btntxt      string `json:"btntxt"`
}

func parseXmlMsg(msg string) *TextcardMsg {
	var xmlMsg XmlMsg
	err := xml.Unmarshal([]byte(msg), &xmlMsg)
	if err != nil {
		log.Println(err)
		return nil
	}
	description := "<div class=\"gray\">告警级别：%s</div>" +
		"<div class=\"gray\">故障时间：%s</div><div class=\"gray\">故障时长：%s</div>" +
		"<div class=\"gray\">IP地址：%s</div>" +
		"<div class=\"gray\">建成项：%s</div>" +
		"<div class=\"highlight\">%s</div>" +
		"<div class=\"gray\">[%s 故障 (%s)]</div>"

	description = fmt.Sprintf(description, xmlMsg.Level, xmlMsg.Time, xmlMsg.Age, xmlMsg.Ip, xmlMsg.Key, xmlMsg.Now, xmlMsg.Ip, xmlMsg.Id)
	return &TextcardMsg{
		Title:       xmlMsg.Name,
		Url:         xmlMsg.Url,
		Btntxt:      "更多",
		Description: description,
	}
}
