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
	"time"
)

type FriendService struct {
	DB *mongo.Database
}

func NewFriendService() *FriendService {
	return &FriendService{DB: config.DB}
}

// Gửi yêu cầu kết bạn
func (fs *FriendService) SendFriendRequest(userID, friendID primitive.ObjectID) error {
	collection := fs.DB.Collection("listFriends")

	// Kiểm tra nếu đã tồn tại mối quan hệ
	filter := bson.M{
		"userID":   userID,
		"friendID": friendID,
	}
	err := collection.FindOne(context.Background(), filter).Err()
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}
	if err == nil {
		return errors.New("yêu cầu đã tồn tại")
	}
	if userID == friendID {
		return errors.New("không thể gửi yêu cầu kết bạn tới chính mình")
	}

	// Tạo yêu cầu mới
	now := time.Now().UTC()
	newRequest := models.ListFriends{
		ID:              primitive.NewObjectID(),
		UserID:          userID,
		FriendID:        friendID,
		FriendType:      models.FriendTypePending,
		RequestSentData: &now,
	}

	_, err = collection.InsertOne(context.Background(), newRequest)
	if err != nil {
		return errors.New("không thể gửi yêu cầu kết bạn")
	}
	return nil
}

// Hủy yêu cầu kết bạn
func (fs *FriendService) CancelFriendRequest(userID, friendID primitive.ObjectID) error {
	collection := fs.DB.Collection("listFriends")

	filter := bson.M{
		"userID":     userID,
		"friendID":   friendID,
		"friendType": models.FriendTypePending,
	}

	_, err := collection.DeleteOne(context.Background(), filter)
	if err != nil {
		return errors.New("không thể hủy yêu cầu kết bạn")
	}
	return nil
}

// Chấp nhận yêu cầu kết bạn
func (fs *FriendService) AcceptFriendRequest(userID, friendID primitive.ObjectID) error {
	collection := fs.DB.Collection("listFriends")
	now := time.Now().UTC()

	filter := bson.M{
		"userID":     friendID,
		"friendID":   userID,
		"friendType": models.FriendTypePending,
	}

	update := bson.M{
		"$set": bson.M{
			"friendType":  models.FriendTypeFriend,
			"confirmData": &now,
		},
	}

	_, err := collection.UpdateOne(context.Background(), filter, update)
	return err
}

// Từ chối yêu cầu kết bạn
func (fs *FriendService) DeclineFriendRequest(userID, friendID primitive.ObjectID) error {
	collection := fs.DB.Collection("listFriends")

	filter := bson.M{
		"userID":     friendID,
		"friendID":   userID,
		"friendType": models.FriendTypePending,
	}

	_, err := collection.DeleteOne(context.Background(), filter)
	return err
}

// Lấy danh sách bạn bè
func (fs *FriendService) GetFriends(userID primitive.ObjectID) ([]map[string]interface{}, error) {
	listFriendCollection := fs.DB.Collection("listFriends")
	userCollection := fs.DB.Collection("users")

	filter := bson.M{
		"$or": []bson.M{
			{"userID": userID, "friendType": models.FriendTypeFriend},
			{"friendID": userID, "friendType": models.FriendTypeFriend},
		},
	}

	cursor, err := listFriendCollection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			log.Println("Lỗi khi đóng cursor:", err)
		}
	}(cursor, ctx)

	var friends []models.ListFriends
	if err = cursor.All(context.Background(), &friends); err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, friend := range friends {
		var user struct {
			Name   string `bson:"name"`
			Phone  string `bson:"phone"`
			Avatar string `bson:"avatar"`
		}
		userFilter := bson.M{"_id": friend.UserID}
		err := userCollection.FindOne(ctx, userFilter).Decode(&user)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				log.Println("Không tìm thấy user: ", friend.UserID)
				continue
			}
			return nil, err
		}

		baseUrl := "http://localhost:8080"
		listFriendItem := map[string]interface{}{
			"friendID":     friend.UserID,
			"friendType":   models.FriendTypeFriend,
			"friendName":   user.Name,
			"friendPhone":  user.Phone,
			"friendAvatar": baseUrl + user.Avatar,
			"requestDate":  friend.RequestSentData,
		}
		result = append(result, listFriendItem)
	}
	return result, nil
}

// Xóa bạn bè
func (fs *FriendService) RemoveFriend(userID, friendID primitive.ObjectID) error {
	collection := fs.DB.Collection("listFriends")

	filter := bson.M{
		"$or": []bson.M{
			{"userID": userID, "friendID": friendID, "friendType": models.FriendTypeFriend},
			{"userID": friendID, "friendID": userID, "friendType": models.FriendTypeFriend},
		},
	}

	_, err := collection.DeleteOne(context.Background(), filter)
	return err
}

// Lấy danh sách lời mời kết bạn
func (fs *FriendService) GetFriendRequests(userID primitive.ObjectID) ([]map[string]interface{}, error) {
	listFriendCollection := fs.DB.Collection("listFriends")
	userCollection := fs.DB.Collection("users")

	filter := bson.M{
		"friendID":   userID,
		"friendType": models.FriendTypePending,
	}

	cursor, err := listFriendCollection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			log.Println("Lỗi khi đóng cursor:", err)
		}
	}(cursor, ctx)

	var requests []models.ListFriends
	if err = cursor.All(context.Background(), &requests); err != nil {
		return nil, err
	}

	// Kết quả trả về
	var result []map[string]interface{}
	for _, request := range requests {
		// Tìm thông tin user theo friendID
		var user struct {
			Name   string `bson:"name"`
			Avatar string `bson:"avatar"`
		}
		userFilter := bson.M{"_id": request.UserID}
		err := userCollection.FindOne(ctx, userFilter).Decode(&user)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				log.Println("Không tìm thấy user:", request.UserID)
				continue
			}
			return nil, err
		}
		baseUrl := "http://localhost:8080"
		listFriendRequestItem := map[string]interface{}{
			"friendID":     request.UserID,
			"friendName":   user.Name,
			"friendAvatar": baseUrl + user.Avatar,
			"requestID":    request.ID,
			"requestDate":  request.RequestSentData,
		}

		result = append(result, listFriendRequestItem)
	}

	return result, nil
}

// Tìm kiếm bạn bè theo tên
func (fs *FriendService) SearchFriendsByName(userID primitive.ObjectID, name string) ([]models.User, error) {
	collectionFriends := fs.DB.Collection("listFriends")
	collectionUsers := fs.DB.Collection("users")

	// Lấy danh sách bạn bè của userID
	filter := bson.M{
		"$or": []bson.M{
			{"userID": userID, "friendType": models.FriendTypeFriend},
			{"friendID": userID, "friendType": models.FriendTypeFriend},
		},
	}

	var friendRelations []models.ListFriends
	cursor, err := collectionFriends.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	if err = cursor.All(context.Background(), &friendRelations); err != nil {
		return nil, err
	}

	// Lấy danh sách ID bạn bè
	friendIDs := make([]primitive.ObjectID, 0)
	for _, relation := range friendRelations {
		if relation.UserID == userID {
			friendIDs = append(friendIDs, relation.FriendID)
		} else {
			friendIDs = append(friendIDs, relation.UserID)
		}
	}

	// Tìm kiếm bạn bè theo tên
	filterUsers := bson.M{
		"_id":  bson.M{"$in": friendIDs},
		"name": bson.M{"$regex": name, "$options": "i"}, // Tìm kiếm không phân biệt chữ hoa/thường
	}

	cursor, err = collectionUsers.Find(context.Background(), filterUsers)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var friends []models.User
	if err = cursor.All(context.Background(), &friends); err != nil {
		return nil, err
	}

	return friends, nil
}

// Kiểm tra trạng thái quan hệ bạn bè
func (fs *FriendService) CheckFriendStatus(userID, friendID primitive.ObjectID) (string, error) {
	collection := fs.DB.Collection("listFriends")

	filter := bson.M{
		"$or": []bson.M{
			{"userID": userID, "friendID": friendID},
			{"userID": friendID, "friendID": userID},
		},
	}

	var relation models.ListFriends
	err := collection.FindOne(context.Background(), filter).Decode(&relation)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Nếu không có tài liệu nào khớp với filter, trả về "none"
			return "none", nil
		}
		return "", err
	}

	return string(relation.FriendType), nil
}
