package services

import (
	"chat-app-backend/config"
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ACLService struct {
	mu        sync.RWMutex
	cache     map[string]map[string]bool // userID -> set(permissionCode)
	cacheTill map[string]time.Time
	ttl       time.Duration
}

func NewACLService() *ACLService {
	return &ACLService{
		cache:     make(map[string]map[string]bool),
		cacheTill: make(map[string]time.Time),
		ttl:       5 * time.Minute,
	}
}

func (a *ACLService) GetUserPermissions(userID primitive.ObjectID) (map[string]bool, error) {
	key := userID.Hex()
	a.mu.RLock()
	if pset, ok := a.cache[key]; ok && time.Now().Before(a.cacheTill[key]) {
		defer a.mu.RUnlock()
		return pset, nil
	}
	a.mu.RUnlock()

	db := config.DB // dùng DB global đã InitDB()

	// Lấy roleIds của user
	var u struct {
		RoleIDs []primitive.ObjectID `bson:"roleIds"`
	}
	if err := db.Collection("users").FindOne(context.TODO(), bson.M{"_id": userID}).Decode(&u); err != nil {
		return nil, err
	}
	if len(u.RoleIDs) == 0 {
		a.mu.Lock()
		a.cache[key] = map[string]bool{}
		a.cacheTill[key] = time.Now().Add(a.ttl)
		a.mu.Unlock()
		return a.cache[key], nil
	}

	// Lấy permissions từ roles
	cur, err := db.Collection("roles").Aggregate(context.TODO(), bson.A{
		bson.M{"$match": bson.M{"_id": bson.M{"$in": u.RoleIDs}}},
		bson.M{"$lookup": bson.M{
			"from":         "permissions",
			"localField":   "permissions",
			"foreignField": "_id",
			"as":           "perms",
		}},
		bson.M{"$unwind": "$perms"},
		bson.M{"$group": bson.M{"_id": nil, "codes": bson.M{"$addToSet": "$perms.code"}}},
	})
	if err != nil {
		return nil, err
	}
	var agg struct {
		Codes []string `bson:"codes"`
	}
	if cur.Next(context.TODO()) {
		if err := cur.Decode(&agg); err != nil {
			return nil, err
		}
	}
	pset := map[string]bool{}
	for _, c := range agg.Codes {
		pset[c] = true
	}

	a.mu.Lock()
	a.cache[key] = pset
	a.cacheTill[key] = time.Now().Add(a.ttl)
	a.mu.Unlock()
	return pset, nil
}

func (a *ACLService) Has(userID primitive.ObjectID, perm string) bool {
	pset, err := a.GetUserPermissions(userID)
	if err != nil {
		return false
	}
	return pset[perm]
}
