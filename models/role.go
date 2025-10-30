package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type RoleModel struct {
	ID          primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Name        string               `bson:"name" json:"name"`
	Permissions []primitive.ObjectID `bson:"permissions" json:"permissions"`
}
