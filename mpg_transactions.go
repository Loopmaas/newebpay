package newebpay

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Loopmaas/misc"
	"github.com/Loopmaas/xtime"
)

type MPGTransaction struct {
	MerchantID  string `json:"MerchantID"`
	TradeInfo   string `json:"TradeInfo"`
	TradeSha    string `json:"TradeSha"`
	Version     string `json:"Version"`
	EncryptType string `json:"EncryptType"`
}

// https://ccore.newebpay.com/MPG/mpg_gateway
type MPGTradeInfo struct {
	MerchantID        string  `json:"MerchantID"`          // 商店代號
	RespondType       string  `json:"RespondType"`         // 回傳格式: JSON
	TimeStamp         string  `json:"TimeStamp"`           // 時間戳記: UTC Unix
	Version           string  `json:"Version"`             // 2.1
	LangType          string  `json:"LangType"`            // zh-tw
	MerchantOrderNo   string  `json:"MerchantOrderNo"`     // 商店訂單編號
	Amt               int     `json:"Amt"`                 // 金額
	ItemDesc          string  `json:"ItemDesc"`            // 商品描述
	ReturnURL         string  `json:"ReturnURL"`           // 支付完成返回商店網址
	NotifyURL         string  `json:"NotifyURL"`           // 支付通知網址
	ClientBackURL     string  `json:"ClientBackURL"`       // 支付取消返回商店網址 /orders/{orderId}
	Email             string  `json:"Email"`               // 付款人電子信箱
	EmailModify       int     `json:"EmailModify"`         // 付款人電子信箱是否開放修改: 1=可修改, 0=不可修改。0
	CREDITAEAGREEMENT int     `json:"CREDITAEAGREEMENT"`   // 美國運通卡啟用: 1=啟用美國運通卡, 0=不啟用。0
	InstFlag          string  `json:"InstFlag"`            // 信用卡分期付款啟用: 0=不開, 1=全開, 3=分3期, 6=分6期, 12=分12期, 18=分18期, 24=分24期, 30=分30期。可多選 "," 分隔。0
	OrderComment      string  `json:"OrderComment"`        // 此參數內容將會於 MPG 頁面呈現給付款人，確認約定信用卡付款之約定事項。
	CREDITAGREEMENT   int     `json:"CREDITAGREEMENT"`     // 約定信用卡付款授權交易。1
	TokenTerm         string  `json:"TokenTerm"`           // 可對應付款人之資料，用於綁定付款人與信用卡卡號時使用
	TokenLife         *string `json:"TokenLife,omitempty"` // 設定 Token 之有效日期，若此參數為空值或設定日期大於信用卡到期日，則系統預設以信用卡到期日為主
	UseFor            int     `json:"UseFor"`              // 0=WEB, 1=APP, 2=定期定額
}

func (a Api) GetBindingCreditCardParams(
	merchant *Merchant,
	email, tokenTerm string,
	returnUrl, notifyUrl, clientBackUrl string,
	requestedAt xtime.Time,
) (*MPGTransaction, error) {
	tradeInfo := MPGTradeInfo{
		MerchantID:        merchant.MerchantId,
		RespondType:       "JSON",
		TimeStamp:         strconv.FormatInt(time.Time(requestedAt).Unix(), 10),
		Version:           "2.1",
		LangType:          "zh-tw",
		MerchantOrderNo:   NewMerchantOrderIdForCreditCardBinding(),
		Amt:               1,
		ItemDesc:          "綁定信用卡",
		ReturnURL:         returnUrl,
		NotifyURL:         notifyUrl,
		ClientBackURL:     clientBackUrl,
		Email:             email,
		EmailModify:       0,
		CREDITAEAGREEMENT: 0,
		InstFlag:          "0",
		OrderComment:      "此為信用卡綁定交易，完成後將會刷退綁定交易的 1 元",
		CREDITAGREEMENT:   1,
		TokenTerm:         tokenTerm,
		TokenLife:         nil,
		UseFor:            0,
	}
	fmt.Printf("!! log get binding credit card params: %+v\n", tradeInfo)
	encTradeInfo, err := encryptData(tradeInfo, merchant.HashKey, merchant.HashIv)
	if err != nil {
		return nil, err
	}

	return &MPGTransaction{
		MerchantID:  merchant.MerchantId,
		TradeInfo:   encTradeInfo,
		TradeSha:    encryptDataSha256(encTradeInfo, merchant.HashKey, merchant.HashIv),
		Version:     "2.1",
		EncryptType: "0",
	}, nil
}

type RespMPGTradeInfo struct {
	Status  string              `json:"Status"`
	Message string              `json:"Message"`
	Result  *ResultMPGTradeInfo `json:"Result"`
}

type ResultMPGTradeInfo struct {
	MerchantID      string `json:"MerchantID"`
	Amt             int    `json:"Amt"`             // 金額
	TradeNo         string `json:"TradeNo"`         // 藍新金流交易序號
	MerchantOrderNo string `json:"MerchantOrderNo"` // 商店訂單編號
	PaymentType     string `json:"PaymentType"`     // 支付方式
	RespondType     string `json:"RespondType"`     // 回傳格式
	PayTime         string `json:"PayTime"`         // 支付完成時間
	IP              string `json:"IP"`              // 付款人取號或交易時的 IP
	EscrowBank      string `json:"EscrowBank"`      // 款項保管銀行
	AuthBank        string `json:"AuthBank"`        // 收單金融機構
	RespondCode     string `json:"RespondCode"`     // 金融機構回應碼：若交易送至收單機構授權時已是失敗狀態，則本欄位的值會以空值回傳
	Auth            string `json:"Auth"`            // 收單機構所回應的授權碼：若交易送至收單機構授權時已是失敗狀態，則本欄位的值會以空值回傳
	Card6No         string `json:"Card6No"`         // 信用卡卡號前六碼
	Card4No         string `json:"Card4No"`         // 信用卡卡號後四碼
	Exp             string `json:"Exp"`             // 信用卡到期日：YYMM ex:1912 為 2019 年 12 月
	Inst            int    `json:"Inst"`            // 信用卡分期交易期別
	InstFirst       int    `json:"InstFirst"`       // 信用卡分期交易首期金額
	InstEach        int    `json:"InstEach"`        // 信用卡分期交易每期金額
	ECI             string `json:"ECI"`             // 3D 回傳值 eci=1,2,5,6，代表為 3D 交易。若交易送至收單機構授權時已是失敗狀態，則本欄位的值會以空值回傳
	TokenUseStatus  int    `json:"TokenUseStatus"`  // 0=非使用信用卡快速結帳功能 1=首次設定信用卡快速結帳功能 2=使用信用卡快速結帳功能 3=取消信用卡快速結帳功能功能
	TokenValue      string `json:"TokenValue"`      // 授權成功才會回傳，提供商店於後續約定付款 Pn 時使用
	TokenLife       string `json:"TokenLife"`       // Token 有效日期：格式為 YYYY-MM-DD。 超過有效日期時無法再以 TokenValue 進行後續約定付款 (Pn)
}

func (r RespMPGTradeInfo) IsSuccess() bool {
	return r.Status == "SUCCESS"
}

func (r RespMPGTradeInfo) GetCreditCardInfo() (string, string, string, string, error) {
	expires, err := convertExpiresToLastDay(r.Result.Exp)
	return r.Result.TokenValue, expires, r.Result.Card6No, r.Result.Card4No, err
}

func convertExpiresToLastDay(expires string) (string, error) {
	if len(expires) != 4 {
		return "", fmt.Errorf("invalid expires format: must be 4 characters")
	}

	year, err := strconv.Atoi("20" + expires[:2])
	if err != nil {
		return "", fmt.Errorf("invalid year in expires: %v", expires[:2])
	}

	month, err := strconv.Atoi(expires[2:])
	if err != nil || month < 1 || month > 12 {
		return "", fmt.Errorf("invalid month in expires: %v", expires[2:])
	}

	firstDayOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1)

	return lastDayOfMonth.Format("2006-01-02"), nil
}

func (r RespMPGTradeInfo) IsCreditCardBinding() bool {
	return strings.HasPrefix(r.Result.MerchantOrderNo, MerchantOrderIdPrefix)
}

func (r RespMPGTradeInfo) MerchantOrderNo() string {
	return r.Result.MerchantOrderNo
}

func (r RespMPGTradeInfo) TransactedAt() (xtime.Time, error) {
	return xtime.Parse("2006-01-02 15:04:05", r.Result.PayTime)
}

func (r RespMPGTradeInfo) MerchantId() string {
	return r.Result.MerchantID
}

func DecryptMPGTradeInfo(encryptedData, hashKey, hashIv string) (*RespMPGTradeInfo, error) {
	tradeInfo := RespMPGTradeInfo{}
	err := decryptData(encryptedData, hashKey, hashIv, &tradeInfo)
	return &tradeInfo, err
}

const (
	TokenTermLen     = 20
	TokenTermCharset = misc.DefCharsetNumber | misc.DefCharsetLowercase | misc.DefCharsetUppercase
)

func NewTokenTerm() string {
	return misc.GenStringWithCharset(TokenTermLen, TokenTermCharset)
}
