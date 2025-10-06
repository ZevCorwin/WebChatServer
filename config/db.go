package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var DB *mongo.Database
var mongoClient *mongo.Client

func InitDB() {
	DB = ConnectDB()
}

func ConnectDB() *mongo.Database {
	LoadEnv()

	uri := os.Getenv("MONGODB_URI")
	dbName := os.Getenv("DB_NAME")

	// Fallback cho dev local nếu không có SRV
	if uri == "" {
		host := os.Getenv("DB_HOST")
		port := os.Getenv("DB_PORT")
		if host == "" || port == "" || dbName == "" {
			log.Fatal("Thiếu MONGODB_URI hoặc (DB_HOST, DB_PORT, DB_NAME)")
		}
		uri = fmt.Sprintf("mongodb://%s:%s/%s", host, port, dbName)
	}

	if dbName == "" {
		// Nếu không set DB_NAME, chọn mặc định
		dbName = "chatapp"
	}

	clientOpts := options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(10 * time.Second)

	client, err := mongo.Connect(context.Background(), clientOpts)
	if err != nil {
		log.Fatalf("Không thể tạo client MongoDB: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Ping primary (v1.6 dùng readpref)
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalf("Ping MongoDB thất bại: %v", err)
	}

	log.Printf("[Mongo] ✅ Kết nối OK: %s", uri)

	db := client.Database(dbName)
	mongoClient = client

	// TTL index cho OTP
	otpCollection := db.Collection("otps")
	indexModel := mongo.IndexModel{
		Keys:    bson.M{"expires_at": 1},
		Options: options.Index().SetExpireAfterSeconds(0),
	}
	if _, err := otpCollection.Indexes().CreateOne(context.Background(), indexModel); err != nil {
		log.Printf("Không thể tạo TTL index cho OTP: %v", err)
	}

	// Index messages / chathistory
	{
		_, err := db.Collection("messages").Indexes().CreateMany(context.Background(), []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "channelID", Value: 1}, {Key: "timestamp", Value: -1}},
				Options: options.Index().SetName("byChannel_time"),
			},
			{
				Keys:    bson.D{{Key: "replyTo", Value: 1}},
				Options: options.Index().SetName("byReplyTo"),
			},
		})
		if err != nil {
			log.Printf("Không thể tạo index messages: %v", err)
		}

		_, err = db.Collection("chathistory").Indexes().CreateOne(context.Background(), mongo.IndexModel{
			Keys:    bson.D{{Key: "channelID", Value: 1}},
			Options: options.Index().SetName("byChannel"),
		})
		if err != nil {
			log.Printf("Không thể tạo index chathistory: %v", err)
		}
	}

	if err := ensureMessageIndexes(db); err != nil {
		log.Printf("Không thể tạo index cho messages: %v", err)
	}

	return db
}

func ensureMessageIndexes(db *mongo.Database) error {
	coll := db.Collection("messages")

	_, err := coll.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "channelID", Value: 1}, {Key: "timestamp", Value: 1}},
		Options: options.Index().SetName("channelID_timestamp_idx"),
	})
	if err != nil {
		return err
	}

	_, err = coll.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "hiddenBy", Value: 1}},
		Options: options.Index().SetName("hiddenBy_idx"),
	})
	return err
}

func CloseDB() {
	if mongoClient != nil {
		_ = mongoClient.Disconnect(context.Background())
	}
}
