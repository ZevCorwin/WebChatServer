package controllers

import (
	"chat-app-backend/models"
	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
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
		UserID           primitive.ObjectID   `json:"userID"`
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
	channel, err := cc.ChannelService.CreateChannel(req.UserID, req.Name, req.Type, req.Members, req.ApprovalRequired)
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

// RemoveMemberHandler xóa thành viên khỏi kênh
func (cc *ChannelController) RemoveMemberHandler(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelID")
	memberIdStr := ctx.Param("memberID")

	// Lấy user từ JWT middleware (đã xác thực thành công trước đó)
	userID := ctx.GetString("user_id")
	removerID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid remover ID"})
		return
	}

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

	// Lấy thông tin kênh
	channel, err := cc.ChannelService.GetChannel(channelID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	// Log thông tin role hiện tại để debug
	for _, m := range channel.Members {
		if m.MemberID == removerID {
			log.Printf("[RemoveMember] remover=%s role=%s", removerID.Hex(), m.Role)
		}
	}

	// Thực hiện xóa thành viên
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
	channelID := ctx.Param("channelID")
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

	// Parse IDs từ param
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

	// Tùy chọn: newLeaderID trong body JSON
	var req struct {
		NewLeaderID string `json:"newLeaderID"`
	}
	var newLeaderID *primitive.ObjectID
	if err := ctx.ShouldBindJSON(&req); err == nil && req.NewLeaderID != "" {
		oid, err := primitive.ObjectIDFromHex(req.NewLeaderID)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid new leader ID"})
			return
		}
		newLeaderID = &oid
	}

	// Lấy channel
	channel, err := cc.ChannelService.GetChannel(channelID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	// Rời nhóm (nếu là leader mà không gửi newLeaderID -> service sẽ báo lỗi)
	if err := cc.ChannelService.LeaveChannel(channel, memberID, newLeaderID); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Lưu thay đổi vào DB
	if err := cc.ChannelService.UpdateChannel(channel); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update channel"})
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

	channel, err := cc.ChannelService.FindOrCreatePrivateChannel(member1, member2)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	ctx.JSON(http.StatusOK, channel)
}
