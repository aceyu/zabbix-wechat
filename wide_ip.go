package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type RegionMembers struct {
	Name string `json:"name"`
}

type Region struct {
	Kind          string           `json:"kind"`
	Name          string           `json:"name"`
	Partition     string           `json:"partition"`
	FullPath      string           `json:"fullPath"`
	RegionMembers []*RegionMembers `json:"regionMembers"`
}

var DCA = "DC_shuitu"
var DCB = "DC_xinpaifang"
var client = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

type F5Data struct {
	Domain string
	DCAIP  string
	DCBIP  string
}

var f5url = flag.String("url", "https://10.1.22.100", "")
var username = flag.String("u", "admin", "")
var password = flag.String("p", "@gL$hcT4m~", "")

func main() {

	var f5data []*F5Data
	f5data = append(f5data, &F5Data{"xxxholic.i.fbank.com", "114.114.100.114", "114.114.200.114"})

	for _, data := range f5data {
		// 1.创建DCA server
		s1 := strings.Replace(server, "{ip}", data.DCAIP, -1)
		s1 = strings.Replace(s1, "{datacenter}", DCA, -1)
		s1 = strings.Replace(s1, "{DC}", "DCA", -1)
		s1 = strings.Replace(s1, "{domain}", data.Domain, -1)

		err := postData(fmt.Sprintf("%s/mgmt/tm/gtm/server/", *f5url), s1)
		if err != nil {
			fmt.Printf("DCA Server创建失败，数据为：%v", data)
			fmt.Println("")
		}

		// 2.创建DCB server
		s2 := strings.Replace(server, "{ip}", data.DCBIP, -1)
		s2 = strings.Replace(s2, "{datacenter}", DCB, -1)
		s2 = strings.Replace(s2, "{DC}", "DCB", -1)
		s2 = strings.Replace(s2, "{domain}", data.Domain, -1)
		err = postData(fmt.Sprintf("%s/mgmt/tm/gtm/server/", *f5url), s2)
		if err != nil {
			fmt.Printf("DCB Server创建失败，数据为：%v", data)
			fmt.Println("")
		}

		// 3.创建DCA pool
		p1 := strings.Replace(pool1, "{ip1}", data.DCAIP, -1)
		p1 = strings.Replace(p1, "{ip2}", data.DCBIP, -1)
		p1 = strings.Replace(p1, "{domain}", data.Domain, -1)
		err = postData(fmt.Sprintf("%s/mgmt/tm/gtm/pool/a/", *f5url), p1)
		if err != nil {
			fmt.Printf("DCA Pool创建失败，数据为：%v", data)
			fmt.Println("")
		}
		// 4.创建DCB pool
		p2 := strings.Replace(pool2, "{ip1}", data.DCAIP, -1)
		p2 = strings.Replace(p2, "{ip2}", data.DCBIP, -1)
		p2 = strings.Replace(p2, "{domain}", data.Domain, -1)
		err = postData(fmt.Sprintf("%s/mgmt/tm/gtm/pool/a/", *f5url), p2)
		if err != nil {
			fmt.Printf("DCB Pool创建失败，数据为：%v", data)
			fmt.Println("")
		}
		// 5.创建WideIp
		w := strings.Replace(wideIp, "{domain}", data.Domain, -1)
		err = postData(fmt.Sprintf("%s/mgmt/tm/gtm/wideip/a/", *f5url), w)
		if err != nil {
			fmt.Printf("Wide IP创建失败，数据为：%v", data)
			fmt.Println("")
		}
		// 6.加入DCA Region
		err = addRegion("DCA", data.Domain)
		if err != nil {
			fmt.Printf("DCA Region创建失败，数据为：%v", data)
			fmt.Println("")
		}
		// 7.加入DCB Region
		err = addRegion("DCB", data.Domain)
		if err != nil {
			fmt.Printf("DCB Region创建失败，数据为：%v", data)
			fmt.Println("")
		}
		fmt.Printf("记录：%v操作成功！", data)
		fmt.Println("")
	}
}

func postData(url, data string) error {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(data))
	if err != nil {
		return err
	}
	req.SetBasicAuth(*username, *password)
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("返回错误")
	}
	return nil
}

func addRegion(dc, domain string) error {
	url := fmt.Sprintf("%s/mgmt/tm/gtm/region/~Common~%s_pool", *f5url, dc)
	req, err := http.NewRequest(http.MethodGet, url, strings.NewReader(""))
	if err != nil {
		return err
	}
	req.SetBasicAuth(*username, *password)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("返回错误")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	regin := &Region{}
	err = json.Unmarshal(body, regin)
	if err != nil {
		return err
	}
	regin.RegionMembers = append(regin.RegionMembers, &RegionMembers{fmt.Sprintf("pool /Common/%s_%s_pool", dc, domain)})
	reginstr, err := json.Marshal(regin)
	if err != nil {
		return err
	}
	req, err = http.NewRequest(http.MethodPut, url, strings.NewReader(string(reginstr)))
	if err != nil {
		return err
	}
	req.SetBasicAuth(*username, *password)
	req.Header.Add("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("返回错误")
	}
	return nil
}

var wideIp = `{
    "kind": "tm:gtm:wideip:a:astate",
    "name": "{domain}",
    "partition": "Common",
    "generation": 1,
    "enabled": true,
    "failureRcode": "noerror",
    "failureRcodeResponse": "disabled",
    "failureRcodeTtl": 0,
    "lastResortPool": "",
    "minimalResponse": "enabled",
    "persistCidrIpv4": 32,
    "persistCidrIpv6": 128,
    "persistence": "disabled",
    "poolLbMode": "topology",
    "ttlPersistence": 3600,
    "pools": [
        {
            "name": "DCA_{domain}_pool",
            "partition": "Common",
            "order": 1,
            "ratio": 1
        },
        {
            "name": "DCB_{domain}_pool",
            "partition": "Common",
            "order": 0,
            "ratio": 1
        }
    ]
}`

var pool1 = `{
    "kind": "tm:gtm:pool:a:astate",
    "name": "DCA_{domain}_pool",
    "partition": "Common",
    "alternateMode": "round-robin",
    "dynamicRatio": "disabled",
    "enabled": true,
    "fallbackIp": "any",
    "fallbackMode": "return-to-dns",
    "limitMaxBps": 0,
    "limitMaxBpsStatus": "disabled",
    "limitMaxConnections": 0,
    "limitMaxConnectionsStatus": "disabled",
    "limitMaxPps": 0,
    "limitMaxPpsStatus": "disabled",
    "loadBalancingMode": "global-availability",
    "manualResume": "disabled",
    "maxAnswersReturned": 1,
    "monitor": "default",
    "qosHitRatio": 5,
    "qosHops": 0,
    "qosKilobytesSecond": 3,
    "qosLcs": 30,
    "qosPacketRate": 1,
    "qosRtt": 50,
    "qosTopology": 0,
    "qosVsCapacity": 0,
    "qosVsScore": 0,
    "ttl": 30,
    "verifyMemberAvailability": "disabled",
    "members": [
        {
            "kind": "tm:gtm:pool:a:members:membersstate",
            "name": "{ip1}:DCA_{domain}",
            "partition": "Common",
            "generation": 1,
            "enabled": true,
            "limitMaxBps": 0,
            "limitMaxBpsStatus": "disabled",
            "limitMaxConnections": 0,
            "limitMaxConnectionsStatus": "disabled",
            "limitMaxPps": 0,
            "limitMaxPpsStatus": "disabled",
            "memberOrder": 0,
            "monitor": "default",
            "ratio": 1
        },
        {
            "kind": "tm:gtm:pool:a:members:membersstate",
            "name": "{ip2}:DCB_{domain}",
            "partition": "Common",
            "generation": 1,
            "enabled": true,
            "limitMaxBps": 0,
            "limitMaxBpsStatus": "disabled",
            "limitMaxConnections": 0,
            "limitMaxConnectionsStatus": "disabled",
            "limitMaxPps": 0,
            "limitMaxPpsStatus": "disabled",
            "memberOrder": 1,
            "monitor": "default",
            "ratio": 1
        }
    ]
}`

var pool2 = `{
    "kind": "tm:gtm:pool:a:astate",
    "name": "DCB_{domain}_pool",
    "partition": "Common",
    "alternateMode": "round-robin",
    "dynamicRatio": "disabled",
    "enabled": true,
    "fallbackIp": "any",
    "fallbackMode": "return-to-dns",
    "limitMaxBps": 0,
    "limitMaxBpsStatus": "disabled",
    "limitMaxConnections": 0,
    "limitMaxConnectionsStatus": "disabled",
    "limitMaxPps": 0,
    "limitMaxPpsStatus": "disabled",
    "loadBalancingMode": "global-availability",
    "manualResume": "disabled",
    "maxAnswersReturned": 1,
    "monitor": "default",
    "qosHitRatio": 5,
    "qosHops": 0,
    "qosKilobytesSecond": 3,
    "qosLcs": 30,
    "qosPacketRate": 1,
    "qosRtt": 50,
    "qosTopology": 0,
    "qosVsCapacity": 0,
    "qosVsScore": 0,
    "ttl": 30,
    "verifyMemberAvailability": "disabled",
    "members": [
        {
            "kind": "tm:gtm:pool:a:members:membersstate",
            "name": "{ip2}:DCB_{domain}",
            "partition": "Common",
            "generation": 1,
            "enabled": true,
            "limitMaxBps": 0,
            "limitMaxBpsStatus": "disabled",
            "limitMaxConnections": 0,
            "limitMaxConnectionsStatus": "disabled",
            "limitMaxPps": 0,
            "limitMaxPpsStatus": "disabled",
            "memberOrder": 0,
            "monitor": "default",
            "ratio": 1
        },
        {
            "kind": "tm:gtm:pool:a:members:membersstate",
            "name": "{ip1}:DCA_{domain}",
            "partition": "Common",
            "generation": 1,
            "enabled": true,
            "limitMaxBps": 0,
            "limitMaxBpsStatus": "disabled",
            "limitMaxConnections": 0,
            "limitMaxConnectionsStatus": "disabled",
            "limitMaxPps": 0,
            "limitMaxPpsStatus": "disabled",
            "memberOrder": 1,
            "monitor": "default",
            "ratio": 1
        }
    ]
}`

var server = `{
    "kind": "tm:gtm:server:serverstate",
    "name": "{ip}",
    "partition": "Common",
    "fullPath": "/Common/{ip}",
    "datacenter": "/Common/{datacenter}", 
    "enabled": true,
    "exposeRouteDomains": "no",
    "iqAllowPath": "yes",
    "iqAllowServiceCheck": "yes",
    "iqAllowSnmp": "yes",
    "limitCpuUsage": 0,
    "limitCpuUsageStatus": "disabled",
    "limitMaxBps": 0,
    "limitMaxBpsStatus": "disabled",
    "limitMaxConnections": 0,
    "limitMaxConnectionsStatus": "disabled",
    "limitMaxPps": 0,
    "limitMaxPpsStatus": "disabled",
    "limitMemAvail": 0,
    "limitMemAvailStatus": "disabled",
    "linkDiscovery": "disabled",
    "proberFallback": "inherit",
    "proberPreference": "inherit",
    "product": "generic-host",
    "virtualServerDiscovery": "disabled",
    "addresses": [
        {
            "name": "{ip}",
            "deviceName": "{ip}",
            "translation": "none"
        }
    ],
    "virtual-servers": [
        {
            "kind": "tm:gtm:server:virtual-servers:virtual-serversstate",
            "name": "{DC}_{domain}",
            "generation": 1,
            "destination": "{ip}:80",
            "enabled": true,
            "limitMaxBps": 0,
            "limitMaxBpsStatus": "disabled",
            "limitMaxConnections": 0,
            "limitMaxConnectionsStatus": "disabled",
            "limitMaxPps": 0,
            "limitMaxPpsStatus": "disabled",
            "translationAddress": "none",
            "translationPort": 0
        }
    ]
}`
