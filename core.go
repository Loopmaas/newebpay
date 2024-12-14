package newebpay

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/hexcraft-biz/misc"
)

type Api struct {
	Env                    string
	ApiUrlAddMerchant      string
	ApiUrlMPGTransaction   string
	ApiUrlTransaction      string
	ApiUrlCreditCardCancel string
	ApiUrlCreditCardClose  string
	ApiUrlInvoiceIssue     string
	ApiUrlInvoiceMemo      string
	ApiUrlQueryTradeInfo   string
}

func New(env string) *Api {
	switch env {
	case "production":
		return &Api{
			Env:                    env,
			ApiUrlAddMerchant:      "https://core.newebpay.com/API/AddMerchant",
			ApiUrlMPGTransaction:   "https://core.newebpay.com/MPG/mpg_gateway",
			ApiUrlTransaction:      "https://core.newebpay.com/API/CreditCard",
			ApiUrlCreditCardCancel: "https://core.newebpay.com/API/CreditCard/Cancel",
			ApiUrlCreditCardClose:  "https://core.newebpay.com/API/CreditCard/Close",
			ApiUrlInvoiceIssue:     "https://inv.ezpay.com.tw/Api/invoice_issue",
			ApiUrlInvoiceMemo:      "https://inv.ezpay.com.tw/Api/allowance_issue",
			ApiUrlQueryTradeInfo:   "https://core.newebpay.com/API/QueryTradeInfo",
		}
	default:
		return &Api{
			Env:                    env,
			ApiUrlAddMerchant:      "https://ccore.newebpay.com/API/AddMerchant",
			ApiUrlMPGTransaction:   "https://ccore.newebpay.com/MPG/mpg_gateway",
			ApiUrlTransaction:      "https://ccore.newebpay.com/API/CreditCard",
			ApiUrlCreditCardCancel: "https://ccore.newebpay.com/API/CreditCard/Cancel",
			ApiUrlCreditCardClose:  "https://ccore.newebpay.com/API/CreditCard/Close",
			ApiUrlInvoiceIssue:     "https://cinv.ezpay.com.tw/Api/invoice_issue",
			ApiUrlInvoiceMemo:      "https://cinv.ezpay.com.tw/Api/allowance_issue",
			ApiUrlQueryTradeInfo:   "https://ccore.newebpay.com/API/QueryTradeInfo",
		}
	}

	return nil
}

func encryptData(data interface{}, hashKey, hashIv string) (string, error) {
	queryData, err := httpBuildQuery(data)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(hashKey))
	if err != nil {
		return "", err
	}

	paddedData := PKCS7Padding([]byte(queryData), block.BlockSize())
	ciphertext := make([]byte, len(paddedData))
	mode := cipher.NewCBCEncrypter(block, []byte(hashIv))
	mode.CryptBlocks(ciphertext, []byte(paddedData))

	return hex.EncodeToString(ciphertext), nil
}

func httpBuildQuery(data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return "", err
	}

	dataMap := make(map[string]string)
	for key, value := range result {
		switch v := value.(type) {
		case string:
			dataMap[key] = v
		case int:
			dataMap[key] = strconv.Itoa(v)
		default:
			dataMap[key] = fmt.Sprintf("%v", v)
		}
	}

	values := url.Values{}
	for key, value := range dataMap {
		values.Set(key, value)
	}

	return values.Encode(), nil
}

func encryptDataSha256(encData, hashKey, hashIv string) string {
	hash := sha256.Sum256([]byte("HashKey=" + hashKey + "&" + encData + "&HashIV=" + hashIv))
	return strings.ToUpper(hex.EncodeToString(hash[:]))
}

func genCheckCode(amount int, merchantId, merchantOrderNo, tradeNo, hashKey, hashIv string) (string, error) {
	params := map[string]interface{}{
		"Amt":             amount,
		"MerchantID":      merchantId,
		"MerchantOrderNo": merchantOrderNo,
		"TradeNo":         tradeNo,
	}

	queryString, err := httpBuildQuery(params)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256([]byte("HashIV=" + hashIv + "&" + queryString + "&HashKey=" + hashKey))
	return strings.ToUpper(hex.EncodeToString(hash[:])), nil
}

func decryptData(encryptedData, hashKey, hashIv string, result interface{}) error {
	ciphertext, err := hex.DecodeString(encryptedData)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher([]byte(hashKey))
	if err != nil {
		return err
	}

	if len(ciphertext)%block.BlockSize() != 0 {
		return errors.New("ciphertext is not a multiple of the block size")
	}

	decrypted := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, []byte(hashIv))
	mode.CryptBlocks(decrypted, ciphertext)

	unpaddedData, err := PKCS7Unpadding(decrypted)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(unpaddedData, result); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

func PKCS7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

func PKCS7Unpadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("data length is zero")
	}

	padding := int(data[length-1])
	if padding > length {
		return nil, errors.New("invalid padding size")
	}

	return data[:length-padding], nil
}

type Merchant struct {
	MerchantId string `json:"merchantId"`
	HashKey    string `json:"hashKey"`
	HashIv     string `json:"hashIv"`
}

func NewMerchant(merchantId, hashKey, hashIv string) *Merchant {
	return &Merchant{
		MerchantId: merchantId,
		HashKey:    hashKey,
		HashIv:     hashIv,
	}
}

type RespPayload struct {
	Status  string `json:"Status"`
	Message string `json:"Message"`
	Result  []byte `json:"Result"`
}

func (r RespPayload) IsSuccess() bool {
	return r.Status == "SUCCESS"
}

func (r RespPayload) Assert(buf any) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, buf); err != nil {
		return err
	}

	return nil
}

const (
	MerchantOrderNoLen     = 20
	MerchantOrderIdCharset = misc.DefCharsetNumber | misc.DefCharsetLowercase | misc.DefCharsetUppercase
	MerchantOrderIdPrefix  = "CCB_"
)

func NewMerchantOrderNo() string {
	return misc.GenStringWithCharset(MerchantOrderNoLen, MerchantOrderIdCharset)
}

func NewMerchantOrderIdForCreditCardBinding() string {
	return MerchantOrderIdPrefix + misc.GenStringWithCharset(MerchantOrderNoLen-len(MerchantOrderIdPrefix), MerchantOrderIdCharset)
}
