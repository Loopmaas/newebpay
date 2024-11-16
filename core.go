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
	"os"
	"strconv"
	"strings"
)

type Api struct {
	Env                    string
	PartnerId              string
	HashKey                string
	HashIv                 string
	ApiUrlAddMerchant      string
	ApiUrlMPGTransaction   string
	ApiUrlTransaction      string
	ApiUrlCreditCardCancel string
	ApiUrlCreditCardClose  string
	NotifyRootUrl          *url.URL
	FrontendAppRootUrl     *url.URL
}

func New(notifyRootUrl, frontendAppRootUrl *url.URL) *Api {
	api := Api{
		Env:                os.Getenv("NEWEBPAY_ENV"),
		PartnerId:          os.Getenv("NEWEBPAY_PARTNER_ID"),
		HashKey:            os.Getenv("NEWEBPAY_HASH_KEY"),
		HashIv:             os.Getenv("NEWEBPAY_HASH_IV"),
		NotifyRootUrl:      notifyRootUrl,
		FrontendAppRootUrl: frontendAppRootUrl,
	}

	switch api.Env {
	case "production":
		api.ApiUrlAddMerchant = "https://core.newebpay.com/API/AddMerchant"
		api.ApiUrlMPGTransaction = "https://core.newebpay.com/MPG/mpg_gateway"
		api.ApiUrlTransaction = "https://core.newebpay.com/API/CreditCard"
		api.ApiUrlCreditCardCancel = "https://core.newebpay.com/API/CreditCard/Cancel"
		api.ApiUrlCreditCardClose = "https://core.newebpay.com/API/CreditCard/Close"
	default:
		api.ApiUrlAddMerchant = "https://ccore.newebpay.com/API/AddMerchant"
		api.ApiUrlMPGTransaction = "https://ccore.newebpay.com/MPG/mpg_gateway"
		api.ApiUrlTransaction = "https://ccore.newebpay.com/API/CreditCard"
		api.ApiUrlCreditCardCancel = "https://ccore.newebpay.com/API/CreditCard/Cancel"
		api.ApiUrlCreditCardClose = "https://ccore.newebpay.com/API/CreditCard/Close"
	}

	return &api
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
