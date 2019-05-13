package main

import (
	"bytes"
	"crypto/md5"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"
)

func main() {
	m := UnifiedOrder{}
	m.MchId = "xxx"
	m.AppId = "xxx"
	m.AppKey = "xxx"
	m.NonceStr = fmt.Sprintf("%x", md5.Sum([]byte(time.Now().Format(time.RFC3339Nano))))
	m.Body = "购买商品"
	m.OutTradeNo = time.Now().Format("2006010215")
	m.TotalFee = 1
	m.SpBillCreateIp = "127.0.0.1"
	m.NotifyUrl = "http://www.test.com/wxPayCallback"
	m.TradeType = "NATIVE"
	r, err := UnifiedOrderPost(m)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%#v\n", r)
	fmt.Println("二维码链接:", r.CodeUrl)
}

type UnifiedOrder struct {
	MchId          string `xml:"mch_id"` // 商户号 通过微信支付商户资料审核后邮件发送
	AppId          string `xml:"appid"`  // 公众账号ID 通过微信支付商户资料审核后邮件发送
	AppKey         string // API密钥 https://pay.weixin.qq.com 帐户设置-安全设置-API安全-API密钥-设置API密钥
	NonceStr       string `xml:"nonce_str"`        // 随机字符串
	Body           string `xml:"body"`             // 商品描述
	OutTradeNo     string `xml:"out_trade_no"`     // 商户订单号
	TotalFee       int64  `xml:"total_fee"`        // 标价金额
	SpBillCreateIp string `xml:"spbill_create_ip"` // 终端IP
	NotifyUrl      string `xml:"notify_url"`       // 通知地址
	TradeType      string `xml:"trade_type"`       // 交易类型: JSAPI NATIVE APP
	Sign           string `xml:"sign"`             // 签名
}

type UnifiedOrderResponse struct {
	ReturnCode string `xml:"return_code"`
	ReturnMsg  string `xml:"return_msg"`
	AppId      string `xml:"appid"`
	MchId      string `xml:"mch_id"`
	NonceStr   string `xml:"nonce_str"`
	Sign       string `xml:"sign"`
	ResultCode string `xml:"result_code"`
	TradeType  string `xml:"trade_type"`
	PrepayId   string `xml:"prepay_id"`
	CodeUrl    string `xml:"code_url"` // 二维码链接
}

// https://pay.weixin.qq.com/wiki/doc/api/native.php?chapter=9_1
// https://cloud.tencent.com/developer/article/1074194
func UnifiedOrderPost(u UnifiedOrder) (r UnifiedOrderResponse, err error) {
	type st struct {
		Key   string
		Value interface{}
	}

	// 1. 提取需要加密的字段
	var query []st
	t := reflect.TypeOf(u)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if v := f.Tag.Get("xml"); v != "" && v != "sign" {
			query = append(query, st{v, reflect.ValueOf(u).Field(i).Interface()})
		}
	}

	// 2. 排序
	sort.Slice(query, func(i, j int) bool { return query[i].Key < query[j].Key })

	// 3. 拼接字符串
	s := ""
	for _, v := range query {
		val := fmt.Sprintf("%v", v.Value)
		if val != "" {
			s += v.Key + "=" + val + "&"
		}
	}
	s += "key=" + u.AppKey // 在键值对的最后加上 API 密钥
	// fmt.Println(s)

	// 4. 进行MD5签名并且将所有字符转为大写
	u.Sign = strings.ToUpper(fmt.Sprintf("%x", md5.Sum([]byte(s))))

	// 5. 生成 XML 数据
	u.AppKey = "" // 清空
	bXml, err := xml.Marshal(u)
	if err != nil {
		return r, err
	}
	sXml := strings.Replace(string(bXml), "UnifiedOrder", "xml", -1)
	sXml = strings.Replace(sXml, "<AppKey></AppKey>", "", -1)
	// fmt.Println(sXml)

	// 6. 发送 POST 请求
	req, err := http.NewRequest("POST", "https://api.mch.weixin.qq.com/pay/unifiedorder", bytes.NewReader([]byte(sXml)))
	if err != nil {
		return r, err
	}
	req.Header.Set("Accept", "application/xml")
	req.Header.Set("Content-Type", "application/xml;charset=utf-8")

	c := http.Client{}
	res, err := c.Do(req)
	if err != nil {
		return r, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return r, err
	}
	// fmt.Printf("%s\n", body)

	// 7. 结果信息
	err = xml.Unmarshal(body, &r)
	if err != nil {
		return r, err
	}

	return r, nil
}
