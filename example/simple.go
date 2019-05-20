package main

import (
	".."
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

func main() {
	ac := wxPay.AppConf{
		MchId:  "xxxx",
		AppId:  "xxxx",
		AppKey: "xxxx",
	}
	uo := wxPay.UnifiedOrder{
		NonceStr:       fmt.Sprintf("%x", md5.Sum([]byte(time.Now().Format(time.RFC3339Nano)))),
		Body:           "商品名称",
		OutTradeNo:     strings.Replace(time.Now().Format("D20060102150405.000000"), ".", "_", -1),
		TotalFee:       1,
		SpBillCreateIp: "127.0.0.1",
		NotifyUrl:      "www.abc.com/wxPayCallback",
		TradeType:      "NATIVE",
	}
	str, err := ac.UnifiedOrderSend(uo)
	if err != nil {
		log.Fatalln(err.Error())
		return
	}
	fmt.Println(str) // 请求结果

	// 检测结果是否正确
	ur, err := wxPay.UnifiedOrderCheck(str)
	if err != nil {
		log.Fatalln(err.Error())
		return
	}
	fmt.Printf("%#v\n", ur)
}

// 微信支付回调验证和结果处理
func wxPayCallback(r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalln(err.Error())
		return
	}
	str := string(body)
	fmt.Println(str) // 通知结果

	// 检测结果是否正确
	ac := wxPay.AppConf{AppKey: "xxxx"}
	un, err := ac.UnifiedOrderNotifyCheck(str)
	// ztj.WriteLog(logFile, toJson(un)) // 记录所有结果通知
	if err != nil {
		log.Fatalln(err.Error())
		return
	}
	fmt.Printf("%#v\n", un)
}
