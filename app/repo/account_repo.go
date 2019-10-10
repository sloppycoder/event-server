package repo

import (
	"context"
	"grpc-account-svc/api"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/copier"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/grpclog"
)

type Balance struct {
	Amount      float64
	Type        string
	CreditFlag  bool
	LastUpdated string `bson:"last_updated, omitempty"`
}

type Account struct {
	Id                primitive.ObjectID `bson:"_id, omitempty"`
	AccountId         string
	Nickname          string
	ProdCode          string
	ProdName          string
	Currency          string
	Servicer          string
	Status            string
	StatusLastUpdated string `bson:"status_last_updated, omitempty"`
	Balances          []*Balance
}

var _db *mongo.Database
var (
	getAccountSuccess = promauto.NewCounter(prometheus.CounterOpts{
		Name: "get_account_success_total",
		Help: "total accounts successfully fetched from database",
	})

	getAccountFailure = promauto.NewCounter(prometheus.CounterOpts{
		Name: "get_account_failure_total",
		Help: "total failed attempts for fetching account from database",
	})

	getAccountHisto = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "get_account_success_histogram",
		Help: "histogram for successful fetch account from database",
		// used to store microseconds
		Buckets: []float64{100, 250, 500, 1_000, 2_500, 5_000, 10_000, 25_000, 50_000, 100_000, 2_500_000},
	})
)

func GetAccountById(ctx context.Context, id string) (*api.Account, error) {
	db := db(ctx)
	start := time.Now()
	cur := db.Collection("accounts").FindOne(ctx, bson.M{"accountId": id})

	if err := cur.Err(); err != nil {
		getAccountFailure.Inc()
		return nil, err
	}

	var acc Account
	err := cur.Decode(&acc)
	if err != nil {
		getAccountFailure.Inc()
		return nil, err
	}

	account, err := mapAccount(&acc)
	if err != nil {
		grpclog.Info("Mapping account got error %v", err)
	}

	getAccountSuccess.Inc()
	elapsed := time.Since(start)
	getAccountHisto.Observe(float64(elapsed / time.Microsecond))

	return account, nil
}

func GetTopAccounts(ctx context.Context, count int64) ([]*api.Account, error) {
	opts := options.Find()
	opts.SetLimit(count)
	opts.SetSort(bson.D{{"accountId", 1}})

	db := db(ctx)
	cur, _ := db.Collection("accounts").Find(ctx, bson.D{}, opts)

	if err := cur.Err(); err != nil {
		return nil, err
	}

	accounts := make([]*api.Account, 0)
	for cur.Next(ctx) {
		var acc Account
		err := cur.Decode(&acc)
		if err != nil {
			continue
		}

		tmp, err := mapAccount(&acc)
		if err != nil {
			continue
		}
		accounts = append(accounts, tmp)
	}

	return accounts, nil
}

func mapAccount(acc *Account) (*api.Account, error) {
	var account api.Account

	_ = copier.Copy(&account, &acc)
	account.Id = acc.Id.String()[10:34] // strip off the ObjectId("...")

	var err error
	t1, _ := time.Parse(time.RFC3339, acc.StatusLastUpdated)
	account.StatusLastUpdated, err = ptypes.TimestampProto(t1)

	for i, bal := range acc.Balances {
		t2, _ := time.Parse(time.RFC3339, bal.LastUpdated)
		pbtimestamp, _ := ptypes.TimestampProto(t2)
		account.Balances[i].LastUpdated = pbtimestamp
	}

	return &account, err
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
	grpclog.Infof("DBURI=%s", dbURI)

	clientOpts := options.Client().ApplyURI(dbURI).SetMinPoolSize(10).SetMaxPoolSize(100)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		grpclog.Error(err)
	}

	_db = client.Database(dbName)
	// TODO: when do I disconnect?
	return _db
}
