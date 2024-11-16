package newebpay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/hexcraft-biz/xtime"
)

type CreditCardCancel struct {
	MerchantID_ string `json:"MerchantID_"` // 商店代號
	PostData_   string `json:"PostData_"`   // 加密資料
}

type CreditCardCancelPostData struct {
	RespondType     string  `json:"RespondType"`     // 回傳格式 JSON
	Version         string  `json:"Version"`         // 串接程式版本 1.0
	Amt             int     `json:"Amt"`             // 取消授權金額, 需與授權金額相同
	MerchantOrderNo string  `json:"MerchantOrderNo"` // 商店訂單編號
	TradeNo         *string `json:"TradeNo"`         // 藍新金流交易序號, 商店訂單編號二擇一填入
	IndexType       int     `json:"IndexType"`       // 單號類別 1=使用商店訂單編號 2=使用藍新金流交易單號
	TimeStamp       string  `json:"TimeStamp"`       // 時間戳記 UTC Unix timestamp
}

func (a Api) CreditCardCancelTransactionAuthorization(merchant *Merchant, merchantOrderNo string, amount int, requestedAt xtime.Time) (*RespCreditCardCancel, error) {
	data := CreditCardCancelPostData{
		RespondType:     "JSON",
		Version:         "1.0",
		Amt:             amount,
		MerchantOrderNo: merchantOrderNo,
		TradeNo:         nil,
		IndexType:       1,
		TimeStamp:       strconv.FormatInt(time.Time(requestedAt).Unix(), 10),
	}

	encData, err := encryptData(data, merchant.HashKey, merchant.HashIv)
	if err != nil {
		return nil, err
	}

	formData := url.Values{
		"MerchantID_": {merchant.MerchantId},
		"PostData_":   {encData},
	}

	resp, err := http.PostForm(a.ApiUrlCreditCardCancel, formData)
	if err != nil {
		return nil, fmt.Errorf("Failed to submit form: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var payload RespCreditCardCancel
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &payload, nil
}

type RespCreditCardCancel struct {
	Status  string                  `json:"Status"`
	Message string                  `json:"Message"`
	Result  *ResultCreditCardCancel `json:"Result"`
}

type ResultCreditCardCancel struct {
	MerchantID      string `json:"MerchantID"`
	TradeNo         string `json:"TradeNo"`
	Amt             int    `json:"Amt"`
	MerchantOrderNo string `json:"MerchantOrderNo"`
	CheckCode       string `json:"CheckCode"`
}

func (r RespCreditCardCancel) IsSuccess() bool {
	return r.Status == "SUCCESS"
}
