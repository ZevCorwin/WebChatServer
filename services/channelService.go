package services

import (
	"chat-app-backend/config"
	"chat-app-backend/models"
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

type ChannelService struct {
	DB                 *mongo.Database
	UserChannelService *UserChannelService
	ChatHistoryService *ChatHistoryService
}

func NewChannelService() *ChannelService {
	return &ChannelService{
		DB:                 config.DB,
		UserChannelService: NewUserChannelService(),
		ChatHistoryService: NewChatHistoryService(),
	}
}

// CreateChannel Xử lý tạo kênh
func (cs *ChannelService) CreateChannel(userID primitive.ObjectID, name string, channelType models.ChannelType, members []primitive.ObjectID, approvalRequired bool) (*models.Channel, error) {
	log.Printf("[CreateChannel]CreateChannel called with userID: %s, name: %s, type: %s, members: %v, approvalRequired: %t", userID.Hex(), name, channelType, members, approvalRequired)
	// Xác thực ChannelType
	if !channelType.IsValid() {
		log.Printf("[CreateChannel]Error: invalid channel type")
		return nil, errors.New("[CreateChannel]invalid channel type")
	}

	// Kiểm tra logic cho Private Channel
	if channelType == models.ChannelTypePrivate && len(members) != 2 {
		log.Printf("[CreateChannel]Error: private channel requires exactly 2 members")
		return nil, errors.New("[CreateChannel]private channel requires exactly 2 members")
	}

	// Kiểm tra logic cho Group Channel
	if channelType == models.ChannelTypeGroup && len(members) < 3 {
		log.Printf("[CreateChannel]Error: group channel requires at least 3 members")
		return nil, errors.New("[CreateChannel]group channel requires at least 3 members")
	}

	//Kiểm tra unique members
	memberMap := make(map[primitive.ObjectID]bool)
	for _, memberID := range members {
		if memberMap[memberID] {
			log.Printf("[CreateChannel]Error: duplicate member ID: %s", memberID.Hex())
			return nil, errors.New("[CreateChannel]Duplicate member ID")
		}
		memberMap[memberID] = true
	}
	log.Printf("[CreateChannel]Passed unique members check")

	// Kiểm tra userID có trong members
	userIDFound := false
	for _, memberID := range members {
		if memberID == userID {
			userIDFound = true
			break
		}
	}
	if userIDFound == false {
		log.Printf("[CreateChannel]Error: creator must be a member of the channel")
		return nil, errors.New("[CreateChannel]creator must be a member of the channel")
	}
	log.Printf("[CreateChannel]Passed creator in members check")

	userService := NewUserService()
	var channelMembers []models.ChannelMember
	var memberNames []string
	var channelAvatar string

	// Duyệt qua danh sách thành viên và tạo thông tin thành viên
	leaderAssigned := false
	for _, memberID := range members {
		member, err := userService.GetUserByID(memberID.Hex())
		if err != nil {
			log.Printf("[CreateChannel]Error: failed to fetch user data for memberID: %s, error: %v", memberID.Hex(), err)
			return nil, fmt.Errorf("failed to fetch user data: %w", err)
		}
		// Kiểm tra và làm sạch
		if utf8.ValidString(member.Name) {
			memberNames = append(memberNames, member.Name)
		}
		role := models.RoleMember
		if memberID == userID && channelType == models.ChannelTypeGroup && !leaderAssigned {
			role = models.RoleLeader
			leaderAssigned = true
		}
		channelMembers = append(channelMembers, models.ChannelMember{
			MemberID: memberID,
			Role:     role,
		})

		// Lấy avatar từ đối phương nếu là kênh riêng tư
		if channelType == models.ChannelTypePrivate && len(channelAvatar) == 0 {
			channelAvatar = member.Avatar
		}
	}
	log.Printf("[CreateChannel]Passed fetching user data check")

	// Check if leader is assigned for group
	if channelType == models.ChannelTypeGroup && !leaderAssigned {
		log.Printf("[CreateChannel]Error: no leader assigned for group channel")
		return nil, errors.New("[CreateChannel]group channel requires a leader")
	}
	log.Printf("[CreateChannel]Passed leader check")

	// Tạo tên kênh theo loại
	channelName := name
	if channelType == models.ChannelTypePrivate {
		// Đặt tên mặc định là đối tượng là đối tượng
		channelName = "" // Không lưu cứng tên
	} else if channelType == models.ChannelTypeGroup && name == "" {
		// Tạo nhóm mặc định từ danh sách tên
		channelName = cs.generateGroupChannelName(memberNames)
	}
	channelName = truncateUTF8(channelName, 50) //Giới hạn an toàn
	// Xử lý ảnh đại diện nhóm
	if channelType == models.ChannelTypeGroup && len(channelAvatar) == 0 {
		defaultChannelAvatar := os.Getenv("DEFAULT_CHANNEL_AVATAR_URL")
		if defaultChannelAvatar == "" {
			// fallback về same default avatar hoặc local path
			defaultChannelAvatar = "/uploads/deadlineDi.jpg"
		}
		channelAvatar = defaultChannelAvatar
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
			"leader":           userID,
		}
	}

	// Lưu vào database hoặc trả về kết quả
	collection := cs.DB.Collection("channels")
	_, err := collection.InsertOne(context.TODO(), channel)
	if err != nil {
		log.Printf("[CreateChannel]Error inserting channel: %v", err)
		return nil, err
	}
	log.Printf("[CreateChannel]Channel inserted successfully")

	// Thêm tất cả thành viên vào userChannel
	for _, memberID := range members {
		err := cs.UserChannelService.AddUserToChannel(memberID, channel.ID)
		if err != nil {
			log.Printf("[CreateChannel]Error adding user to channel: %v", err)
			return nil, fmt.Errorf("[CreateChannel]failed to add user to channel: %w", err)
		}
	}
	log.Printf("[CreateChannel]All members added to userChannel")

	return channel, nil
}

// truncateUTF8 cắt chuỗi UTF-8 an toàn
func truncateUTF8(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLength {
		return s
	}
	return string(runes[:maxLength-3]) + "..."
}

// generateGroupChannelName Tạo tên kênh nhóm dựa trên danh sách tên thành viên
func (cs *ChannelService) generateGroupChannelName(memberNames []string) string {
	const maxLength = 20 // Giới hạn độ dài tên
	// Lọc tên rỗng và làm sạch
	var vailName []string
	for _, groupName := range memberNames {
		if groupName != "" && utf8.ValidString(groupName) {
			vailName = append(vailName, groupName)
		}
	}
	if len(vailName) == 0 {
		return "Nhóm " + time.Now().Format("2006-01-02 15:04:05")
	}
	groupName := strings.Join(vailName, ", ")
	return truncateUTF8(groupName, maxLength)
}

// AddMember Thêm thành viên vào kênh
func (cs *ChannelService) AddMember(channel *models.Channel, memberID primitive.ObjectID) error {
	// Kiểm tra quyền
	if err := cs.HasPermission(channel, "addMember", memberID); err != nil {
		return err
	}

	// Kiểm tra trùng
	for _, member := range channel.Members {
		if member.MemberID == memberID {
			return errors.New("Member already exists")
		}
	}

	// Kiểm tra xem có trong danh sách bị chặn không
	for _, blocked := range channel.BlockMembers {
		if blocked == memberID {
			return errors.New("This member is blocked and cannot be re-added")
		}
	}

	// Cập nhật DB
	collection := cs.DB.Collection("channels")
	filter := bson.M{"_id": channel.ID}
	update := bson.M{"$push": bson.M{"members": models.ChannelMember{
		MemberID: memberID,
		Role:     models.RoleMember,
	}}}

	_, err := collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return err
	}

	return nil
}

// RemoveMember Xóa thành viên khỏi kênh
func (cs *ChannelService) RemoveMember(channel *models.Channel, removerID, memberID primitive.ObjectID) error {
	// 1. Kiểm tra quyền
	if err := cs.HasPermission(channel, "removeMember", removerID); err != nil {
		return err
	}

	// 2. Tìm và xóa
	idx := -1
	for i, member := range channel.Members {
		if member.MemberID == memberID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errors.New("member not found in the channel")
	}

	channel.Members = append(channel.Members[:idx], channel.Members[idx+1:]...)

	// 3. Update DB
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

// BlockMember chặn thành viên (đồng thời xóa khỏi nhóm)
func (cs *ChannelService) BlockMember(channel *models.Channel, blockerID, memberID primitive.ObjectID) error {
	var blockerRole, targetRole models.MemberRole
	idx := -1

	for i, m := range channel.Members {
		if m.MemberID == blockerID {
			blockerRole = m.Role
		}
		if m.MemberID == memberID {
			targetRole = m.Role
			idx = i
		}
	}

	if blockerRole != models.RoleLeader && blockerRole != models.RoleDeputy {
		return errors.New("Only leader or deputy can block members")
	}

	if blockerID == memberID {
		return errors.New("Cannot block yourself")
	}

	// Không cho chặn leader hoặc deputy
	if targetRole == models.RoleLeader || targetRole == models.RoleDeputy {
		return errors.New("Cannot block leader or deputy")
	}

	// Nếu chưa có trong BlockMembers thì thêm vào
	for _, blocked := range channel.BlockMembers {
		if blocked == memberID {
			return errors.New("Member is already blocked")
		}
	}
	channel.BlockMembers = append(channel.BlockMembers, memberID)

	// Xóa khỏi Members nếu có trong danh sách
	if idx != -1 {
		channel.Members = append(channel.Members[:idx], channel.Members[idx+1:]...)
	}

	// Cập nhật DB
	if err := cs.UpdateChannel(channel); err != nil {
		return errors.New("failed to update channel after blocking member")
	}

	return nil
}

// UnblockMember bỏ chặn thành viên
func (cs *ChannelService) UnblockMember(channel *models.Channel, unblockerID, memberID primitive.ObjectID) error {
	// 1) Lấy role người thực hiện từ channel.Members (đồng bộ với BlockMember)
	var unblockerRole models.MemberRole
	for _, m := range channel.Members {
		if m.MemberID == unblockerID {
			unblockerRole = m.Role
			break
		}
	}
	// 2) Chỉ Leader/Deputy được bỏ chặn
	if unblockerRole != models.RoleLeader && unblockerRole != models.RoleDeputy {
		return errors.New("Only leader or deputy can unblock members")
	}
	// 3) Không tự bỏ chặn chính mình (tuỳ chính sách, giữ giống BlockMember)
	if unblockerID == memberID {
		return errors.New("Cannot unblock yourself")
	}
	// 4) Tìm trong danh sách BlockMembers
	idx := -1
	for i, blocked := range channel.BlockMembers {
		if blocked == memberID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errors.New("Member is not in the blocked list")
	}
	// 5) Xoá khỏi BlockMembers
	channel.BlockMembers = append(channel.BlockMembers[:idx], channel.BlockMembers[idx+1:]...)

	// 6) Cập nhật DB
	if err := cs.UpdateChannel(channel); err != nil {
		return errors.New("failed to update channel after unblocking member")
	}
	return nil
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
	requesterRole := cs.roleOf(channel, requesterID)

	switch action {
	case "removeMember", "blockMember", "unblockMember":
		if requesterRole != models.RoleLeader && requesterRole != models.RoleDeputy {
			return errors.New("Only leader or deputy can perform this action")
		}
	case "dissolveChannel", "toggleApproval":
		if requesterRole != models.RoleLeader {
			return errors.New("Only the leader can perform this action")
		}
	}
	return nil
}

func (cs *ChannelService) roleOf(channel *models.Channel, userID primitive.ObjectID) models.MemberRole {
	for _, m := range channel.Members {
		if m.MemberID == userID {
			return m.Role
		}
	}
	return models.RoleMember // nếu không tìm thấy thì mặc định Member
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

func (cs *ChannelService) ListBlockedMembers(channel *models.Channel) ([]map[string]interface{}, error) {
	var blocked []map[string]interface{}
	userColl := cs.DB.Collection("users")

	log.Printf("[ListBlockedMembers] ChannelID=%s, BlockMembers=%v", channel.ID.Hex(), channel.BlockMembers)

	for _, memberID := range channel.BlockMembers {
		log.Printf("[ListBlockedMembers] Đang xử lý memberID=%s", memberID.Hex())

		var user struct {
			ID     primitive.ObjectID `bson:"_id"`
			Name   string             `bson:"name"`
			Avatar string             `bson:"avatar"`
			Phone  string             `bson:"phone"`
		}

		err := userColl.FindOne(context.TODO(), bson.M{"_id": memberID}).Decode(&user)
		if err != nil {
			log.Printf("[ListBlockedMembers] ❌ Không tìm thấy user với ID=%s, error=%v", memberID.Hex(), err)
			blocked = append(blocked, map[string]interface{}{
				"memberId": memberID.Hex(),
				"name":     "Người dùng không tồn tại",
				"avatar":   "/default-avatar.png",
				"phone":    "",
			})
			continue
		}

		log.Printf("[ListBlockedMembers] ✅ Tìm thấy user: %+v", user)

		blocked = append(blocked, map[string]interface{}{
			"memberId": user.ID.Hex(),
			"name":     user.Name,
			"avatar":   "http://localhost:8080" + user.Avatar,
			"phone":    user.Phone,
		})
	}

	log.Printf("[ListBlockedMembers] Hoàn tất, tổng số=%d", len(blocked))
	return blocked, nil
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
func (cs *ChannelService) GetChannel(channelID primitive.ObjectID) (*models.Channel, error) {
	collection := cs.DB.Collection("channels")
	var channel models.Channel
	err := collection.FindOne(context.TODO(), bson.M{"_id": channelID}).Decode(&channel)
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

func (cs *ChannelService) FindOrCreatePrivateChannel(member1 string, member2 string) (map[string]interface{}, error) {
	id1, err := primitive.ObjectIDFromHex(member1)
	if err != nil {
		return nil, fmt.Errorf("Invalid member1 ID")
	}
	id2, err := primitive.ObjectIDFromHex(member2)
	if err != nil {
		return nil, fmt.Errorf("Invalid member2 ID")
	}

	collection := cs.DB.Collection("channels")
	filter := bson.M{
		"channelType": models.ChannelTypePrivate,
		"members.memberID": bson.M{
			"$all": []primitive.ObjectID{id1, id2},
		},
		"$expr": bson.M{
			"$eq": []interface{}{bson.M{"$size": "$members"}, 2},
		},
	}

	var channel models.Channel
	err = collection.FindOne(context.TODO(), filter).Decode(&channel)

	if err == mongo.ErrNoDocuments {
		createdChannel, err := cs.CreateChannel(id1, "Private channel", models.ChannelTypePrivate, []primitive.ObjectID{id1, id2}, false)
		if err != nil {
			return nil, err
		}
		channel = *createdChannel
	} else if err != nil {
		return nil, err
	}

	chatItems, err := cs.ChatHistoryService.GetChatHistoryByUserID(id1)
	if err != nil {
		return nil, err
	}

	for _, item := range chatItems {
		if item["channelID"] == channel.ID {
			return item, nil
		}
	}

	return map[string]interface{}{
		"channelID":   channel.ID,
		"channelName": channel.ChannelName,
		"channelType": channel.ChannelType,
	}, nil
}
