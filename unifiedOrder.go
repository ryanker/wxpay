package wxPay

import (
	"bytes"
	"crypto/md5"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"strings"
)

type AppConf struct {
	AppId  string // 公众账号ID 通过微信支付商户资料审核后邮件发送
	AppKey string // API密钥 https://pay.weixin.qq.com 帐户设置-安全设置-API安全-API密钥-设置API密钥
	MchId  string // 商户号 通过微信支付商户资料审核后邮件发送
}

type UnifiedOrder struct {
	AppId          string `xml:"appid"`            // 公众账号ID
	MchId          string `xml:"mch_id"`           // 商户号
	NonceStr       string `xml:"nonce_str"`        // 随机字符串
	Body           string `xml:"body"`             // 商品描述
	OutTradeNo     string `xml:"out_trade_no"`     // 商户订单号
	TotalFee       int64  `xml:"total_fee"`        // 标价金额
	SpBillCreateIp string `xml:"spbill_create_ip"` // 终端IP
	NotifyUrl      string `xml:"notify_url"`       // 通知地址
	TradeType      string `xml:"trade_type"`       // 交易类型: JSAPI NATIVE APP
	Sign           string `xml:"sign"`             // 签名
}

// https://pay.weixin.qq.com/wiki/doc/api/native.php?chapter=9_1
// https://cloud.tencent.com/developer/article/1074194
// 统一下单
func (ac *AppConf) UnifiedOrderSend(u UnifiedOrder) (string, error) {
	u.AppId = ac.AppId
	u.MchId = ac.MchId

	// 生成签名
	sign, err := getSign(u, ac.AppKey)
	if err != nil {
		return "", err
	}

	// 生成 XML 数据
	u.Sign = sign
	bXml, err := xml.Marshal(u)
	if err != nil {
		return "", err
	}
	sXml := strings.Replace(string(bXml), "UnifiedOrder", "xml", -1)
	// fmt.Println(sXml)

	// 发送 POST 请求
	req, err := http.NewRequest("POST", "https://api.mch.weixin.qq.com/pay/unifiedorder", bytes.NewReader([]byte(sXml)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/xml")
	req.Header.Set("Content-Type", "application/xml;charset=utf-8")

	c := http.Client{}
	res, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	// fmt.Printf("%s\n", body)

	return string(body), nil
}

type UnifiedOrderResponse struct {
	ReturnCode string `xml:"return_code"` // 返回状态码
	ReturnMsg  string `xml:"return_msg"`  // 返回状态码
	AppId      string `xml:"appid"`
	MchId      string `xml:"mch_id"`
	DeviceInfo string `xml:"device_info"`
	NonceStr   string `xml:"nonce_str"`
	Sign       string `xml:"sign"`
	ResultCode string `xml:"result_code"`
	ErrCode    string `xml:"err_code"`
	ErrCodeDes string `xml:"err_code_des"`
	TradeType  string `xml:"trade_type"`
	PrepayId   string `xml:"prepay_id"`
	CodeUrl    string `xml:"code_url"` // 二维码链接
}

// 结果数据转结构体
func UnifiedOrderCheck(s string) (ur UnifiedOrderResponse, err error) {
	err = xml.Unmarshal([]byte(s), &ur)
	if err != nil {
		return ur, err
	}
	if ur.ReturnCode != "SUCCESS" || ur.ResultCode != "SUCCESS" {
		return ur, fmt.Errorf("return is error, ReturnCode: %s, ResultCode: %s", ur.ReturnCode, ur.ResultCode)
	}
	return
}

type UnifiedOrderNotify struct {
	ReturnCode         string `xml:"return_code"`          // 返回状态码
	ReturnMsg          string `xml:"return_msg"`           // 返回信息
	AppId              string `xml:"appid"`                // 公众账号ID
	MchId              string `xml:"mch_id"`               // 商户号
	DeviceInfo         string `xml:"device_info"`          // 设备号
	NonceStr           string `xml:"nonce_str"`            // 随机字符串
	Sign               string `xml:"sign"`                 // 签名
	SignType           string `xml:"sign_type"`            // 签名类型
	ResultCode         string `xml:"result_code"`          // 业务结果
	ErrCode            string `xml:"err_code"`             // 错误代码
	ErrCodeDes         string `xml:"err_code_des"`         // 错误代码描述
	OpenId             string `xml:"openid"`               // 微信用户标识 (用户在商户 appid 下的唯一标识)
	IsSubscribe        string `xml:"is_subscribe"`         // 是否关注公众账号
	TradeType          string `xml:"trade_type"`           // 交易类型
	BankType           string `xml:"bank_type"`            // 付款银行
	TotalFee           int64  `xml:"total_fee"`            // 订单金额
	SettlementTotalFee int64  `xml:"settlement_total_fee"` // 应结订单金额
	FeeType            string `xml:"fee_type"`             // 货币种类
	CashFee            int64  `xml:"cash_fee"`             // 现金支付金额
	CashFeeType        string `xml:"cash_fee_type"`        // 现金支付货币类型
	TransactionId      string `xml:"transaction_id"`       // 微信支付订单号
	OutTradeNo         string `xml:"out_trade_no"`         // 商户订单号
	Attach             string `xml:"attach"`               // 商家数据包
	TimeEnd            string `xml:"time_end"`             // 支付完成时间
}

// https://pay.weixin.qq.com/wiki/doc/api/native.php?chapter=9_7&index=8
// 微信支付结果通知验证
func (ac *AppConf) UnifiedOrderNotifyCheck(s string) (un UnifiedOrderNotify, err error) {
	err = xml.Unmarshal([]byte(s), &un)
	if err != nil {
		return un, err
	}
	if un.ReturnCode != "SUCCESS" || un.ResultCode != "SUCCESS" {
		return un, fmt.Errorf("return is error, ReturnCode: %s, ResultCode: %s", un.ReturnCode, un.ResultCode)
	}

	// 生成签名
	sign, err := getSign(un, ac.AppKey)
	if err != nil {
		return un, err
	}

	// 验证签名是否一致
	if un.Sign != sign {
		return un, fmt.Errorf("sign is error, un.Sign:%s, sign:%s", un.Sign, sign)
	}

	return un, nil
}

// 微信支付完成返回参数
func UnifiedOrderNotifySuccess() string {
	return `<xml><return_code><![CDATA[SUCCESS]]></return_code><return_msg><![CDATA[OK]]></return_msg></xml>`
}

type Field struct {
	Key   string
	Value interface{}
}

// 生成签名
func getSign(i interface{}, key string) (string, error) {
	st := reflect.TypeOf(i)
	sv := reflect.ValueOf(i)
	if sv.Kind() == reflect.Struct {
	} else if sv.Kind() == reflect.Ptr && sv.Elem().Kind() == reflect.Struct {
		st = st.Elem()
		sv = sv.Elem()
	} else {
		return "", errors.New("getSign() value is error")
	}

	// 1. 提取需要加密的字段
	var query []Field
	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		if key := f.Tag.Get("xml"); key != "" && key != "sign" {
			query = append(query, Field{key, sv.Field(i).Interface()})
		}
	}

	// 2. 排序
	sort.Slice(query, func(i, j int) bool { return query[i].Key < query[j].Key })

	// 3. 拼接字符串
	s := ""
	for _, v := range query {
		val := fmt.Sprintf("%v", v.Value)
		if val != "" && val != "0" {
			s += v.Key + "=" + val + "&"
		}
	}
	s += "key=" + key // 在键值对的最后加上 API 密钥
	// fmt.Println(s)

	// 4. 进行MD5签名并且将所有字符转为大写
	sign := strings.ToUpper(fmt.Sprintf("%x", md5.Sum([]byte(s))))
	return sign, nil
}
