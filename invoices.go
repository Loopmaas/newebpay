package newebpay

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Loopmaas/xtime"
)

type IssueInvoicePostData struct {
	RespondType      string  `json:"RespondType"`                // 回應格式 (JSON 或 String) "JSON"
	Version          string  `json:"Version"`                    // API 版本號， "1.5"
	TimeStamp        string  `json:"TimeStamp"`                  // 時間戳記，Unix 格式
	TransNum         *string `json:"TransNum,omitempty"`         // 交易序號 (選填)
	MerchantOrderNo  string  `json:"MerchantOrderNo"`            // 商店訂單編號
	Status           string  `json:"Status"`                     // 發票狀態 (1 為立即開立)
	CreateStatusTime *string `json:"CreateStatusTime,omitempty"` // 預計開立日期，預約自動開立發票時才需要帶此參數
	Category         string  `json:"Category"`                   // 發票類別，B2C
	BuyerName        string  `json:"BuyerName"`                  // 買受人名稱, 個人姓名
	BuyerUBN         *string `json:"BuyerUBN,omitempty"`         // 買受人統一編號 B2B 必填
	BuyerAddress     *string `json:"BuyerAddress,omitempty"`     // 買受人地址
	BuyerEmail       string  `json:"BuyerEmail"`                 // 買受人電子信箱
	CarrierType      string  `json:"CarrierType"`                // 載具類別 (選填, 0)
	CarrierNum       string  `json:"CarrierNum"`                 // 載具編號 (若有載具類別時必填, /[0-9A-Z\+-]{7})
	LoveCode         *string `json:"LoveCode,omitempty"`         // 愛心碼 (選填，捐贈發票用)
	PrintFlag        string  `json:"PrintFlag"`                  // 是否列印紙本發票 (Y 或 [N])
	KioskPrintFlag   *string `json:"KioskPrintFlag,omitempty"`   // 是否列印於 Kiosk (選填)
	TaxType          string  `json:"TaxType"`                    // 課稅類別，例如 "1" (應稅)
	TaxRate          string  `json:"TaxRate"`                    // 稅率，例如 "5"
	CustomsClearance *string `json:"CustomsClearance,omitempty"` // 通關標記 (零稅率適用，選填)
	Amt              int     `json:"Amt"`                        // 銷售金額 (未稅)
	AmtSales         *int    `json:"AmtSales,omitempty"`         // 銷售額應稅 (選填)
	AmtZero          *int    `json:"AmtZero,omitempty"`          // 銷售額零稅率 (選填)
	AmtFree          *int    `json:"AmtFree,omitempty"`          // 銷售額免稅 (選填)
	TaxAmt           int     `json:"TaxAmt"`                     // 稅額
	TotalAmt         int     `json:"TotalAmt"`                   // 總金額 (含稅)
	ItemName         string  `json:"ItemName"`                   // 商品名稱 (多項以 | 分隔)
	ItemCount        string  `json:"ItemCount"`                  // 商品數量 (多項以 | 分隔)
	ItemUnit         string  `json:"ItemUnit"`                   // 商品單位 (多項以 | 分隔)
	ItemPrice        string  `json:"ItemPrice"`                  // 商品單價 (多項以 | 分隔), 含稅金額
	ItemAmt          string  `json:"ItemAmt"`                    // 商品小計 (多項以 | 分隔)
	ItemTaxType      *string `json:"ItemTaxType,omitempty"`      // 商品稅別 (選填，多項以 | 分隔)
	// ItemRate         string  `json:"ItemRate"`                   // 商品稅率 (多項以 | 分隔)
	Comment string `json:"Comment"` // 備註 (選填)
}

func calcTaxExclusiveSalesAmount(amount int) (int, int) {
	taxExclusiveSalesAmount := int(math.Round(float64(amount) / 1.05))
	taxAmount := amount - taxExclusiveSalesAmount
	return taxExclusiveSalesAmount, taxAmount
}

type InvoiceItem struct {
	Name  string
	Count int
	Unit  string
	Price int // B2C 含稅金額
}

func (ii InvoiceItem) Amount() int {
	return ii.Count * ii.Price
}

func (a Api) IssueInvoice(merchant *Merchant,
	name, email string, mobileCarrierNum *string,
	merchantOrderNo string, items []*InvoiceItem, requestedAt xtime.Time,
	carrier_type *int, invoice_carrie *string,
) (*RespInvoiceIssue, error) {
	fmt.Println("[newebpay] IssueInvoice")
	itemLen := len(items)
	if itemLen <= 0 {
		return nil, errors.New("Missing item")
	}

	totalAmount := 0
	itemNames := make([]string, itemLen)
	itemCounts := make([]string, itemLen)
	itemUnits := make([]string, itemLen)
	itemPrices := make([]string, itemLen)
	itemAmts := make([]string, itemLen)
	itemRates := make([]string, itemLen)

	for i, item := range items {
		amount := item.Amount()
		totalAmount += amount

		itemNames[i] = item.Name
		itemCounts[i] = strconv.Itoa(item.Count)
		itemUnits[i] = item.Unit
		itemRates[i] = "1"
		if carrier_type != nil && *carrier_type == 3 {
			itemPrices[i] = strconv.Itoa(int(math.Round(float64(item.Price) / 1.05)))
			itemAmts[i] = strconv.Itoa(int(math.Round(float64(amount) / 1.05)))
		} else {
			itemPrices[i] = strconv.Itoa(item.Price)
			itemAmts[i] = strconv.Itoa(amount)
		}

	}

	taxExclusiveSalesAmount, taxAmount := calcTaxExclusiveSalesAmount(totalAmount)

	printFlag := "Y"
	carrierType := ""
	carrierNum := ""

	category := "B2C"
	buyerNBN := ""
	if mobileCarrierNum != nil {
		printFlag = "N"
		carrierType = "0"
		carrierNum = *mobileCarrierNum
	}

	if carrier_type != nil {
		switch *carrier_type {
		case 2: // 手機個人載具
			printFlag = "N"
			carrierType = "0"
			carrierNum = *invoice_carrie
		case 3: // 營業人 B2B 統編
			category = "B2B"
			buyerNBN = *invoice_carrie
		}
	}

	ItemTaxType := "1"
	if len(items) > 0 {
		ItemTaxType = strings.Join(itemRates, "|")
	}
	postData := IssueInvoicePostData{
		RespondType:      "JSON",
		Version:          "1.5",
		TimeStamp:        strconv.FormatInt(time.Time(requestedAt).Unix(), 10),
		TransNum:         nil,
		MerchantOrderNo:  merchantOrderNo,
		Status:           "1",
		CreateStatusTime: nil,
		Category:         category,
		BuyerName:        name,
		BuyerUBN:         &buyerNBN,
		BuyerAddress:     nil,
		BuyerEmail:       email,
		CarrierType:      carrierType,
		CarrierNum:       carrierNum,
		LoveCode:         nil,
		PrintFlag:        printFlag,
		KioskPrintFlag:   nil,
		TaxType:          "1",
		TaxRate:          "5",
		CustomsClearance: nil,
		Amt:              taxExclusiveSalesAmount,
		AmtSales:         nil,
		AmtZero:          nil,
		AmtFree:          nil,
		TaxAmt:           taxAmount,
		TotalAmt:         totalAmount,
		ItemName:         strings.Join(itemNames, "|"),
		ItemCount:        strings.Join(itemCounts, "|"),
		ItemUnit:         strings.Join(itemUnits, "|"),
		ItemPrice:        strings.Join(itemPrices, "|"),
		ItemAmt:          strings.Join(itemAmts, "|"),
		ItemTaxType:      &ItemTaxType,
		Comment:          "",
	}

	jsonData, err := json.Marshal(postData)
	if err != nil {
		fmt.Println("postData json.Marshal error:", err)
	}
	fmt.Println("postData json:", string(jsonData))
	encData, err := encryptData(postData, merchant.HashKey, merchant.HashIv)
	if err != nil {
		return nil, fmt.Errorf("Encryption failed: %v", err)
	}

	formData := url.Values{
		"MerchantID_": {merchant.MerchantId},
		"PostData_":   {encData},
	}

	resp, err := http.PostForm(a.ApiUrlInvoiceIssue, formData)
	if err != nil {
		fmt.Println("http.PostForm error:", err)
		return nil, fmt.Errorf("Failed to submit form: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("http response status code:", resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	receivedData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read received data failed: %v", err)
	}

	var tp RespPayload
	if err := json.Unmarshal(receivedData, &tp); err != nil {
		return nil, fmt.Errorf("[issue-invoice] failed to decode response: %v, received data: %s", err, string(receivedData))
	}

	if !tp.IsSuccess() {
		return nil, fmt.Errorf("[issue-invoice] %s: %s", tp.Status, tp.Message)
	}

	payload := RespInvoiceIssue{
		Status:  tp.Status,
		Message: tp.Message,
	}
	if err := tp.AssertString(&payload.Result); err != nil {
		return nil, fmt.Errorf("[issue-invoice] assert: %v", err)
	}

	return &payload, nil
}

type RespInvoiceIssue struct {
	Status  string             `json:"Status"`
	Message string             `json:"Message"`
	Result  ResultInvoiceIssue `json:"Result"`
}

func (r RespInvoiceIssue) IsSuccess() bool {
	return r.Status == "SUCCESS"
}

type ResultInvoiceIssue struct {
	MerchantID      string  `json:"MerchantID"`
	InvoiceTransNo  string  `json:"InvoiceTransNo"`
	MerchantOrderNo string  `json:"MerchantOrderNo"`
	TotalAmt        int     `json:"TotalAmt"`
	InvoiceNumber   string  `json:"InvoiceNumber"`
	RandomNum       string  `json:"RandomNum"`
	CreateTime      string  `json:"CreateTime"`
	CheckCode       string  `json:"CheckCode"`
	BarCode         *string `json:"BarCode,omitempty"`
	QRcodeL         *string `json:"QRcodeL,omitempty"`
	QRcodeR         *string `json:"QRcodeR,omitempty"`
}

func (a Api) MemoInvoice(merchant *Merchant,
	name, email string,
	invoiceNo, merchantOrderNo string,
	items []*InvoiceItem,
	requestedAt xtime.Time,
) (*RespInvoiceMemo, error) {
	fmt.Println("[newebpay] MemoInvoice")
	itemLen := len(items)
	if itemLen <= 0 {
		return nil, errors.New("Missing item")
	}

	totalAmount := 0
	itemNames := make([]string, itemLen)
	itemCounts := make([]string, itemLen)
	itemUnits := make([]string, itemLen)
	itemPrices := make([]string, itemLen)
	itemAmts := make([]string, itemLen)
	itemTaxAmts := make([]string, itemLen)

	ItemTaxType := "1"
	for i, item := range items {
		amount := item.Amount()
		totalAmount += amount

		taxExclusiveSalesAmount, taxAmount := calcTaxExclusiveSalesAmount(amount)

		itemNames[i] = item.Name
		itemCounts[i] = strconv.Itoa(item.Count)
		itemUnits[i] = item.Unit
		itemPrices[i] = strconv.Itoa(item.Price)
		itemAmts[i] = strconv.Itoa(taxExclusiveSalesAmount)
		itemTaxAmts[i] = strconv.Itoa(taxAmount)

	}

	postData := struct {
		RespondType     string  `json:"RespondType"` // JSON
		Version         string  `json:"Version"`     // 1.3
		TimeStamp       string  `json:"TimeStamp"`
		InvoiceNo       string  `json:"InvoiceNo"`
		MerchantOrderNo string  `json:"MerchantOrderNo"`
		ItemName        string  `json:"ItemName"`
		ItemCount       string  `json:"ItemCount"`
		ItemUnit        string  `json:"ItemUnit"`
		ItemPrice       string  `json:"ItemPrice"`
		ItemAmt         string  `json:"ItemAmt"`
		TaxTypeForMixed *string `json:"TaxTypeForMixed,omitempty"` //  nil
		ItemTaxAmt      string  `json:"ItemTaxAmt"`                // 0
		TotalAmt        int     `json:"TotalAmt"`
		BuyerEmail      string  `json:"BuyerEmail"`
		Status          string  `json:"Status"` // 1
	}{
		RespondType:     "JSON",
		Version:         "1.3",
		TimeStamp:       strconv.FormatInt(time.Time(requestedAt).Unix(), 10),
		InvoiceNo:       invoiceNo,
		MerchantOrderNo: merchantOrderNo,
		ItemName:        strings.Join(itemNames, "|"),
		ItemCount:       strings.Join(itemCounts, "|"),
		ItemUnit:        strings.Join(itemUnits, "|"),
		ItemPrice:       strings.Join(itemPrices, "|"),
		ItemAmt:         strings.Join(itemAmts, "|"),
		TaxTypeForMixed: &ItemTaxType,
		ItemTaxAmt:      strings.Join(itemTaxAmts, "|"),
		TotalAmt:        totalAmount,
		BuyerEmail:      email,
		Status:          "1",
	}

	encData, err := encryptData(postData, merchant.HashKey, merchant.HashIv)
	if err != nil {
		return nil, fmt.Errorf("Encryption failed: %v", err)
	}

	formData := url.Values{
		"MerchantID_": {merchant.MerchantId},
		"PostData_":   {encData},
	}

	resp, err := http.PostForm(a.ApiUrlInvoiceMemo, formData)
	if err != nil {
		return nil, fmt.Errorf("Failed to submit form: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var tp RespPayload
	if err := json.NewDecoder(resp.Body).Decode(&tp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if !tp.IsSuccess() {
		return nil, fmt.Errorf("[memo-invoice] %s: %s invoiceNo: %s", tp.Status, tp.Message, invoiceNo)
	}

	payload := RespInvoiceMemo{
		Status:  tp.Status,
		Message: tp.Message,
	}
	if err := tp.AssertString(&payload.Result); err != nil {
		return nil, fmt.Errorf("[memo-invoice] assert: %v", err)
	}

	return &payload, nil
}

type RespInvoiceMemo struct {
	Status  string             `json:"Status"`
	Message string             `json:"Message"`
	Result  *ResultInvoiceMemo `json:"Result"`
}

func (r RespInvoiceMemo) IsSuccess() bool {
	return r.Status == "SUCCESS"
}

type ResultInvoiceMemo struct {
	MerchantID      string `json:"MerchantID"`
	AllowanceNo     string `json:"AllowanceNo"`
	InvoiceNumber   string `json:"InvoiceNumber"`
	MerchantOrderNo string `json:"MerchantOrderNo"`
	AllowanceAmt    int    `json:"AllowanceAmt"`
	RemainAmt       int    `json:"RemainAmt"`
	CheckCode       string `json:"CheckCode"`
}
