package newebpay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hexcraft-biz/xtime"
	"github.com/mitchellh/mapstructure"
)

type Transaction struct {
	MerchantID_ string `json:"MerchantID_"` // 商店代號
	PostData_   string `json:"PostData_"`   // 加密資料
	Pos_        string `json:"Pos_"`        // 回傳格式: JSON
}

type TransactionPostData struct {
	TimeStamp       string `json:"TimeStamp"`       // 時間戳記: UTC Unix time
	Version         string `json:"Version"`         // 串接程式版本: 2.1
	P3D             string `json:"P3D"`             // 3D 交易: 0
	UseFor          int    `json:"UseFor"`          // 使用情境: 0=WEB, 1=APP, 2=定期定額, 0
	NotifyURL       string `json:"NotifyURL"`       // 支付通知網址: 本欄位僅支援 3D 交易
	ReturnURL       string `json:"ReturnURL"`       // 支付完成返回商店網址: 本欄位僅支援 3D 交易
	MerchantOrderNo string `json:"MerchantOrderNo"` // 商店訂單編號
	Amt             int    `json:"Amt"`             // 訂單金額
	ProdDesc        string `json:"ProdDesc"`        // 商品描述: len 50
	PayerEmail      string `json:"PayerEmail"`      // 付款人電子信箱
	Inst            string `json:"Inst"`            // 信用卡分期付款啟用: 此欄位值=0或無值時，即代表不開啟分期
	TokenValue      string `json:"TokenValue"`      // 約定 Token: 為首次約定付款 (P1) 成功時，所回傳之 TokenValue 值
	TokenTerm       string `json:"TokenTerm"`       // Token 名稱: 為首次約定付款 (P1) 時，所使用之 TokenTerm 值
	TokenSwitch     string `json:"TokenSwitch"`     // Token 類別: on
}

func (a Api) CreditCardTransactionDownPayment1(c *gin.Context,
	merchant *Merchant, merchantOrderNo string,
	email string,
	tokenTerm, tokenValue string,
	amount int,
	notifyUrl, returnUrl string,
	requestedAt xtime.Time,
) error {
	data := TransactionPostData{
		TimeStamp:       strconv.FormatInt(time.Time(requestedAt).Unix(), 10),
		Version:         "2.1",
		P3D:             "1",
		UseFor:          0,
		NotifyURL:       notifyUrl,
		ReturnURL:       returnUrl,
		MerchantOrderNo: merchantOrderNo,
		Amt:             amount,
		ProdDesc:        "租賃訂金 (30%)",
		PayerEmail:      email,
		Inst:            "0",
		TokenValue:      tokenValue,
		TokenTerm:       tokenTerm,
		TokenSwitch:     "on",
	}

	encData, err := encryptData(data, merchant.HashKey, merchant.HashIv)
	if err != nil {
		return err
	}

	formData := url.Values{
		"MerchantID_": {merchant.MerchantId},
		"PostData_":   {encData},
		"Pos_":        {"JSON"},
	}

	resp, err := http.PostForm(a.ApiUrlTransaction, formData)
	if err != nil {
		return fmt.Errorf("Failed to submit form: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var payload RespTransaction
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if payload.Status != "3dVerify" {
		return fmt.Errorf("request failed: [%s]", payload.Status)
	}

	result, ok := payload.Result.(string)
	if !ok {
		return fmt.Errorf("result assertion failed")
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(result))
	return nil
}

type RespTransaction struct {
	Status  string `json:"Status"`
	Message string `json:"Message"`
	Result  any    `json:"Result"` // 當 Status 為 3dVerify 時, 此時會回傳 Html 字串
}

func (r RespTransaction) IsSuccess() bool {
	return r.Status == "SUCCESS"
}

type ResultTransaction struct {
	MerchantID      string  `json:"MerchantID"`      // 商店代號
	Amt             int     `json:"Amt"`             // 交易金額
	TradeNo         string  `json:"TradeNo"`         // 藍新金流交易序號
	MerchantOrderNo string  `json:"MerchantOrderNo"` // 商店訂單編號
	RespondCode     string  `json:"RespondCode"`     // 金融機構回應碼
	AuthBank        string  `json:"AuthBank"`        // 收單金融機構
	Auth            string  `json:"Auth"`            // 授權碼
	AuthDate        string  `json:"AuthDate"`        // 授權日期
	AuthTime        string  `json:"AuthTime"`        // 授權時間
	Card6No         string  `json:"Card6No"`         // 卡號前六碼
	Card4No         string  `json:"Card4No"`         // 卡號後四碼
	Exp             string  `json:"Exp"`             // 信用卡到期日
	Inst            int     `json:"Inst"`            // 信用卡分期交易期別
	InstFirst       int     `json:"InstFirst"`       // 信用卡分期交易首期金額
	InstEach        int     `json:"InstEach"`        // 信用卡分期交易每期金額
	ECI             string  `json:"ECI"`             // eci=1,2,5,6，代表為 3D 交易, 若非 3D 交易或交易送至收單機構授權時已是失敗狀態，則本欄位的值會以空值回傳
	PaymentMethod   string  `json:"PaymentMethod"`   // 交易類別, CREDIT=台灣發卡機構核發之信用卡 FOREIGN=國外發卡機構核發之信用卡
	IP              *string `json:"IP,omitempty"`    // 付款人交易時的 IP [newebpay issue]
	EscrowBank      string  `json:"EscrowBank"`      // 款項保管銀行
	CheckCode       string  `json:"CheckCode"`       // 檢核碼
	TokenLife       string  `json:"TokenLife"`       // Token 有效日期
	TokenUseStatus  int     `json:"TokenUseStatus"`  // [newebpay issue]
}

func (r ResultTransaction) TransactedAt() (xtime.Time, error) {
	location, err := time.LoadLocation("Asia/Taipei") // UTC+8
	if err != nil {
		return xtime.Time{}, err
	}

	layout := "20060102150405"

	parsedTime, err := time.ParseInLocation(layout, r.AuthDate+r.AuthTime, location)
	if err != nil {
		return xtime.Time{}, err
	}

	return xtime.Time(parsedTime.UTC()), nil
}

func (r ResultTransaction) VerifyCheckCode(hashKey, hashIv string) (bool, error) {
	checkCode, err := genCheckCode(r.Amt, r.MerchantID, r.MerchantOrderNo, r.TradeNo, hashKey, hashIv)
	if err != nil {
		return false, err
	}

	return (checkCode == r.CheckCode), nil
}

func (r ResultTransaction) GetMerchantId() string {
	return r.MerchantID
}

func (r ResultTransaction) GetTransactionId() string {
	return r.MerchantOrderNo
}

func (a Api) CreditCardTransaction(merchant *Merchant, email string,
	merchantOrderNo, prodDesc, tokenTerm, tokenValue string,
	amount int,
	requestedAt xtime.Time,
) (*RespTransaction, error) {
	data := TransactionPostData{
		TimeStamp:       strconv.FormatInt(time.Time(requestedAt).Unix(), 10),
		Version:         "2.1",
		P3D:             "0",
		UseFor:          0,
		NotifyURL:       "",
		ReturnURL:       "",
		MerchantOrderNo: merchantOrderNo,
		Amt:             amount,
		ProdDesc:        prodDesc,
		PayerEmail:      email,
		Inst:            "0",
		TokenValue:      tokenValue,
		TokenTerm:       tokenTerm,
		TokenSwitch:     "on",
	}

	encData, err := encryptData(data, merchant.HashKey, merchant.HashIv)
	if err != nil {
		return nil, err
	}

	formData := url.Values{
		"MerchantID_": {merchant.MerchantId},
		"PostData_":   {encData},
		"Pos_":        {"JSON"},
	}

	resp, err := http.PostForm(a.ApiUrlTransaction, formData)
	if err != nil {
		return nil, fmt.Errorf("Failed to submit form: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var payload RespTransaction
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	var result ResultTransaction
	if err := mapstructure.Decode(payload.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to decode result: %w", err)
	}
	payload.Result = &result

	return &payload, nil
}
