package newebpay

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Loopmaas/xtime"
)

type CreditCardClose struct {
	MerchantID_ string `json:"MerchantID_"` // 商店代號
	PostData_   string `json:"PostData_"`   // 加密資料
}

type CreditCardClosePostData struct {
	RespondType     string  `json:"RespondType"`       // 回傳格式 JSON
	Version         string  `json:"Version"`           // 串接程式版本 1.0
	Amt             int     `json:"Amt"`               // 請退款金額
	MerchantOrderNo string  `json:"MerchantOrderNo"`   // 商店訂單編號
	TimeStamp       string  `json:"TimeStamp"`         // 時間戳記 UTC Unix timestamp
	IndexType       int     `json:"IndexType"`         // 單號類別 1=使用商店訂單編號 2=使用藍新金流交易單號, 1
	TradeNo         *string `json:"TradeNo,omitempty"` // 藍新金流交易序號, 商店訂單編號二擇一填入
	CloseType       int     `json:"CloseType"`         // 1=請款 2=退款
	Cancel          int     `json:"Cancel"`            // 取消請款或退款
}

// 信用卡請款 B031: CloseType=1, Cancel=0
// 信用卡退款 B032: CloseType=2, Cancel=0
// 信用卡取消請款 B033: CloseType=1, Cancel=1
// 信用卡取消退款 B034: CloseType=2, Cancel=1 *

func (a Api) CreditCardPaymentRequest(m *Merchant, merchantOrderNo string, amount int, requestedAt xtime.Time) (*RespCreditCardBehavior, error) {
	return a.creditCardClose(m, "B031", merchantOrderNo, amount, requestedAt)
}

func (a Api) CreditCardCancelPaymentRequest(m *Merchant, merchantOrderNo string, amount int, requestedAt xtime.Time) (*RespCreditCardBehavior, error) {
	return a.creditCardClose(m, "B033", merchantOrderNo, amount, requestedAt)
}

func (a Api) CreditCardRefundRequest(m *Merchant, merchantOrderNo string, amount int, requestedAt xtime.Time) (*RespCreditCardBehavior, error) {
	return a.creditCardClose(m, "B032", merchantOrderNo, amount, requestedAt)
}

func (a Api) creditCardClose(m *Merchant, requestType, merchantOrderNo string, amount int, requestedAt xtime.Time) (*RespCreditCardBehavior, error) {
	var (
		closeType int
		cancel    int
	)

	switch requestType {
	case "B031":
		closeType = 1
		cancel = 0
	case "B032":
		closeType = 2
		cancel = 0
	case "B033":
		closeType = 1
		cancel = 1
	case "B034":
		closeType = 2
		cancel = 1
	default:
		return nil, fmt.Errorf("Invalid request type: %s", requestType)
	}

	data := CreditCardClosePostData{
		RespondType:     "JSON",
		Version:         "1.0",
		Amt:             amount,
		MerchantOrderNo: merchantOrderNo,
		TimeStamp:       strconv.FormatInt(time.Time(requestedAt).Unix(), 10),
		IndexType:       1,
		TradeNo:         nil,
		CloseType:       closeType,
		Cancel:          cancel,
	}

	encData, err := encryptData(data, m.HashKey, m.HashIv)
	if err != nil {
		return nil, err
	}

	formData := url.Values{
		"MerchantID_": {m.MerchantId},
		"PostData_":   {encData},
	}

	resp, err := http.PostForm(a.ApiUrlCreditCardClose, formData)
	if err != nil {
		return nil, fmt.Errorf("Failed to submit form: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	receivedData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read received data failed: %v", err)
	}

	var tp RespPayload
	if err := json.Unmarshal(receivedData, &tp); err != nil {
		return nil, fmt.Errorf("[close] failed to decode response: %v, received data: %s", err, string(receivedData))
	}

	if !tp.IsSuccess() {
		return nil, fmt.Errorf("[close] %s: %s", tp.Status, tp.Message)
	}

	payload := RespCreditCardBehavior{
		Status:  tp.Status,
		Message: tp.Message,
	}
	if err := tp.Assert(&payload.Result); err != nil {
		return nil, fmt.Errorf("[close] assert: %v", err)
	}

	return &payload, nil
}
