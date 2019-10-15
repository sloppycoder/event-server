package eventserver

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
)

type Account struct {
	AccountId         string `json:"accountId,omitempty"`
	Nickname          string `json:"nickname,omitempty"`
	ProdCode          string `json:"prodCode,omitempty"`
	LedgerBalance     string `json:"ledgerBalance"`
	Currency          string `json:"currency,omitempty"`
	Status            string `json:"status,omitempty"`
	StatusLastUpdated int64  `json:"statusLastUpdated,omitempty"`
}

var _db *mongo.Database

func GetTopAccounts(ctx context.Context, count int64) ([]*Account, error) {
	opts := options.Find()
	opts.SetLimit(count)
	opts.SetSort(bson.D{{"accountId", 1}})

	db := db(ctx)
	cur, _ := db.Collection("accounts").Find(ctx, bson.D{}, opts)

	if err := cur.Err(); err != nil {
		return nil, err
	}

	accounts := make([]*Account, 0)
	for cur.Next(ctx) {
		var acc Account
		err := cur.Decode(&acc)
		if err != nil {
			continue
		}

		if acc.LedgerBalance == "" {
			acc.LedgerBalance = "0.00"
		}

		accounts = append(accounts, &acc)
	}

	return accounts, nil
}

func db(ctx context.Context) *mongo.Database {
	if _db != nil {
		return _db
	}

	dbName := os.Getenv("DBNAME")
	if dbName == "" {
		dbName = "dev"
	}

	dbURI := os.Getenv("DBURI")
	if dbURI == "" {
		dbURI = "mongodb://dev:dev@127.0.0.1:27017/dev?authSource=dev&authMechanism=SCRAM-SHA-256"
	}
	log.Printf("DBURI=%s", dbURI)

	clientOpts := options.Client().ApplyURI(dbURI).SetMinPoolSize(10).SetMaxPoolSize(100)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Println(err)
	}

	_db = client.Database(dbName)
	// TODO: when do I disconnect?
	return _db
}
