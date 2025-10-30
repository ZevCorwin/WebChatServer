package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Permission struct {
	ID   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Code string             `bson:"code" json:"code"` // ví dụ: "user.read", "user.lock"
	Desc string             `bson:"desc,omitempty" json:"desc,omitempty"`
}
