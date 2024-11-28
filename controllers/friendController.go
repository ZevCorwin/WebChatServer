package controllers

// Lưu ý về userID và friendID có một vài hàm còn dùng nhầm
import (
	"chat-app-backend/services"
	"errors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
)

type FriendController struct {
	FriendService *services.FriendService
}

func NewFriendController(friendService *services.FriendService) *FriendController {
	return &FriendController{FriendService: friendService}
}

func parseObjectIDs(ctx *gin.Context) (primitive.ObjectID, primitive.ObjectID, error) {
	userIDHex := ctx.Param("userID")
	friendIDHex := ctx.Param("friendID")

	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, errors.New("ID người dùng không hợp lệ")
	}

	friendID, err := primitive.ObjectIDFromHex(friendIDHex)
	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, errors.New("ID bạn bè không hợp lệ")
	}

	return userID, friendID, nil
}

// Gửi yêu cầu kết bạn
func (fc *FriendController) SendFriendRequest(ctx *gin.Context) {
	userID, friendID, err := parseObjectIDs(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = fc.FriendService.SendFriendRequest(userID, friendID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Gửi lời mời kết bạn thành công"})
}

// Hủy lời mời kết bạn
func (fc *FriendController) CancelFriendRequest(ctx *gin.Context) {
	userID, friendID, err := parseObjectIDs(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = fc.FriendService.CancelFriendRequest(userID, friendID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Hủy lời mời kết bạn thành công"})
}

// Chấp nhận yêu cầu kết bạn
func (fc *FriendController) AcceptFriendRequest(ctx *gin.Context) {
	userID, friendID, err := parseObjectIDs(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = fc.FriendService.AcceptFriendRequest(userID, friendID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Chấp nhận lời mời kết bạn thành công"})
}

// Từ chối yêu cầu kết bạn
func (fc *FriendController) DeclineFriendRequest(ctx *gin.Context) {
	userID, friendID, err := parseObjectIDs(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = fc.FriendService.DeclineFriendRequest(userID, friendID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Từ chối lời mời kết bạn thành công"})
}

// Lấy danh sách bạn bè
func (fc *FriendController) GetFriends(ctx *gin.Context) {
	userIDHex := ctx.Param("userID")

	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID người dùng không hợp lệ"})
		return
	}

	friends, err := fc.FriendService.GetFriends(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"friends": friends})
}

// Xóa bạn bè
func (fc *FriendController) RemoveFriend(ctx *gin.Context) {
	userID, friendID, err := parseObjectIDs(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = fc.FriendService.RemoveFriend(userID, friendID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Xóa bạn bè thành công"})
}

// Lấy danh sách lời mời kết bạn
func (fc *FriendController) GetFriendRequests(ctx *gin.Context) {
	userIDHex := ctx.Param("userID")

	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID người dùng không hợp lệ"})
		return
	}

	requests, err := fc.FriendService.GetFriendRequests(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"requests": requests})
}

// Tìm kiếm bạn bè theo tên
func (fc *FriendController) SearchFriendsByName(ctx *gin.Context) {
	userIDHex := ctx.Param("userID")
	name := ctx.Query("name")

	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID người dùng không hợp lệ"})
		return
	}

	friends, err := fc.FriendService.SearchFriendsByName(userID, name)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"friends": friends})
}

// Kiểm tra trạng thái quan hệ bạn bè
func (fc *FriendController) CheckFriendStatus(ctx *gin.Context) {
	userID, friendID, err := parseObjectIDs(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if userID == primitive.NilObjectID || friendID == primitive.NilObjectID {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "userID hoặc friendID không hợp lệ"})
		return
	}

	status, err := fc.FriendService.CheckFriendStatus(userID, friendID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": status})
}
