package services

import (
	"chat-app-backend/config"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type StatsService struct {
	DB *mongo.Database
}

func NewStatsService() *StatsService {
	return &StatsService{DB: config.DB}
}

// Struct để trả về dữ liệu cho dashboard
type OverviewStats struct {
	TotalUsers    int64 `json:"totalUsers"`
	TotalMessages int64 `json:"totalMessages"`
	TotalChannels int64 `json:"totalChannels"`
	DAU           int64 `json:"dau"`
	MAU           int64 `json:"mau"`
}

// Dùng để đếm số lượng theo nhóm (ví dụ: theo loại, theo giờ)
type GroupCount struct {
	ID    interface{} `json:"id" bson:"_id"`
	Count int64       `json:"count"`
}

// Struct trả về cho thống kê tin nhắn
type MessageActivityStats struct {
	ByType []GroupCount `json:"byType"` // Phân loại theo MessageType (Text, File, Voice...)
	ByHour []GroupCount `json:"byHour"` // Phân loại theo giờ trong ngày (0-23)
}

// Lấy các thống kê tổng quan
func (ss *StatsService) GetOverviewStats(ctx context.Context) (*OverviewStats, error) {
	// Lấy các collection cần thiết
	userColl := ss.DB.Collection("users")
	msgColl := ss.DB.Collection("messages")
	chanColl := ss.DB.Collection("channels")

	// 1. Đếm tổng số user
	// Dùng filter bson.M{} (rỗng) để đếm tất cả
	totalUsers, err := userColl.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	// 2. Đếm tổng số tin nhắn
	totalMessages, err := msgColl.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	// 3. Đếm tổng số kênh chat (cả private và group)
	totalChannels, err := chanColl.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	// 4. Tính DAU (Daily Active Users)
	// Lấy thời điểm đầu ngày hôm nay (00:00:00)
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	dau, err := userColl.CountDocuments(ctx, bson.M{
		"lastOnlineTime": bson.M{"$gte": startOfDay}, // <--- SỬA TÊN TRƯỜNG Ở ĐÂY
	})
	if err != nil {
		return nil, err
	}

	// 5. Tính MAU (Monthly Active Users)
	// Lấy thời điểm 30 ngày trước
	startOf30DaysAgo := now.AddDate(0, 0, -30)

	mau, err := userColl.CountDocuments(ctx, bson.M{
		"lastOnlineTime": bson.M{"$gte": startOf30DaysAgo}, // <--- SỬA TÊN TRƯỜNG Ở ĐÂY
	})
	if err != nil {
		return nil, err
	}

	// Tạo đối tượng để trả về
	stats := &OverviewStats{
		TotalUsers:    totalUsers,
		TotalMessages: totalMessages,
		TotalChannels: totalChannels,
		DAU:           dau,
		MAU:           mau,
	}

	return stats, nil
}

// Lấy thống kê hoạt động của tin nhắn (theo loại, theo giờ)
func (ss *StatsService) GetMessageActivityStats(ctx context.Context) (*MessageActivityStats, error) {
	msgColl := ss.DB.Collection("messages")

	// Lấy múi giờ của server (ví dụ: "Asia/Ho_Chi_Minh")
	location, err := time.LoadLocation("Asia/Ho_Chi_Minh") // <-- Giữ nguyên
	if err != nil {
		location = time.UTC // Fallback về UTC
	}

	// === SỬA LỖI CÚ PHÁP Ở ĐÂY ===
	// Chúng ta phải dùng bson.D cho mỗi giai đoạn của pipeline
	pipeline := mongo.Pipeline{
		// Giai đoạn 1: $facet
		bson.D{
			{Key: "$facet", Value: bson.M{
				// Pipeline con 1: byType
				"byType": mongo.Pipeline{
					// Giai đoạn 1.1: $group
					bson.D{
						{Key: "$group", Value: bson.M{
							"_id":   "$messageType", // Nhóm theo trường messageType
							"count": bson.M{"$sum": 1},
						}},
					},
				},
				// Pipeline con 2: byHour
				"byHour": mongo.Pipeline{
					// Giai đoạn 2.1: $group
					bson.D{
						{Key: "$group", Value: bson.M{
							// Trích xuất giờ từ timestamp (theo múi giờ đã set)
							"_id": bson.M{
								"$hour": bson.M{
									"date":     "$timestamp",
									"timezone": location.String(),
								},
							},
							"count": bson.M{"$sum": 1},
						}},
					},
					// Giai đoạn 2.2: $sort
					bson.D{
						{Key: "$sort", Value: bson.M{"_id": 1}}, // Sắp xếp theo giờ tăng dần (0h -> 23h)
					},
				},
			}},
		},
	}
	// === KẾT THÚC SỬA LỖI ===

	// Chạy aggregation
	cursor, err := msgColl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Vì $facet luôn trả về 1 mảng chứa 1 document, ta lấy document đầu tiên
	var result []struct {
		ByType []GroupCount `bson:"byType"`
		ByHour []GroupCount `bson:"byHour"`
	}
	if err = cursor.All(ctx, &result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		// Không có tin nhắn nào, trả về mảng rỗng
		return &MessageActivityStats{
			ByType: []GroupCount{},
			ByHour: []GroupCount{},
		}, nil
	}

	// Trả về kết quả
	return &MessageActivityStats{
		ByType: result[0].ByType,
		ByHour: result[0].ByHour,
	}, nil
}

// GetUserGrowthStats lấy thống kê tăng trưởng người dùng mới (theo ngày)
func (ss *StatsService) GetUserGrowthStats(ctx context.Context) ([]GroupCount, error) {
	userColl := ss.DB.Collection("users")

	// Lấy múi giờ (giống lần trước, để nhóm theo ngày cho đúng)
	location, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		location = time.UTC
	}

	// Định nghĩa pipeline
	pipeline := mongo.Pipeline{
		// Giai đoạn 1: $group
		bson.D{
			{Key: "$group", Value: bson.M{
				// Nhóm theo ngày (YYYY-MM-DD)
				// Dùng $dateToString để chuyển timestamp thành chuỗi ngày
				"_id": bson.M{
					"$dateToString": bson.M{
						"format":   "%Y-%m-%d",            // Format: 2025-11-04
						"date":     "$accountCreatedDate", // Dùng trường này từ models.User
						"timezone": location.String(),
					},
				},
				"count": bson.M{"$sum": 1},
			}},
		},
		// Giai đoạn 2: $sort
		bson.D{
			{Key: "$sort", Value: bson.M{"_id": 1}}, // Sắp xếp theo ngày tăng dần
		},
	}

	// Chạy aggregation
	cursor, err := userColl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []GroupCount
	if err = cursor.All(ctx, &result); err != nil {
		return nil, err
	}

	// Đảm bảo trả về mảng rỗng thay vì nil nếu không có kết quả
	if result == nil {
		return []GroupCount{}, nil
	}

	return result, nil
}
