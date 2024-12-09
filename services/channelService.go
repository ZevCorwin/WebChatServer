package services

import (
	"chat-app-backend/config"
	"chat-app-backend/models"
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"strings"
	"time"
)

type ChannelService struct {
	DB *mongo.Database
}

func NewChannelService() *ChannelService {
	return &ChannelService{DB: config.DB}
}

// CreateChannel Xử lý tạo kênh
func (cs *ChannelService) CreateChannel(name string, channelType models.ChannelType, members []primitive.ObjectID, approvalRequired bool) (*models.Channel, error) {
	// Xác thực ChannelType
	if !channelType.IsValid() {
		return nil, errors.New("invalid channel type")
	}

	// Kiểm tra logic cho Private Channel
	if channelType == models.ChannelTypePrivate && len(members) != 2 {
		return nil, errors.New("private channel requires exactly 2 members")
	}

	// Kiểm tra logic cho Group Channel
	if channelType == models.ChannelTypeGroup && len(members) < 3 {
		return nil, errors.New("group channel requires at least 3 members")
	}

	userService := NewUserService()
	var channelMembers []models.ChannelMember
	var memberNames []string
	var channelAvatar string

	// Duyệt qua danh sách thành viên và tạo thông tin thành viên
	for _, memberID := range members {
		member, err := userService.GetUserByID(memberID.Hex())
		if err != nil {
			return nil, errors.New("failed to fetch user data")
		}
		memberNames = append(memberNames, member.Name)

		channelMembers = append(channelMembers, models.ChannelMember{
			MemberID: memberID,
			Role:     models.RoleMember,
		})

		// Lấy avatar từ đối phương nếu là kênh riêng tư
		if channelType == models.ChannelTypePrivate && len(channelAvatar) == 0 {
			channelAvatar = member.Avatar
		}
	}

	// Tạo kênh theo loại
	channelName := name
	if channelType == models.ChannelTypePrivate {
		// Đặt tên là đối tượng
		channelName = "" // Không lưu cứng tên
	} else if channelType == models.ChannelTypeGroup && name == "" {
		// Tạo nhóm mặc định từ danh sách tên
		channelName = cs.generateGroupChannelName(memberNames)
	}

	// Xử lý ảnh đại diện nhóm
	if channelType == models.ChannelTypeGroup && len(channelAvatar) == 0 {
		channelAvatar = "/uploads/default-group-avatar.png" // Đường dẫn ảnh đại diện mặc định cho nhóm
	}

	// Tạo đối tượng Channel
	channel := &models.Channel{
		ID:          primitive.NewObjectID(),
		ChannelName: channelName,
		ChannelType: channelType,
		Members:     channelMembers,
		Avatar:      channelAvatar,
	}

	// Xử lý logic riêng cho Group Channel
	if channelType == models.ChannelTypeGroup {
		channel.ExtraData = map[string]interface{}{
			"approvalRequired": approvalRequired,
			"createdAt":        time.Now(),
		}
	}

	// Lưu vào database hoặc trả về kết quả
	collection := cs.DB.Collection("channels")
	_, err := collection.InsertOne(context.TODO(), channel)
	if err != nil {
		return nil, err
	}
	return channel, nil
}

// generateGroupChannelName Tạo tên kênh nhóm dựa trên danh sách tên thành viên
func (cs *ChannelService) generateGroupChannelName(memberNames []string) string {
	const maxLength = 30 // Giới hạn độ dài tên
	name := strings.Join(memberNames, ", ")
	if len(name) > maxLength {
		name = name[:maxLength-3] + "..."
	}
	return name
}

// AddMember Thêm thành viên vào kênh
func (cs *ChannelService) AddMember(channel *models.Channel, memberID primitive.ObjectID) error {
	if err := cs.HasPermission(channel, "addMember", memberID); err != nil {
		return err
	}

	for _, member := range channel.Members {
		if member.MemberID == memberID {
			return errors.New("Member already exists")
		}
	}
	channel.Members = append(channel.Members, models.ChannelMember{
		MemberID: memberID,
		Role:     models.RoleMember,
	})
	return nil
}

// RemoveMember Xóa thành viên khỏi kênh
func (cs *ChannelService) RemoveMember(channel *models.Channel, removerID, memberID primitive.ObjectID) error {
	// Kiểm tra quyền của người thực hiện hành động
	if err := cs.HasPermission(channel, "removeMember", removerID); err != nil {
		return err
	}

	// Kiểm tra thành viên xóa có trong danh sách không
	memberFound := false
	for i, member := range channel.Members {
		if member.MemberID == memberID {
			// Xóa thành viên khỏi danh sách
			channel.Members = append(channel.Members[:i], channel.Members[i+1:]...)
			memberFound = true
			break
		}
	}

	// Nếu không tìm thấy thành viên, trả về lỗi
	if !memberFound {
		return errors.New("member not found in the channel")
	}

	// Cập nhật kênh trong database
	if err := cs.UpdateChannel(channel); err != nil {
		return errors.New("failed to update channel after removing member")
	}

	return nil
}

// Thành viên rời khỏi nhóm
func (cs *ChannelService) LeaveChannel(channel *models.Channel, memberID primitive.ObjectID, newLeaderID *primitive.ObjectID) error {
	leaderID, ok := channel.ExtraData["leader"].(primitive.ObjectID)
	if !ok {
		return errors.New("Leader is not set or has invalid type")
	}

	for i, member := range channel.Members {
		if member.MemberID == memberID {
			if memberID == leaderID {
				if newLeaderID == nil {
					return errors.New("Leader must assign a new leader before leaving")
				}
				channel.ExtraData["leader"] = *newLeaderID
			}
			channel.Members = append(channel.Members[:i], channel.Members[i+1:]...)
			return nil
		}
	}
	return errors.New("Member not found in the channel")
}

// Trưởng nhóm giải tán nhóm
func (cs *ChannelService) DissolveChannel(channel *models.Channel, leaderID primitive.ObjectID) error {
	if err := cs.HasPermission(channel, "dissolveChannel", leaderID); err != nil {
		return err
	}
	channel.Members = nil
	channel.BlockMembers = nil
	return nil
}

// Chặn thành viên
func (cs *ChannelService) BlockMember(channel *models.Channel, blockID, memberID primitive.ObjectID) error {
	leaderID, ok := channel.ExtraData["leader"].(primitive.ObjectID)
	if !ok {
		return errors.New("Leader is not set or has invalid type")
	}
	deputyID, _ := channel.ExtraData["deputy"].(primitive.ObjectID)

	if blockID != leaderID && blockID != deputyID {
		return errors.New("Only leader or deputy can block members")
	}

	if blockID == memberID {
		return errors.New("Cannot block yourself")
	}

	for _, blocked := range channel.BlockMembers {
		if blocked == memberID {
			return errors.New("Member is already blocked")
		}
	}

	channel.BlockMembers = append(channel.BlockMembers, memberID)
	return nil
}

// Bỏ chặn thành viên
func (cs *ChannelService) UnblockMember(channel *models.Channel, unblockedID, memberID primitive.ObjectID) error {
	leaderID, ok := channel.ExtraData["leader"].(primitive.ObjectID)
	if !ok {
		return errors.New("Leader is not set or has invalid type")
	}
	deputyID, _ := channel.ExtraData["deputy"].(primitive.ObjectID)

	if unblockedID != leaderID && unblockedID != deputyID {
		return errors.New("Only leader or deputy can unblock members")
	}

	if unblockedID == memberID {
		return errors.New("Cannot unblock yourself")
	}

	for i, blocked := range channel.BlockMembers {
		if blocked == memberID {
			channel.BlockMembers = append(channel.BlockMembers[:i], channel.BlockMembers[i+1:]...)
			return nil
		}
	}
	return errors.New("Member is not in the blocked list")
}

// Trưởng nhóm bật tắt chức năng phê duyệt
func (cs *ChannelService) ToggleApproval(channel *models.Channel, leaderID primitive.ObjectID, enable bool) error {
	currentLeaderID, ok := channel.ExtraData["leader"].(primitive.ObjectID)
	if !ok {
		return errors.New("Current Leader is not set or has invalid type")
	}
	if leaderID != currentLeaderID {
		return errors.New("Only the leader can toggle approval")
	}

	channel.ExtraData["approvalRequired"] = enable
	return nil
}

func (cs *ChannelService) HasPermission(channel *models.Channel, action string, requesterID primitive.ObjectID) error {
	leaderID, _ := channel.ExtraData["leader"].(primitive.ObjectID)
	deputyID, _ := channel.ExtraData["deputy"].(primitive.ObjectID)

	switch action {
	case "removeMember", "blockMember", "unblockMember":
		if requesterID != leaderID && requesterID != deputyID {
			return errors.New("Only leader or deputy can remove members")
		}
	case "dissolveChannel", "toggleApproval":
		if requesterID != leaderID {
			return errors.New("Only the leader can perform this action")
		}
	}
	return nil
}

func (cs *ChannelService) ValidateChannel(channel *models.Channel) error {
	leaderID, ok := channel.ExtraData["leader"].(primitive.ObjectID)
	if !ok {
		return errors.New("Leader is missing in the channel")
	}

	memberSet := make(map[primitive.ObjectID]bool)
	for _, member := range channel.Members {
		memberSet[member.MemberID] = true
	}

	if !memberSet[leaderID] {
		return errors.New("Leader must be a member of the channel")
	}

	if len(memberSet) < 3 && channel.ChannelType == models.ChannelTypeGroup {
		return errors.New("Group must have at least 3 members")
	}

	return nil
}

// Lấy danh sách thành viên
func (cs *ChannelService) ListMembers(channel *models.Channel) []models.ChannelMember {
	return channel.Members
}

// Xác minh vai trò thành viên
func (cs *ChannelService) CheckMemberRole(channel *models.Channel, memberID primitive.ObjectID) (models.MemberRole, error) {
	leaderID, _ := channel.ExtraData["leader"].(primitive.ObjectID)
	deputyID, _ := channel.ExtraData["deputy"].(primitive.ObjectID)

	switch {
	case memberID == leaderID:
		return models.RoleLeader, nil
	case memberID == deputyID:
		return models.RoleDeputy, nil
	default:
		return models.RoleMember, nil
	}
}

// Lấy thông tin kênh
func (cs *ChannelService) GetChannel(channelId primitive.ObjectID) (*models.Channel, error) {
	collection := cs.DB.Collection("channels")
	var channel models.Channel
	err := collection.FindOne(context.TODO(), bson.M{"_id": channelId}).Decode(&channel)
	if err != nil {
		return nil, errors.New("Channel not found")
	}
	return &channel, nil
}

// Kiểm tra xem thành viên có trong kênh hay không
func (cs *ChannelService) IsMember(channel *models.Channel, memberID primitive.ObjectID) bool {
	for _, member := range channel.Members {
		if member.MemberID == memberID {
			return true
		}
	}
	return false
}

// Cập nhật dữ liệu kênh
func (cs *ChannelService) UpdateChannel(channel *models.Channel) error {
	_, err := cs.DB.Collection("channels").
		UpdateOne(
			context.TODO(),
			bson.M{"_id": channel.ID},
			bson.M{"$set": channel},
		)
	return err
}

// SearchChannels Tìm kiếm kênh theo tên
func (cs *ChannelService) SearchChannels(keyword string) ([]models.Channel, error) {
	collection := cs.DB.Collection("channels")
	var channels []models.Channel

	// Tạo bộ lọc tìm kiếm theo tên
	filter := bson.M{
		"channelName": bson.M{
			"$regex":   keyword,
			"$options": "i",
		},
	}

	// Thực hiện truy vấn
	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	ctx := context.TODO()
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			log.Println("Lỗi khi đóng cursor:", err)
		}
	}(cursor, ctx)

	for cursor.Next(ctx) {
		var channel models.Channel
		if err := cursor.Decode(&channel); err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return channels, nil
}

// Lấy danh sách tất cả các kênh mà người dùng đã tham gia
func (cs *ChannelService) GetChannelsByUserID(userID primitive.ObjectID) ([]models.Channel, error) {
	// Lấy collection liên quan
	userChannelCollection := cs.DB.Collection("userChannels")
	channelCollection := cs.DB.Collection("channels")

	// Lấy danh sách các ChannelID từ bảng UserChannel
	cursor, err := userChannelCollection.Find(context.TODO(), bson.M{"userID": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	// Trích xuất danh sách ChannelID
	var userChannels []models.UserChannel
	if err := cursor.All(context.TODO(), &userChannels); err != nil {
		return nil, err
	}

	if len(userChannels) == 0 {
		return []models.Channel{}, nil // Không có kênh nào
	}

	// Lấy danh sách các ChannelID
	var channelIDs []primitive.ObjectID
	for _, userChannel := range userChannels {
		channelIDs = append(channelIDs, userChannel.ChannelID)
	}

	// Truy vấn danh sách kênh từ bảng Channels
	cursor, err = channelCollection.Find(context.TODO(), bson.M{"_id": bson.M{"$in": channelIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	// Lưu danh sách kênh
	var channels []models.Channel
	if err := cursor.All(context.TODO(), &channels); err != nil {
		return nil, err
	}

	return channels, nil
}
