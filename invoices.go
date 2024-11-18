package newebpay

type InvoiceRequest struct {
	RespondType      string  `json:"RespondType"`        // 回應格式 (JSON 或 String) "JSON"
	Version          string  `json:"Version"`            // API 版本號， "1.5"
	TimeStamp        string  `json:"TimeStamp"`          // 時間戳記，Unix 格式
	TransNum         *string `json:"TransNum,omitempty"` // 交易序號 (選填)
	MerchantOrderNo  string  `json:"MerchantOrderNo"`    // 商店訂單編號
	Status           string  `json:"Status"`             // 發票狀態 (1 為立即開立)
	CreateStatusTime string  `json:"CreateStatusTime"`   // 發票開立時間 (選填)
	Category         string  `json:"Category"`           // 發票類別，例如 "B2C"
	BuyerName        string  `json:"BuyerName"`          // 買受人名稱
	BuyerUBN         string  `json:"BuyerUBN"`           // 買受人統一編號 (B2B 必填)
	BuyerAddress     string  `json:"BuyerAddress"`       // 買受人地址
	BuyerEmail       string  `json:"BuyerEmail"`         // 買受人電子信箱
	CarrierType      string  `json:"CarrierType"`        // 載具類別 (選填)
	CarrierNum       string  `json:"CarrierNum"`         // 載具編號 (若有載具類別時必填)
	LoveCode         string  `json:"LoveCode"`           // 愛心碼 (選填，捐贈發票用)
	PrintFlag        string  `json:"PrintFlag"`          // 是否列印紙本發票 (Y 或 N)
	KioskPrintFlag   string  `json:"KioskPrintFlag"`     // 是否列印於 Kiosk (選填)
	TaxType          string  `json:"TaxType"`            // 課稅類別，例如 "1" (應稅)
	TaxRate          string  `json:"TaxRate"`            // 稅率，例如 "5"
	CustomsClearance string  `json:"CustomsClearance"`   // 通關標記 (零稅率適用，選填)
	Amt              string  `json:"Amt"`                // 銷售金額 (未稅)
	AmtSales         string  `json:"AmtSales"`           // 銷售額應稅 (選填)
	AmtZero          string  `json:"AmtZero"`            // 銷售額零稅率 (選填)
	AmtFree          string  `json:"AmtFree"`            // 銷售額免稅 (選填)
	TaxAmt           string  `json:"TaxAmt"`             // 稅額
	TotalAmt         string  `json:"TotalAmt"`           // 總金額 (含稅)
	ItemName         string  `json:"ItemName"`           // 商品名稱 (多項以 | 分隔)
	ItemCount        string  `json:"ItemCount"`          // 商品數量 (多項以 | 分隔)
	ItemUnit         string  `json:"ItemUnit"`           // 商品單位 (多項以 | 分隔)
	ItemPrice        string  `json:"ItemPrice"`          // 商品單價 (多項以 | 分隔)
	ItemAmt          string  `json:"ItemAmt"`            // 商品小計 (多項以 | 分隔)
	ItemTaxType      string  `json:"ItemTaxType"`        // 商品稅別 (選填，多項以 | 分隔)
	Comment          string  `json:"Comment"`            // 備註 (選填)
}

//func IssueInvoice(req InvoiceRequest) (u, error) {
//	postData := url.Values{}
//	postData.Set("RespondType", req.RespondType)
//	postData.Set("Version", req.Version)
//	postData.Set("TimeStamp", req.TimeStamp)
//	postData.Set("TransNum", req.TransNum)
//	postData.Set("MerchantOrderNo", req.MerchantOrderNo)
//	postData.Set("Status", req.Status)
//	postData.Set("CreateStatusTime", req.CreateStatusTime)
//	postData.Set("Category", req.Category)
//	postData.Set("BuyerName", req.BuyerName)
//	postData.Set("BuyerUBN", req.BuyerUBN)
//	postData.Set("BuyerAddress", req.BuyerAddress)
//	postData.Set("BuyerEmail", req.BuyerEmail)
//	postData.Set("CarrierType", req.CarrierType)
//	postData.Set("CarrierNum", req.CarrierNum)
//	postData.Set("LoveCode", req.LoveCode)
//	postData.Set("PrintFlag", req.PrintFlag)
//	postData.Set("KioskPrintFlag", req.KioskPrintFlag)
//	postData.Set("TaxType", req.TaxType)
//	postData.Set("TaxRate", req.TaxRate)
//	postData.Set("CustomsClearance", req.CustomsClearance)
//	postData.Set("Amt", req.Amt)
//	postData.Set("AmtSales", req.AmtSales)
//	postData.Set("AmtZero", req.AmtZero)
//	postData.Set("AmtFree", req.AmtFree)
//	postData.Set("TaxAmt", req.TaxAmt)
//	postData.Set("TotalAmt", req.TotalAmt)
//	postData.Set("ItemName", req.ItemName)
//	postData.Set("ItemCount", req.ItemCount)
//	postData.Set("ItemUnit", req.ItemUnit)
//	postData.Set("ItemPrice", req.ItemPrice)
//	postData.Set("ItemAmt", req.ItemAmt)
//	postData.Set("ItemTaxType", req.ItemTaxType)
//	postData.Set("Comment", req.Comment)
//
//	encryptedData, err := encryptAES(postData.Encode(), req.HashKey, req.HashIV)
//	if err != nil {
//		return "", fmt.Errorf("Encryption failed: %v", err)
//	}
//
//	checkCode := generateCheckCode(postData.Encode(), req.HashKey, req.HashIV)
//	fmt.Println("CheckCode:", checkCode)
//
//	payload := url.Values{}
//	payload.Set("MerchantID_", req.MerchantID)
//	payload.Set("PostData_", encryptedData)
//
//	resp, err := http.PostForm(a.ApiUrlInvoiceIssue, payload)
//	if err != nil {
//		return fmt.Errorf("Failed to submit form: %v", err)
//	}
//
//	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
//
//	client := &http.Client{}
//	resp, err := client.Do(httpReq)
//	if err != nil {
//		return "", fmt.Errorf("Failed to send request: %v", err)
//	}
//	defer resp.Body.Close()
//
//	body, err := ioutil.ReadAll(resp.Body)
//	if err != nil {
//		return "", fmt.Errorf("Failed to read response: %v", err)
//	}
//
//	return string(body), nil
//}
