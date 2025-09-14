package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type ChannelType string

type MemberRole string

const (
	ChannelTypePrivate ChannelType = "Private"
	ChannelTypeGroup   ChannelType = "Group"

	RoleLeader MemberRole = "Leader"
	RoleDeputy MemberRole = "Deputy"
	RoleMember MemberRole = "Member"
)

// Xác thực ChannelType
func (ct ChannelType) IsValid() bool {
	switch ct {
	case ChannelTypePrivate, ChannelTypeGroup:
		return true
	default:
		return false
	}
}

// Xác thực MemberRole
func (mr MemberRole) IsValid() bool {
	switch mr {
	case RoleLeader, RoleDeputy, RoleMember:
		return true
	default:
		return false
	}
}

type ChannelMember struct {
	MemberID primitive.ObjectID `bson:"memberID"`
	Role     MemberRole         `bson:"role"`
}

type Channel struct {
	ID           primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	ChannelName  string                 `json:"channelName" bson:"channelName"`
	ChannelType  ChannelType            `json:"channelType" bson:"channelType"`
	Members      []ChannelMember        `json:"members" bson:"members"`
	BlockMembers []primitive.ObjectID   `json:"blockMembers" bson:"blockMembers"`
	ExtraData    map[string]interface{} `json:"extraData" bson:"extraData,omitempty"`
	Avatar       string                 `json:"avatar" bson:"avatar"`
}
