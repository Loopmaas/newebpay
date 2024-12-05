package newebpay

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hexcraft-biz/xtime"
)

type QueryPostData struct {
	MerchantID      string `json:"MerchantID"`
	Version         string `json:"Version"`     // 1.3
	RespondType     string `json:"RespondType"` // JSON
	CheckValue      string `json:"CheckValue"`  // IV
	TimeStamp       string `json:"TimeStamp"`
	MerchantOrderNo string `json:"MerchantOrderNo"`
	Amt             int    `json:"Amt"`
	Gateway         string `json:"Gateway"` // Composite
}

func (a Api) QueryTradeInfo(m *Merchant, merchantOrderNo string, amount int, requestedAt xtime.Time) (*RespQueryTradeInfo, error) {
	// generate check value
	checkValueData := fmt.Sprintf("IV=%s&Amt=%d&MerchantID=%s&MerchantOrderNo=%s&Key=%s", m.HashIv, amount, m.MerchantId, merchantOrderNo, m.HashKey)
	hash := sha256.Sum256([]byte(checkValueData))
	checkValue := strings.ToUpper(hex.EncodeToString(hash[:]))
	unixTimestamp := time.Time(requestedAt).UTC().Unix()

	formData := url.Values{
		"MerchantID":      {m.MerchantId},
		"Version":         {"1.3"},
		"RespondType":     {"JSON"},
		"CheckValue":      {checkValue},
		"TimeStamp":       {strconv.FormatInt(unixTimestamp, 10)},
		"MerchantOrderNo": {merchantOrderNo},
		"Amt":             {strconv.FormatInt(int64(amount), 10)},
	}

	resp, err := http.PostForm(a.ApiUrlQueryTradeInfo, formData)
	if err != nil {
		return nil, fmt.Errorf("Failed to submit form: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var payload RespQueryTradeInfo
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &payload, nil
}

type RespQueryTradeInfo struct {
	Status  string                `json:"Status"`
	Message string                `json:"Message"`
	Result  *ResultQueryTradeInfo `json:"Result,omitempty"`
}

func (r RespQueryTradeInfo) Retain(a *Api, m *Merchant, amount int, requestedAt xtime.Time) (*RespCreditCardBehavior, error) {
	if r.Status != "SUCCESS" {
		return nil, fmt.Errorf("query failed [%s]: %s", r.Status, r.Message)
	}

	result := r.Result
	if amount == result.Amt {
		return nil, nil
	}

	refundAmount := result.Amt - amount
	switch result.CloseStatus {
	case "0": // 未請款
		if amount > 0 {
			return a.CreditCardPaymentRequest(m, result.MerchantOrderNo, amount, requestedAt)
		} else {
			return a.CreditCardCancelTransactionAuthorization(m, result.MerchantOrderNo, result.Amt, requestedAt)
		}
	case "1": // 請款申請中
		resp, err := a.CreditCardCancelPaymentRequest(m, result.MerchantOrderNo, result.Amt, requestedAt)
		if err != nil {
			return nil, err
		} else if resp.Status != "SUCCESS" {
			return resp, nil
		}

		if amount > 0 {
			return a.CreditCardPaymentRequest(m, result.MerchantOrderNo, amount, requestedAt)
		} else {
			return a.CreditCardCancelTransactionAuthorization(m, result.MerchantOrderNo, result.Amt, requestedAt)
		}
	case "2", "3": // 請款處理中, 請款完成
		return a.CreditCardRefundRequest(m, result.MerchantOrderNo, refundAmount, requestedAt)
	}

	return nil, fmt.Errorf("invalid CloseStatus: %s", result.CloseStatus)
}

type ResultQueryTradeInfo struct {
	MerchantID      string `json:"MerchantID"`
	Amt             int    `json:"Amt"`
	TradeNo         string `json:"TradeNo"`
	MerchantOrderNo string `json:"MerchantOrderNo"`
	TradeStatus     string `json:"TradeStatus"`
	PaymentType     string `json:"PaymentType"`
	CreateTime      string `json:"CreateTime"`
	PayTime         string `json:"PayTime"`
	CheckCode       string `json:"CheckCode"`
	FundTime        string `json:"FundTime"`
	ShopMerchantID  string `json:"ShopMerchantID"`
	RespondCode     string `json:"RespondCode"`
	Auth            string `json:"Auth"`
	ECI             string `json:"ECI"`
	CloseAmt        string `json:"CloseAmt"`
	CloseStatus     string `json:"CloseStatus"`
	BackBalance     string `json:"BackBalance"`
	BackStatus      string `json:"BackStatus"`
	RespondMsg      string `json:"RespondMsg"`
	Inst            string `json:"Inst"`
	InstFirst       string `json:"InstFirst"`
	InstEach        string `json:"InstEach"`
	PaymentMethod   string `json:"PaymentMethod"`
	Card6No         string `json:"Card6No"`
	Card4No         string `json:"Card4No"`
	AuthBank        string `json:"AuthBank"`
}
