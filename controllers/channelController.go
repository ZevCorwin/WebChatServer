package controllers

import (
	"chat-app-backend/models"
	"chat-app-backend/services"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChannelController struct {
	ChannelService *services.ChannelService
}

func NewChannelController(service *services.ChannelService) *ChannelController {
	return &ChannelController{ChannelService: service}
}

// Tạo kênh
func (cc *ChannelController) CreateChannelHandler(ctx *gin.Context) {
	var req struct {
		Name             string               `json:"name"`
		Type             models.ChannelType   `json:"type"`
		Members          []primitive.ObjectID `json:"members"`
		ApprovalRequired bool                 `json:"approvalRequired"`
	}

	// Decode JSON request body
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Gọi service để tạo kênh
	channel, err := cc.ChannelService.CreateChannel(req.Name, req.Type, req.Members, req.ApprovalRequired)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, channel)
}

// Thêm thành viên
func (cc *ChannelController) AddMemberHandler(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelID")
	memberIdStr := ctx.Param("memberID")

	// Convert ChannelID from string to ObjectID
	channelID, err := primitive.ObjectIDFromHex(channelIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel id"})
		return
	}

	// Convert MemberID from string to ObjectID
	memberID, err := primitive.ObjectIDFromHex(memberIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}

	// Lấy thông tin kênh
	channel, err := cc.ChannelService.GetChannel(channelID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	// Thêm thành viên
	if err := cc.ChannelService.AddMember(channel, memberID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"channel": channel})
}

// Xóa thành viên khỏi kênh
func (cc *ChannelController) RemoveMemberHandler(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelID")
	memberIdStr := ctx.Param("memberID")
	removerIdStr := ctx.Param("removerID")

	channelID, err := primitive.ObjectIDFromHex(channelIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel id"})
		return
	}
	memberID, err := primitive.ObjectIDFromHex(memberIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}
	removerID, err := primitive.ObjectIDFromHex(removerIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid remover ID"})
		return
	}

	channel, err := cc.ChannelService.GetChannel(channelID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	// Xóa thành viên
	if err := cc.ChannelService.RemoveMember(channel, removerID, memberID); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Cập nhật DB
	if err := cc.ChannelService.UpdateChannel(channel); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update channel"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}

// Lấy danh sách thành viên
func (cc *ChannelController) ListMembersHandler(ctx *gin.Context) {
	channelID := ctx.Query("channelId")
	id, err := primitive.ObjectIDFromHex(channelID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel id"})
		return
	}

	channel, err := cc.ChannelService.GetChannel(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	members := cc.ChannelService.ListMembers(channel)

	ctx.JSON(http.StatusOK, gin.H{"members": members})
}

// Bật/tắt phê duyệt
func (cc *ChannelController) ToggleApprovalHandler(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelID")

	var req struct {
		LeaderID string `json:"leaderId"`
		Enable   bool   `json:"enable"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	channelID, err := primitive.ObjectIDFromHex(channelIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel id"})
		return
	}
	leaderID, err := primitive.ObjectIDFromHex(req.LeaderID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IDs provided"})
		return
	}

	channel, err := cc.ChannelService.GetChannel(channelID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	// Cập nhật trạng thái phê duyệt
	if err := cc.ChannelService.ToggleApproval(channel, leaderID, req.Enable); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Cập nhật DB
	if err := cc.ChannelService.UpdateChannel(channel); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update channel"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Approval setting updated successfully"})
}

// Thành viên rời khỏi nhóm
func (cc *ChannelController) LeaveChannelHandler(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelID")
	memberIdStr := ctx.Param("memberID")
	newLeaderIdStr := ctx.Param("newLeaderID")

	channelID, err := primitive.ObjectIDFromHex(channelIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel id"})
		return
	}

	memberID, err := primitive.ObjectIDFromHex(memberIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}

	var newLeaderId *primitive.ObjectID
	if newLeaderIdStr != "" {
		id, err := primitive.ObjectIDFromHex(newLeaderIdStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid new leader ID"})
			return
		}
		newLeaderId = &id
	}

	channel, err := cc.ChannelService.GetChannel(channelID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	if err := cc.ChannelService.LeaveChannel(channel, memberID, newLeaderId); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Leave channel successfully"})
}

// Trưởng nhóm giải tán nhóm
func (cc *ChannelController) DissolveChannelHandler(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelID")
	leaderIdStr := ctx.Param("leaderID")

	channelID, err := primitive.ObjectIDFromHex(channelIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel id"})
		return
	}

	leaderID, err := primitive.ObjectIDFromHex(leaderIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid leader ID"})
		return
	}

	channel, err := cc.ChannelService.GetChannel(channelID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	if err := cc.ChannelService.DissolveChannel(channel, leaderID); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Dissolve channel successfully"})
}

// Chặn thành viên
func (cc *ChannelController) BlockMemberHandler(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelID")
	blockerIdStr := ctx.Param("blockID")
	memberIdStr := ctx.Param("memberID")

	channelID, err := primitive.ObjectIDFromHex(channelIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel id"})
		return
	}

	blockerID, err := primitive.ObjectIDFromHex(blockerIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blocker ID"})
		return
	}

	memberID, err := primitive.ObjectIDFromHex(memberIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}

	channel, err := cc.ChannelService.GetChannel(channelID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	if err := cc.ChannelService.BlockMember(channel, blockerID, memberID); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Block member successfully"})
}

// Bỏ chăn thành viên
func (cc *ChannelController) UnblockMemberHandler(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelID")
	unblockerIdStr := ctx.Param("unblockID")
	memberIdStr := ctx.Param("memberID")

	channelID, err := primitive.ObjectIDFromHex(channelIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel id"})
		return
	}

	unblockerID, err := primitive.ObjectIDFromHex(unblockerIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unblocker ID"})
		return
	}

	memberID, err := primitive.ObjectIDFromHex(memberIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}

	channel, err := cc.ChannelService.GetChannel(channelID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	if err := cc.ChannelService.UnblockMember(channel, unblockerID, memberID); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Unblock member successfully"})
}

// SearchChannelsHandler Xử lý tìm kiếm kênh theo tên
func (cc *ChannelController) SearchChannelsHandler(ctx *gin.Context) {
	keyword := ctx.Query("keyword") // Lấy từ khóa từ query parameters
	if keyword == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "keyword is required"})
		return
	}

	// Gọi service để tìm kiếm kênh
	channels, err := cc.ChannelService.SearchChannels(keyword)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"channels": channels})
}

// Lấy danh sách các kênh mà người dùng đã tham gia
func (cc *ChannelController) GetUserChannelsHandler(c *gin.Context) {
	userIDStr := c.Param("userID")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	channels, err := cc.ChannelService.GetChannelsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user channels"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"channels": channels})
}

// Tìm kênh riêng tư hoặc tạo kênh mới nếu không tìm thấy
func (cc *ChannelController) FindPrivateChannelHandler(ctx *gin.Context) {
	member1 := ctx.Query("member1")
	member2 := ctx.Query("member2")
	fmt.Println("member1:", member1, "member2:", member2)

	// Chuyển đổi ID thành ObjectID
	id1, err := primitive.ObjectIDFromHex(member1)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member1 ID"})
		return
	}
	id2, err := primitive.ObjectIDFromHex(member2)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member2 ID"})
		return
	}

	// Tìm kênh
	collection := cc.ChannelService.DB.Collection("channels")
	// Tìm kênh với điều kiện chính xác
	filter := bson.M{
		"channelType": models.ChannelTypePrivate,
		"members": bson.M{
			"$size": 2, // Đảm bảo chỉ có 2 thành viên trong kênh
			"$all": []bson.M{
				{"memberID": id1},
				{"memberID": id2},
			},
		},
	}

	var channel models.Channel
	err = collection.FindOne(context.TODO(), filter).Decode(&channel)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Nếu không tìm thấy kênh, tạo kênh mới
			// Gọi hàm tạo kênh mới
			newChannel, createErr := cc.ChannelService.CreateChannel("Private Channel", models.ChannelTypePrivate, []primitive.ObjectID{id1, id2}, false)
			if createErr != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": createErr.Error()})
				return
			}
			ctx.JSON(http.StatusCreated, newChannel) // Trả về kênh mới đã tạo
			return
		} else {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
	}

	ctx.JSON(http.StatusOK, channel)
}
