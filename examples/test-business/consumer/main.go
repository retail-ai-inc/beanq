package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	beanq "github.com/retail-ai-inc/beanq/v4"
	"github.com/retail-ai-inc/beanq/v4/helper/json"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Transaction struct {
	TransactionID      string `json:"TransactionId"`
	WalletServiceID    int64  `json:"WalletServiceId"`
	TransactionType    int    `json:"TransactionType"`
	Status             int    `json:"Status"`
	PayerType, PayerID string
	PayeeType          *string `json:"PayeeType"`
	PayeeID            *string `json:"PayeeId"`
	Amount             string  `json:"Amount"`
	Currency           string  `json:"Currency"`
}
type JournalLine struct {
	JournalLineID         string    `bson:"JournalLineId"`
	TransactionID         string    `bson:"TransactionId"`
	AccountID             string    `bson:"AccountId"`
	Sequence              int64     `bson:"Sequence"`
	Direction             string    `bson:"Direction"`
	Currency              string    `bson:"Currency"`
	Amount                string    `bson:"Amount"`
	BeforeBalance         string    `bson:"BeforeBalance"`
	AfterBalance          string    `bson:"AfterBalance"`
	PreviousJournalLineID *string   `bson:"PreviousJournalLineId"`
	JournalLineHash       string    `bson:"JournalLineHash"`
	CreatedAt             time.Time `bson:"CreatedAt"`
}

func decodeJournalLine(document bson.M) (JournalLine, error) {
	for _, field := range []string{"Amount", "BeforeBalance", "AfterBalance"} {
		if amount, ok := document[field].(primitive.Decimal128); ok {
			document[field] = amount.String()
		}
	}
	var line JournalLine
	data, err := bson.Marshal(document)
	if err != nil {
		return line, err
	}
	err = bson.Unmarshal(data, &line)
	return line, err
}

func loadVerifiedAccount(ctx context.Context, collection *mongo.Collection, account string, salt string) (*AccountState, error) {
	cursor, err := collection.Find(ctx, bson.M{"AccountId": account}, options.Find().SetSort(bson.D{{Key: "AccountId", Value: -1}, {Key: "Sequence", Value: -1}}).SetLimit(3))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var documents []bson.M
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, err
	}
	lines := make([]JournalLine, len(documents))
	for index, document := range documents {
		lines[index], err = decodeJournalLine(document)
		if err != nil {
			return nil, err
		}
	}
	for index := len(lines) - 1; index >= 0; index-- {
		line := lines[index]
		previousHash := ""
		if line.PreviousJournalLineID != nil {
			var previous JournalLine
			if index+1 < len(lines) {
				previous = lines[index+1]
			} else {
				var document bson.M
				if err := collection.FindOne(ctx, bson.M{"AccountId": account, "JournalLineId": *line.PreviousJournalLineID}).Decode(&document); err != nil {
					return nil, fmt.Errorf("account %s: previous journal line: %w", account, err)
				}
				previous, err = decodeJournalLine(document)
				if err != nil {
					return nil, err
				}
			}
			before, beforeErr := decimal(line.BeforeBalance)
			after, afterErr := decimal(previous.AfterBalance)
			if previous.JournalLineID != *line.PreviousJournalLineID || line.Sequence != previous.Sequence+1 || beforeErr != nil || afterErr != nil || before.Cmp(after) != 0 {
				return nil, fmt.Errorf("account %s: broken journal chain at %s", account, line.JournalLineID)
			}
			previousHash = previous.JournalLineHash
		} else {
			before, err := decimal(line.BeforeBalance)
			if index != len(lines)-1 || line.Sequence != 1 || err != nil || before.Sign() != 0 {
				return nil, fmt.Errorf("account %s: invalid first journal line", account)
			}
		}
		if hashLine(salt, previousHash, line) != line.JournalLineHash {
			return nil, fmt.Errorf("account %s: JournalLineHash mismatch at %s", account, line.JournalLineID)
		}
	}
	if len(lines) == 0 {
		return &AccountState{Balance: "0"}, nil
	}
	latest := lines[0]
	if _, err := decimal(latest.AfterBalance); err != nil {
		return nil, err
	}
	return &AccountState{Sequence: latest.Sequence, Balance: latest.AfterBalance, LastID: latest.JournalLineID, LastHash: latest.JournalLineHash}, nil
}

type AccountState struct {
	Sequence         int64
	Balance          string
	LastID, LastHash string
}

func accountID(ownerType, ownerID, currency string) string {
	return fmt.Sprintf("acct_%s_%s_%s", ownerType, ownerID, currency)
}
func decimal(v string) (*big.Rat, error) {
	r, ok := new(big.Rat).SetString(v)
	if !ok {
		return nil, fmt.Errorf("invalid decimal %q", v)
	}
	return r, nil
}
func formatDecimal(r *big.Rat) string { return r.FloatString(2) }
func hashLine(salt, previous string, line JournalLine) string {
	input := fmt.Sprintf("%s|%s|%s|%s|%d|%s|%s|%s|%s|%s|%s|%s", salt, previous, line.JournalLineID, line.TransactionID, line.Sequence, line.Direction, line.Amount, line.Currency, line.BeforeBalance, line.AfterBalance, value(line.PreviousJournalLineID), line.CreatedAt.UTC().Format(time.RFC3339Nano))
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}
func value(v *string) string {
	if v == nil {
		return "<null>"
	}
	return *v
}

func GenerateJournalLines(tx Transaction, accounts map[string]*AccountState, salt string) ([]JournalLine, error) {
	amount, err := decimal(tx.Amount)
	if err != nil || amount.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if tx.PayeeType == nil || tx.PayeeID == nil {
		return nil, fmt.Errorf("payee is required")
	}
	debitID, creditID := accountID(tx.PayerType, tx.PayerID, tx.Currency), accountID(tx.PayerType, tx.PayerID, tx.Currency)
	lines := make([]JournalLine, 0, 2)
	for i, item := range []struct{ id, direction string }{{debitID, "DEBIT"}, {creditID, "CREDIT"}} {
		state := accounts[item.id]
		if state == nil {
			state = &AccountState{}
			accounts[item.id] = state
		}
		before, _ := decimal(state.Balance)
		if state.Balance == "" {
			before = new(big.Rat)
		}
		after := new(big.Rat).Set(before)
		if item.direction == "DEBIT" {
			after.Sub(after, amount)
		} else {
			after.Add(after, amount)
		}
		id := fmt.Sprintf("jli_%s_%s", tx.TransactionID, item.id)
		var prev *string
		if state.LastID != "" {
			p := state.LastID
			prev = &p
		}
		line := JournalLine{JournalLineID: id, TransactionID: tx.TransactionID, AccountID: item.id, Sequence: state.Sequence + 1, Direction: item.direction, Amount: formatDecimal(amount), Currency: tx.Currency, BeforeBalance: formatDecimal(before), AfterBalance: formatDecimal(after), PreviousJournalLineID: prev, CreatedAt: time.Now().UTC().Truncate(time.Millisecond)}
		previousHash := state.LastHash
		line.JournalLineHash = hashLine(salt, previousHash, line)
		state.Sequence, state.Balance, state.LastID, state.LastHash = line.Sequence, line.AfterBalance, line.JournalLineID, line.JournalLineHash
		lines = append(lines, line)
		_ = i
	}
	return lines, nil
}

var once sync.Once

var config beanq.BeanqConfig

func initConfig() *beanq.BeanqConfig {
	once.Do(func() {
		_, f, _, _ := runtime.Caller(0)
		vp := viper.New()
		vp.AddConfigPath(filepath.Dir(filepath.Dir(f)))
		vp.SetConfigName("env")
		vp.SetConfigType("json")
		_ = vp.ReadInConfig()
		_ = vp.Unmarshal(&config)
	})
	return &config
}
func main() {
	salt := os.Getenv("ACCOUNTING_JOURNAL_SALT")
	if salt == "" {
		salt = "development-only-salt"
	}

	config := initConfig()
	if config.Mongo == nil {
		log.Fatal("mongo configuration is required")
	}
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", config.Mongo.UserName, config.Mongo.Password, config.Mongo.Host, config.Mongo.Port, config.Mongo.Database)
	mongoCtx, cancel := context.WithTimeout(context.Background(), config.Mongo.ConnectTimeOut)
	defer cancel()
	dbClient, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	defer dbClient.Disconnect(context.Background())
	if err := dbClient.Ping(mongoCtx, nil); err != nil {
		log.Fatal(err)
	}
	journalCollection := dbClient.Database(config.Mongo.Database).Collection("JournalLines")
	csm := beanq.New(config)
	_, err = csm.BQ().WithContext(context.Background()).SubscribeSequence("accounting", "transactions", beanq.DefaultHandle{DoHandle: func(ctx context.Context, message *beanq.Message) error {
		var tx Transaction
		if err := json.Unmarshal([]byte(message.Payload), &tx); err != nil {
			return err
		}

		if tx.PayeeType == nil || tx.PayeeID == nil {
			return fmt.Errorf("payee is required")
		}
		accounts := make(map[string]*AccountState)
		for _, account := range []string{accountID(tx.PayerType, tx.PayerID, tx.Currency), accountID(tx.PayerType, tx.PayerID, tx.Currency)} {
			state, err := loadVerifiedAccount(ctx, journalCollection, account, salt)
			if err != nil {
				return fmt.Errorf("journal verification failed: %w", err)
			}
			accounts[account] = state
		}
		lines, err := GenerateJournalLines(tx, accounts, salt)
		if err != nil {
			return err
		}
		for _, line := range lines {
			log.Printf("journal line: %+v", line)
			_, err := journalCollection.InsertOne(ctx, bson.M{
				"JournalLineId":         line.JournalLineID,
				"TransactionId":         line.TransactionID,
				"AccountId":             line.AccountID,
				"Sequence":              line.Sequence,
				"Direction":             line.Direction,
				"Amount":                line.Amount,
				"Currency":              line.Currency,
				"BeforeBalance":         line.BeforeBalance,
				"AfterBalance":          line.AfterBalance,
				"PreviousJournalLineId": line.PreviousJournalLineID,
				"JournalLineHash":       line.JournalLineHash,
				"CreatedAt":             line.CreatedAt})
			if err != nil && !mongo.IsDuplicateKeyError(err) {
				log.Printf("journal line insertion failed: %v", err)
				return err
			}
		}
		return nil
	}, DoCancel: func(context.Context, *beanq.Message) error { return nil }, DoError: func(context.Context, error) {}})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("consumer ready")
	csm.Wait(context.Background())
}
