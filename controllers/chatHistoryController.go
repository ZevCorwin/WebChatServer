package controllers

import (
	"chat-app-backend/services"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
)

type ChatHistoryController struct {
	ChatHistoryService *services.ChatHistoryService
}

// NewChatHistoryController tạo một instance mới của ChatHistoryController
func NewChatHistoryController(service *services.ChatHistoryService) *ChatHistoryController {
	return &ChatHistoryController{ChatHistoryService: service}
}

func (chc *ChatHistoryController) GetChatHistory(ctx *gin.Context) {
	channelID := ctx.Param("channelID")
	id, err := primitive.ObjectIDFromHex(channelID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	message, err := chc.ChatHistoryService.GetChatHistory(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": message})
}

func (chc *ChatHistoryController) GetChatHistoryByUserID(ctx *gin.Context) {
	userID := ctx.Param("userID")
	id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	channels, err := chc.ChatHistoryService.GetChatHistoryByUserID(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"channels": channels})
}

func (chc *ChatHistoryController) DeleteChatHistory(ctx *gin.Context) {
	channelID := ctx.Param("channelID")
	id, err := primitive.ObjectIDFromHex(channelID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	err = chc.ChatHistoryService.DeleteChatHistory(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Chat history deleted succesfully"})
}
