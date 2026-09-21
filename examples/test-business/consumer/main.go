package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"time"

	beanq "github.com/retail-ai-inc/beanq/v4"
	"github.com/retail-ai-inc/beanq/v4/helper/json"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	channel              = "accounting"
	customerTopic        = "transactions"
	merchantTopic        = "merchant-postings"
)

type Transaction struct {
	TransactionID string `json:"TransactionId"`
	PayerType     string `json:"PayerType"`
	PayerID       string `json:"PayerId"`
	PayeeType     string `json:"PayeeType"`
	PayeeID       string `json:"PayeeId"`
	Amount        string `json:"Amount"`
	Currency      string `json:"Currency"`
}

type JournalLine struct {
	JournalLineID         string               `bson:"JournalLineId"`
	TransactionID         string               `bson:"TransactionId"`
	AccountID             string               `bson:"AccountId"`
	Sequence              int64                `bson:"Sequence"`
	Direction             string               `bson:"Direction"`
	Currency              string               `bson:"Currency"`
	Amount                primitive.Decimal128 `bson:"Amount"`
	BeforeBalance         primitive.Decimal128 `bson:"BeforeBalance"`
	AfterBalance          primitive.Decimal128 `bson:"AfterBalance"`
	PreviousJournalLineID *string              `bson:"PreviousJournalLineId"`
	JournalLineHash       string               `bson:"JournalLineHash"`
	CreatedAt             time.Time            `bson:"CreatedAt"`
}

type ledger struct {
	journals *mongo.Collection
	salt     string
	publish  *beanq.Client
}

func accountID(ownerType, ownerID, currency string) string {
	return "acct_" + ownerType + "_" + ownerID + "_" + currency
}

func dec(v string) primitive.Decimal128 {
	d, err := primitive.ParseDecimal128(v)
	if err != nil {
		panic(err)
	}
	return d
}

func rat(v string) (*big.Rat, error) {
	r, ok := new(big.Rat).SetString(v)
	if !ok {
		return nil, fmt.Errorf("invalid decimal %q", v)
	}
	return r, nil
}

func money(r *big.Rat) primitive.Decimal128 { return dec(r.FloatString(2)) }

func hashLine(salt, prevHash string, line JournalLine) string {
	prevID := "<null>"
	if line.PreviousJournalLineID != nil {
		prevID = *line.PreviousJournalLineID
	}
	input := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%s|%s|%s|%s|%s|%s|%s",
		salt, prevHash, line.JournalLineID, line.TransactionID, line.AccountID, line.Sequence,
		line.Direction, line.Amount.String(), line.Currency, line.BeforeBalance.String(),
		line.AfterBalance.String(), prevID, line.CreatedAt.UTC().Format(time.RFC3339Nano))
	mac := hmac.New(sha256.New, []byte(salt))
	_, _ = mac.Write([]byte(input))
	return hex.EncodeToString(mac.Sum(nil))
}

func (l *ledger) apply(ctx context.Context, tx Transaction, accID, direction string, amount *big.Rat) error {
	var prev JournalLine
	err := l.journals.FindOne(ctx, bson.M{"AccountId": accID}, options.FindOne().SetSort(bson.D{{Key: "Sequence", Value: -1}})).Decode(&prev)
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}

	before := new(big.Rat)
	seq := int64(1)
	var prevID *string
	prevHash := ""
	if err == nil {
		before, err = rat(prev.AfterBalance.String())
		if err != nil {
			return err
		}
		seq = prev.Sequence + 1
		id := prev.JournalLineID
		prevID = &id
		prevHash = prev.JournalLineHash
	}

	after := new(big.Rat).Set(before)
	if direction == "DEBIT" {
		after.Sub(after, amount)
	} else {
		after.Add(after, amount)
	}

	line := JournalLine{
		JournalLineID:         "jli_" + tx.TransactionID + "_" + accID,
		TransactionID:         tx.TransactionID,
		AccountID:             accID,
		Sequence:              seq,
		Direction:             direction,
		Currency:              tx.Currency,
		Amount:                money(amount),
		BeforeBalance:         money(before),
		AfterBalance:          money(after),
		PreviousJournalLineID: prevID,
		CreatedAt:             time.Now().UTC().Truncate(time.Millisecond),
	}
	line.JournalLineHash = hashLine(l.salt, prevHash, line)
	log.Printf("journal line: %+v", line)

	_, err = l.journals.InsertOne(ctx, line)
	if mongo.IsDuplicateKeyError(err) {
		return nil
	}
	return err
}

func parseTx(payload string) (Transaction, *big.Rat, string, string, error) {
	var tx Transaction
	if err := json.Unmarshal([]byte(payload), &tx); err != nil {
		return Transaction{}, nil, "", "", err
	}
	amount, err := rat(tx.Amount)
	if err != nil || amount.Sign() <= 0 || tx.PayerID == "" || tx.PayeeID == "" {
		return Transaction{}, nil, "", "", fmt.Errorf("invalid transaction")
	}
	payer := accountID(tx.PayerType, tx.PayerID, tx.Currency)
	payee := accountID(tx.PayeeType, tx.PayeeID, tx.Currency)
	if payer == payee {
		return Transaction{}, nil, "", "", fmt.Errorf("payer and payee must differ")
	}
	return tx, amount, payer, payee, nil
}

func (l *ledger) handle(direction string) beanq.DefaultHandle {
	return beanq.DefaultHandle{
		DoHandle: func(ctx context.Context, msg *beanq.Message) error {
			tx, amount, payer, payee, err := parseTx(msg.Payload)
			if err != nil {
				return err
			}
			accID := payer
			if direction == "CREDIT" {
				accID = payee
			}
			if err := l.apply(ctx, tx, accID, direction, amount); err != nil {
				return err
			}
			if direction != "DEBIT" {
				return nil
			}
			body, err := json.Marshal(tx)
			if err != nil {
				return err
			}
			return l.publish.BQ().WithContext(ctx).
				SetId(tx.TransactionID + ":merchant").
				PublishSequence(channel, merchantTopic, payee, body).
				Error()
		},
		DoCancel: func(context.Context, *beanq.Message) error { return nil },
		DoError:  func(_ context.Context, err error) { log.Printf("accounting consumer error: %v", err) },
	}
}

func loadConfig() *beanq.BeanqConfig {
	_, f, _, _ := runtime.Caller(0)
	vp := viper.New()
	vp.AddConfigPath(filepath.Dir(filepath.Dir(f)))
	vp.SetConfigName("env")
	vp.SetConfigType("json")
	_ = vp.ReadInConfig()
	cfg := &beanq.BeanqConfig{}
	_ = vp.Unmarshal(cfg)
	return cfg
}

func main() {
	salt := os.Getenv("ACCOUNTING_JOURNAL_SALT")
	if salt == "" {
		salt = "development-only-salt"
	}
	cfg := loadConfig()
	if cfg == nil || cfg.Mongo == nil {
		log.Fatal("mongo configuration is required")
	}

	uri := fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", cfg.Mongo.UserName, cfg.Mongo.Password, cfg.Mongo.Host, cfg.Mongo.Port, cfg.Mongo.Database)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Mongo.ConnectTimeOut)
	defer cancel()
	db, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Disconnect(context.Background())

	journals := db.Database(cfg.Mongo.Database).Collection("JournalLines")
	_, err = journals.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "JournalLineId", Value: 1}}, Options: options.Index().SetName("JournalLineId_1").SetUnique(true)},
		{Keys: bson.D{{Key: "AccountId", Value: 1}, {Key: "Sequence", Value: 1}}, Options: options.Index().SetName("AccountId_1_Sequence_1")},
	})
	if err != nil {
		log.Fatal(err)
	}

	csm := beanq.New(cfg)
	l := &ledger{journals: journals, salt: salt, publish: csm}
	client := csm.BQ().WithContext(context.Background())
	if _, err = client.SubscribeSequence(channel, customerTopic, l.handle("DEBIT")); err != nil {
		log.Fatal(err)
	}
	if _, err = client.SubscribeSequence(channel, merchantTopic, l.handle("CREDIT")); err != nil {
		log.Fatal(err)
	}
	log.Println("customer and merchant consumers ready")
	csm.Wait(context.Background())
}

