package services

import (
	"chat-app-backend/config"
	"chat-app-backend/models" // Đảm bảo đã import models
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RoleService struct {
	DB *mongo.Database
}

// NewRoleService tạo một service mới
func NewRoleService() *RoleService {
	return &RoleService{DB: config.DB}
}

// ListPermissions lấy tất cả các quyền (permissions) có trong hệ thống
func (rs *RoleService) ListPermissions(ctx context.Context) ([]models.Permission, error) {
	permColl := rs.DB.Collection("permissions") // Tên collection chứa quyền

	// Sắp xếp theo "code" để danh sách trả về luôn ổn định
	opts := options.Find().SetSort(bson.D{{Key: "code", Value: 1}})

	cursor, err := permColl.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var permissions []models.Permission
	if err = cursor.All(ctx, &permissions); err != nil {
		return nil, err
	}

	// Trả về mảng rỗng thay vì nil
	if permissions == nil {
		return []models.Permission{}, nil
	}

	return permissions, nil
}

// ListRoles lấy tất cả các vai trò (roles) trong hệ thống
func (rs *RoleService) ListRoles(ctx context.Context) ([]models.RoleModel, error) {
	roleColl := rs.DB.Collection("roles") // Tên collection của RoleModel

	opts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}}) // Sắp xếp theo tên

	cursor, err := roleColl.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var roles []models.RoleModel
	if err = cursor.All(ctx, &roles); err != nil {
		return nil, err
	}

	if roles == nil {
		return []models.RoleModel{}, nil
	}

	return roles, nil
}

// CreateRole tạo một vai trò mới
// Payload đầu vào là tên và một mảng các ObjectID của quyền
func (rs *RoleService) CreateRole(ctx context.Context, name string, permissionIDs []primitive.ObjectID) (*models.RoleModel, error) {
	roleColl := rs.DB.Collection("roles")

	// (Nâng cao - có thể thêm sau): Kiểm tra xem các permissionIDs có thật
	// trong collection 'permissions' hay không.
	// Hiện tại, ta tin tưởng Super Admin gửi lên ID đúng.

	newRole := models.RoleModel{
		ID:          primitive.NewObjectID(),
		Name:        name,
		Permissions: permissionIDs, // Gán mảng ID quyền vào
	}

	_, err := roleColl.InsertOne(ctx, newRole)
	if err != nil {
		// (Nâng cao): Check lỗi duplicate key nếu em có tạo index cho "name"
		return nil, err
	}

	return &newRole, nil
}

// UpdateRole cập nhật một vai trò (tên và danh sách quyền)
func (rs *RoleService) UpdateRole(ctx context.Context, roleID primitive.ObjectID, name string, permissionIDs []primitive.ObjectID) error {
	roleColl := rs.DB.Collection("roles")

	// Đảm bảo mảng permissionIDs không bị nil (nếu frontend gửi null)
	if permissionIDs == nil {
		permissionIDs = []primitive.ObjectID{}
	}

	// Tạo filter (tìm đúng ID) và update (cập nhật 2 trường)
	filter := bson.M{"_id": roleID}
	update := bson.M{
		"$set": bson.M{
			"name":        name,
			"permissions": permissionIDs,
		},
	}

	// Thực hiện cập nhật
	result, err := roleColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return err // Lỗi CSDL
	}

	// Kiểm tra xem có update được không (để báo lỗi nếu sai ID)
	if result.MatchedCount == 0 {
		return errors.New("không tìm thấy vai trò với ID này")
	}

	return nil
}

// DeleteRole xóa một vai trò
func (rs *RoleService) DeleteRole(ctx context.Context, roleID primitive.ObjectID) error {
	roleColl := rs.DB.Collection("roles")
	userColl := rs.DB.Collection("users")

	// --- KIỂM TRA QUAN TRỌNG ---
	// 1. Kiểm tra xem có user nào đang giữ vai trò (RoleID) này không
	// Ta dùng trường RoleIDs trong models.User
	filterCheck := bson.M{"roleIds": roleID}
	count, err := userColl.CountDocuments(ctx, filterCheck)
	if err != nil {
		// Lỗi CSDL khi đang kiểm tra
		return errors.New("lỗi khi kiểm tra user: " + err.Error())
	}

	if count > 0 {
		// Nếu có (ví dụ: 1 user) đang dùng vai trò này -> từ chối xóa
		return errors.New("không thể xóa vai trò này vì đang có người dùng sử dụng")
	}
	// --- KẾT THÚC KIỂM TRA ---

	// 2. Nếu không có ai dùng, tiến hành xóa
	filterDelete := bson.M{"_id": roleID}
	result, err := roleColl.DeleteOne(ctx, filterDelete)
	if err != nil {
		return err // Lỗi CSDL
	}

	// Kiểm tra xem có xóa được không
	if result.DeletedCount == 0 {
		return errors.New("không tìm thấy vai trò với ID này")
	}

	return nil
}

// AssignRoleToUser gán một vai trò cho một người dùng (Ban chức)
func (rs *RoleService) AssignRoleToUser(ctx context.Context, userID primitive.ObjectID, roleID primitive.ObjectID) error {
	userColl := rs.DB.Collection("users")

	filter := bson.M{"_id": userID}
	update := bson.M{
		// 1. Thêm vai trò động
		"$addToSet": bson.M{"roleIds": roleID},
		// 2. NÂNG CẤP: Đặt vai trò cứng thành "Quản trị viên"
		"$set": bson.M{"role": models.RoleAdmin},
	}

	result, err := userColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("không tìm thấy người dùng với ID này")
	}
	return nil
}

// RevokeRoleFromUser tước một vai trò khỏi người dùng (Giáng chức)
func (rs *RoleService) RevokeRoleFromUser(ctx context.Context, adminID, targetUserID, roleID primitive.ObjectID) error {

	if adminID == targetUserID {
		return errors.New("không thể tự tước vai trò của chính mình")
	}

	userColl := rs.DB.Collection("users")

	// 1. Tước vai trò động (pull from roleIds)
	filter := bson.M{"_id": targetUserID}
	update := bson.M{
		"$pull": bson.M{"roleIds": roleID},
	}

	result, err := userColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("không tìm thấy người dùng (mục tiêu) với ID này")
	}

	// 2. NÂNG CẤP: Đọc lại user để kiểm tra các vai trò còn lại
	var updatedUser models.User
	err = userColl.FindOne(ctx, bson.M{"_id": targetUserID}).Decode(&updatedUser)
	if err != nil {
		// Vẫn trả về nil vì đã tước quyền thành công, chỉ là bước 2 bị lỗi
		// Hoặc có thể trả lỗi tùy logic em muốn
		return errors.New("tước vai trò động thành công, nhưng không thể đọc lại user: " + err.Error())
	}

	// 3. NÂNG CẤP: Nếu user không còn vai trò động nào...
	if len(updatedUser.RoleIDs) == 0 {
		// ... thì giáng cấp vai trò CỨNG về "Người dùng"
		updateRole := bson.M{"$set": bson.M{"role": models.RoleUser}}
		_, err = userColl.UpdateOne(ctx, filter, updateRole)
		if err != nil {
			return errors.New("tước vai trò động thành công, nhưng giáng cấp vai trò cứng thất bại: " + err.Error())
		}
	}

	// Nếu user vẫn còn vai trò động khác (ví dụ: "Super Admin"),
	// thì không làm gì cả, cứ để vai trò cứng là "Quản trị viên".

	return nil
}

/*
   Tương lai chúng ta sẽ thêm các hàm ở đây:
   - ListRoles()
   - CreateRole(name string, permissionIDs []primitive.ObjectID)
   - UpdateRole(roleID, name, permissionIDs)
   - DeleteRole(roleID)
   - AssignRoleToUser(userID, roleID)
   - RevokeRoleFromUser(userID, roleID)
*/
