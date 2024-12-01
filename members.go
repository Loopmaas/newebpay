package newebpay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/hexcraft-biz/misc"
	"github.com/hexcraft-biz/xtime"
	"github.com/hexcraft-biz/xuuid"
	"github.com/mitchellh/mapstructure"
)

const (
	NewebpayMerchantIdDigitNum = 12
	NewebpayMerchantIdCharset  = misc.DefCharsetNumber
)

func NewMerchantId() string {
	return "LOP" + misc.GenStringWithCharset(NewebpayMerchantIdDigitNum, NewebpayMerchantIdCharset)
}

func NewRequestAddMerchant(member *Member, merchantDetail *MerchantDetail,
	frontendAppRootUrl *url.URL, rentalAgencyId xuuid.UUID, merchantId string, requestedAt xtime.Time,
) *RequestAddMerchant {
	r := RequestAddMerchant{
		Member:         member,
		MerchantDetail: merchantDetail,
	}

	r.Version = "1.9"
	r.TimeStamp = strconv.FormatInt(time.Time(requestedAt).Unix(), 10)

	r.ManagerID = "5;" + r.MemberUnified
	r.IDCardDate = &r.IncorporationDate
	r.RepresentCPAdd = &r.CompanyAddress
	r.RepresentCapitalAmt = &r.CapitalAmount
	r.RepresentManagerName = &r.RepresentName

	r.MerchantID = merchantId
	r.MCType = 1
	r.MerchantWebURL = frontendAppRootUrl.JoinPath("rental-agencies", rentalAgencyId.String()).String()
	r.MerchantType = 2
	r.BusinessType = "7519"
	r.CreditAutoType = 1
	r.CreditLimit = nil
	r.PaymentType = "CREDIT:1"
	r.AgreedFee = "CREDIT:0.02"
	r.AgreedDay = "CREDIT:3"
	r.Withdraw = nil
	r.WithdrawMer = nil
	r.WithdrawSetting = nil

	return &r
}

type Member struct {
	MemberUnified     string `json:"MemberUnified" binding:"required" example:"12345678 - (企業統編)"`                         // 企業統一編號
	RepresentName     string `json:"RepresentName" binding:"required" example:"代表人姓名"`                                     // 企業代表人姓名
	CapitalAmount     string `json:"CapitalAmount" binding:"required" example:"資本額"`                                       // 實收資本額：帶入該企業於工商登記資料相同之資訊
	IncorporationDate string `json:"IncorporationDate" binding:"required" example:"20241116 (核准設立日期)"`                     // 核准設立日期：帶入該企業於工商登記資料相同之資訊
	CompanyAddress    string `json:"CompanyAddress" binding:"required" example:"台北市中山區某某路550號 (公司登記地址)"`                   // 公司登記地址：帶入該企業於工商登記資料相同之資訊
	MemberName        string `json:"MemberName" binding:"required" example:"某某有限公司"`                                       // 公司登記之名稱：帶入該企業於工商登記資料相同之資訊
	MemberPhone       string `json:"MemberPhone" binding:"required" example:"02-22442424"`                                 // 會員聯絡電話
	MemberAddress     string `json:"MemberAddress" binding:"required" example:"台北市中山區某某路550號 (會員聯絡地址)"`                    // 會員聯絡地址
	ManagerName       string `json:"ManagerName" binding:"required" example:"藍新管理者姓名"`                                     // 管理者中文姓名：合作商店於藍新金流平台的主要連絡人及最高權限人員
	ManagerNameE      string `json:"ManagerNameE" binding:"required" example:"English name of Newebpay administrator"`     // 管理者英文姓名：合作商店的管理者英文姓名 {名,姓}
	LoginAccount      string `json:"LoginAccount" binding:"required" example:"藍新管理者帳號"`                                    // 管理者帳號：管理者登入藍新金流平台的帳號, [0-9a-zA-Z_\.@]{5, 20}
	ManagerMobile     string `json:"ManagerMobile" binding:"required" example:"藍新管理者行動電話號碼"`                               // 管理者行動電話號碼：用於日後會員身份確認使用, [0912345678]{10}
	ManagerEmail      string `json:"ManagerEmail" binding:"required" example:"admin@myhost.com (藍新管理者 Email)"`             // 管理者 E-mail
	DisputeMail       string `json:"DisputeMail" binding:"required" example:"dispute@myhost.com (合作商店如發生爭議款相關議題時可供聯絡的信箱)"` // 合作商店如發生爭議款相關議題時可供聯絡的信箱
}

type MerchantDetail struct {
	MerchantEmail    string `json:"MerchantEmail" binding:"required" example:"cs@myhost.com (客服商店信箱)"` // 客服商店信箱 "," delimeter
	MerchantName     string `json:"MerchantName" binding:"required" example:"某某有限公司"`                  // 合作商店中文名稱
	MerchantNameE    string `json:"MerchantNameE" binding:"required" example:"Some Big Company Ltd."`  // 合作商店英文名稱，若商店需啟用信用卡類支付方式時為必填
	MerchantAddrCity string `json:"MerchantAddrCity" binding:"required" example:"台北市"`                 // 聯絡地址 - 城市，合作商店聯絡地址之城市
	MerchantAddrArea string `json:"MerchantAddrArea" binding:"required" example:"中山區"`                 // 聯絡地址 - 地區，合作商店聯絡地址之地區
	MerchantAddrCode string `json:"MerchantAddrCode" binding:"required" example:"247 (Zipcode)"`       // 聯絡地址 - 郵遞區號，合作商店聯絡地址之郵遞區號三碼
	MerchantAddr     string `json:"MerchantAddr" binding:"required" example:"某某路550號"`                 // 聯絡地址 - 路名及門牌號碼，合作商店聯絡地址之路名及門牌號碼
	MerchantEnAddr   string `json:"MerchantEnAddr" binding:"required" example:"Address in English(*)"` // 商店英文聯絡地址
	NationalE        string `json:"NationalE" binding:"required" example:"Taiwan"`                     // 合作商店設立登記營業國家英文名稱 "Taiwan"
	CityE            string `json:"CityE" binding:"required" example:"Taipei"`                         // 設立登記營業城市英文名稱 "Keelung"
	MerchantDesc     string `json:"MerchantDesc" binding:"required" example:"商店簡介(*)"`                 // 商店簡介，"車輛租賃服務"
	BankCode         string `json:"BankCode" binding:"required" example:"銀行代碼"`                        // 合作商店金融機構帳戶金融機構代碼
	SubBankCode      string `json:"SubBankCode" binding:"required" example:"分行代碼"`                     // 合作商店金融機構帳戶金融機構分行代碼
	BankAccount      string `json:"BankAccount" binding:"required" example:"銀行帳戶"`                     // 合作商店金融機構帳戶之帳號
	AccountName      string `json:"AccountName" binding:"required" example:"合作商店銀行帳戶戶名"`               // 合作商店金融機構帳戶戶名
}

type RequestAddMerchant struct {
	Version   string `json:"Version"`
	TimeStamp string `json:"TimeStamp"`

	// 建立會員所需資料
	*Member
	ManagerID            string  `json:"ManagerID"`            // 5;{UBN}
	IDCardDate           *string `json:"IDCardDate"`           // 法人公司的核准設立日期 YYYYMMDD
	RepresentCPAdd       *string `json:"RepresentCPAdd"`       // 法人登記地址
	RepresentCapitalAmt  *string `json:"RepresentCapitalAmt"`  // 法人實收資本額
	RepresentManagerName *string `json:"RepresentManagerName"` // 法人公司代表人名稱

	// 建立商店所需資料
	*MerchantDetail
	MerchantID      string  `json:"MerchantID"`                // 商店代號：格式為金流合作推廣商代號，格式 LOP[0-9]{12}
	MCType          int     `json:"MCType"`                    // 商店類別，1=網路商店
	MerchantWebURL  string  `json:"MerchantWebURL"`            // 合作商店網址
	MerchantType    int     `json:"MerchantType"`              // 販售類別，2=服務
	BusinessType    string  `json:"BusinessType"`              // 行業別，7519
	CreditAutoType  int     `json:"CreditAutoType"`            // 1=自動請款
	CreditLimit     *int    `json:"CreditLimit,omitempty"`     // 合作商店信用卡 30 天收款額，若未帶入此參數，合作推廣商與藍新金流約定之預設值 nil
	PaymentType     string  `json:"PaymentType"`               // [支付方式代號:啟用狀態] CREDIT:1
	AgreedFee       string  `json:"AgreedFee"`                 // 交易手續費 [支付方式代號:費率] CREDIT:0.02
	AgreedDay       string  `json:"AgreedDay"`                 // 撥款天數 [支付方式代號:天數]  CREDIT:3
	Withdraw        *string `json:"Withdraw,omitempty"`        // nil
	WithdrawMer     *string `json:"WithdrawMer,omitempty"`     // nil
	WithdrawSetting *string `json:"WithdrawSetting,omitempty"` // nil
}

func (a Api) AddMerchant(partnerId, hashKey, hashIv string, data *RequestAddMerchant) (*ResultAddMerchant, error) {
	encData, err := encryptData(data, hashKey, hashIv)
	if err != nil {
		return nil, err
	}

	formData := url.Values{
		"PartnerID_": {partnerId},
		"PostData_":  {encData},
	}

	resp, err := http.PostForm(a.ApiUrlAddMerchant, formData)
	if err != nil {
		return nil, fmt.Errorf("Failed to submit form: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var payload RespAddMerchant
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if payload.Status != "SUCCESS" {
		return nil, fmt.Errorf("request failed: [%s]", payload.Status)
	}

	var result ResultAddMerchant
	if err := mapstructure.Decode(payload.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to decode result: %w", err)
	}

	return &result, nil
}

type RespAddMerchant struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  any    `json:"result"`
}

type ResultAddMerchant struct {
	MerchantID      string `json:"MerchantID"`
	MerchantHashKey string `json:"MerchantHashKey"`
	MerchantIvKey   string `json:"MerchantIvKey"`
	MemberType      string `json:"MemberType"`
}
