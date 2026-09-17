package main

import (
	"context"
	"log"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	beanq "github.com/retail-ai-inc/beanq/v4"
	"github.com/retail-ai-inc/beanq/v4/helper/json"
	"github.com/spf13/viper"
)

type Transaction struct {
	IdempotencyKey       string     `json:"IdempotencyKey,omitempty"`
	TransactionID        string     `json:"TransactionId"`
	WalletServiceID      int64      `json:"WalletServiceId"`
	TransactionType      int        `json:"TransactionType"`
	Status               int        `json:"Status"`
	PayerType            string     `json:"PayerType"`
	PayerID              string     `json:"PayerId"`
	PayeeType            *string    `json:"PayeeType,omitempty"`
	PayeeID              *string    `json:"PayeeId,omitempty"`
	Amount               string     `json:"Amount"`
	FeeAmount            string     `json:"FeeAmount,omitempty"`
	NetAmount            string     `json:"NetAmount,omitempty"`
	Currency             string     `json:"Currency"`
	PaymentMethodID      *int64     `json:"PaymentMethodId,omitempty"`
	Channel              *int       `json:"Channel,omitempty"`
	ChannelTransactionID *string    `json:"ChannelTransactionId,omitempty"`
	RelatedTransactionID *string    `json:"RelatedTransactionId,omitempty"`
	OriginTransactionID  *string    `json:"OriginTransactionId,omitempty"`
	CreatedAt            time.Time  `json:"CreatedAt"`
	UpdatedAt            *time.Time `json:"UpdatedAt,omitempty"`
}

var once sync.Once
var config beanq.BeanqConfig

func initConfig() *beanq.BeanqConfig {
	once.Do(func() {
		_, file, _, _ := runtime.Caller(0)
		vp := viper.New()
		vp.AddConfigPath(filepath.Dir(filepath.Dir(file)))
		vp.SetConfigName("env")
		vp.SetConfigType("json")
		if err := vp.ReadInConfig(); err != nil {
			log.Printf("env.json unavailable: %v", err)
		} else if err := vp.Unmarshal(&config); err != nil {
			log.Fatal(err)
		}
	})
	return &config
}

func main() {
	//
	merchantType, merchantID := "MERCHANT", "mer_001"
	tx := Transaction{
		TransactionID:   "txn_payment_002",
		WalletServiceID: 1,
		TransactionType: 2,
		Status:          1,
		PayerType:       "CUSTOMER",
		PayerID:         "cus_001",
		PayeeType:       &merchantType,
		PayeeID:         &merchantID,
		Amount:          "1000.00",
		FeeAmount:       "0.00",
		NetAmount:       "1000.00",
		Currency:        "JPY",
		CreatedAt:       time.Now().UTC()}
	payload, err := json.Marshal(tx)
	if err != nil {
		log.Fatal(err)
	}
	cmd := beanq.New(initConfig()).BQ().
		WithContext(context.Background()).
		SetId(tx.TransactionID).
		PublishSequence("accounting", "transactions", tx.PayerID, payload)
	if err := cmd.Error(); err != nil {
		log.Fatal(err)
	}
}
