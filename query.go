package newebpay

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Loopmaas/xtime"
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

	receivedData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read received data failed: %v", err)
	}

	var tp RespPayload
	if err := json.Unmarshal(receivedData, &tp); err != nil {
		return nil, fmt.Errorf("[query] failed to decode response: %v, received data: %s", err, string(receivedData))
	}

	if !tp.IsSuccess() {
		return nil, fmt.Errorf("[query] %s: %s", tp.Status, tp.Message)
	}

	payload := RespQueryTradeInfo{
		Status:  tp.Status,
		Message: tp.Message,
	}
	if err := tp.Assert(&payload.Result); err != nil {
		return nil, fmt.Errorf("[query] assert: %v", err)
	}

	return &payload, nil
}

type RespQueryTradeInfo struct {
	Status  string               `json:"Status"`
	Message string               `json:"Message"`
	Result  ResultQueryTradeInfo `json:"Result"`
}

func (r RespQueryTradeInfo) IsSuccess() bool {
	return r.Status == "SUCCESS"
}

func (r RespQueryTradeInfo) RefundAll(a *Api, m *Merchant, requestedAt xtime.Time) (*RespCreditCardBehavior, error) {
	return r.Retain(a, m, 0, requestedAt)
}

func (r RespQueryTradeInfo) Retain(a *Api, m *Merchant, amount int, requestedAt xtime.Time) (*RespCreditCardBehavior, error) {
	if r.Status != "SUCCESS" {
		return nil, fmt.Errorf("[query] %s: %s", r.Status, r.Message)
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
		if _, err := a.CreditCardCancelPaymentRequest(m, result.MerchantOrderNo, result.Amt, requestedAt); err != nil {
			return nil, err
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
