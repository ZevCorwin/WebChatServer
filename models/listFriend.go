package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type FriendType string

const (
	FriendTypePending FriendType = "Pending"
	FriendTypeFriend  FriendType = "Friend"
)

type ListFriends struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID          primitive.ObjectID `json:"userID" bson:"userID"`
	FriendID        primitive.ObjectID `json:"friendID" bson:"friendID"`
	FriendType      FriendType         `json:"friendType" bson:"friendType"`
	RequestSentData *time.Time         `json:"requestSentData" bson:"requestSentData,omitempty"`
	ConfirmData     *time.Time         `json:"confirmData" bson:"confirmData,omitempty"`
}
