package models

import (
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

// Định nghĩa Enum cho các trạng thái
// Gender là kiểu dữ liệu enum cho giới tính
type Gender int

// Role là kiểu dữ liệu enum cho vai trò người dùng
type Role string

// Status là kiểu dữ liệu enum cho trạng thái người dùng
type Status string

// MaritalStatus là kiểu dữ liệu enum cho trạng thái hôn nhân
type MaritalStatus string

// BlockType là kiểu dữ liệu enum cho loại chặn người dùng
type BlockType string

const (
	GenderMale Gender = iota
	GenderFemale
	GenderOther

	RoleUser      Role = "Người dùng"
	RoleTeamLead  Role = "Trưởng nhóm"
	RoleSubLeader Role = "Phó nhóm"

	StatusOnline  Status = "Online"
	StatusOffline Status = "Offline"

	MaritalSingle  MaritalStatus = "Độc thân"
	MaritalDating  MaritalStatus = "Đang hẹn hò"
	MaritalMarried MaritalStatus = "Đã kết hôn"

	BlockMessage BlockType = "Chặn tin nhắn"
	BlockCall    BlockType = "Chặn gọi điện"
	BlockAll     BlockType = "Chặn tất cả"
)

func (g Gender) String() string {
	return [...]string{"Nam", "Nữ", "Khác"}[g]
}

// User là cấu trúc dữ liệu đại diện cho người dùng
type User struct {
	ID                 primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name               string             `json:"name" bson:"name"`
	Email              string             `json:"email" bson:"email"`
	Phone              string             `json:"phone" bson:"phone"`
	Password           string             `json:"password" bson:"password"`
	Address            string             `json:"address" bson:"address"`
	BirthDate          string             `json:"birthDate" bson:"birthDate"`
	Gender             Gender             `json:"gender" bson:"gender"`
	Avatar             string             `json:"avatar" bson:"avatar"`
	CoverPhoto         string             `json:"coverPhoto" bson:"coverPhoto"`
	Role               Role               `json:"role" bson:"role"`
	Status             Status             `json:"status" bson:"status"`
	LastOnlineTime     time.Time          `json:"lastOnlineTime" bson:"lastOnlineTime"`
	MaritalStatus      MaritalStatus      `json:"maritalStatus" bson:"maritalStatus"`
	ConnectedWith      *string            `json:"connectedWith" bson:"connectedWith,omitempty"` // Nullable
	BlockedByUsers     []string           `json:"blockedByUsers" bson:"blockedByUsers"`
	BlockedUsers       []string           `json:"blockedUsers" bson:"blockedUsers"`
	BlockType          BlockType          `json:"blockType" bson:"blockType"`
	AccountCreatedDate time.Time          `json:"accountCreatedDate" bson:"accountCreatedDate"`
	AdminLocked        bool               `json:"adminLocked" bson:"adminLocked"`
}

var db *mongo.Database
var Validate = validator.New()
