package services

import (
	"chat-app-backend/config"
	"chat-app-backend/models"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type UserChannelService struct {
	DB *mongo.Database
}

func NewUserChannelService() *UserChannelService {
	return &UserChannelService{DB: config.DB}
}

func (ucs *UserChannelService) AddUserToChannel(userID, channelID primitive.ObjectID) error {
	collection := ucs.DB.Collection("userChannels")

	// Kiểm tra xem user đã có trong kênh hay chưa
	filter := bson.M{"userID": userID, "channelID": channelID}
	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return fmt.Errorf("failed to count documents: %w", err)
	}

	if count == 0 {
		userChannel := models.UserChannel{
			ID:         primitive.NewObjectID(),
			UserID:     userID,
			ChannelID:  channelID,
			LastActive: time.Now(),
		}
		_, err := collection.InsertOne(context.Background(), userChannel)
		if err != nil {
			return fmt.Errorf("failed to insert user channel: %w", err)
		}
	}
	return nil
}

func (ucs *UserChannelService) UpdateLastActive(userID, channelID primitive.ObjectID) error {
	collection := ucs.DB.Collection("userChannels")
	filter := bson.M{"userID": userID, "channelID": channelID}
	update := bson.M{"$set": bson.M{"lastActive": time.Now()}}

	_, err := collection.UpdateOne(context.Background(), filter, update)
	return err
}
