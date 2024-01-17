package mongorepo

import (
	"context"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/Plotrader/core/outbound"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	followsColName = "follows"
	ordersColName  = "orders"
)

type Config struct {
	DBName string
	URI    string
}

type Repository struct {
	dbName string
	uri    string
	c      *mongo.Client
}

func New(cfg Config) *Repository {
	return &Repository{
		dbName: cfg.DBName,
		uri:    cfg.URI,
	}
}

func (r *Repository) Connect(ctx context.Context) error {
	c, err := mongo.Connect(ctx, options.Client().ApplyURI(r.uri))
	if err != nil {
		return err
	}
	r.c = c
	return nil
}

func (r *Repository) Disconnect(ctx context.Context) error {
	return r.c.Disconnect(ctx)
}

// Follow
func (r *Repository) CreateFollow(ctx context.Context, req outbound.CreateFollowRequest) error {
	col := r.followsCol()
	_, err := col.InsertOne(ctx, req.Follow)
	return err
}

func (r *Repository) GetFollow(ctx context.Context, req outbound.GetFollowRequest) (domain.Follow, error) {
	col := r.followsCol()
	res := col.FindOne(ctx, bson.D{{Key: "id", Value: req.FollowID}})
	follow := domain.Follow{}
	if err := res.Decode(&follow); err != nil {
		return domain.Follow{}, err
	}
	return follow, nil
}

func (r *Repository) UpdateFollow(ctx context.Context, req outbound.UpdateFollowRequest) error {
	col := r.followsCol()
	_, err := col.ReplaceOne(ctx, bson.D{{Key: "id", Value: req.Follow}}, req.Follow)
	return err
}

// Order
func (r *Repository) CreateOrder(ctx context.Context, req outbound.CreateOrderRequest) error {
	col := r.ordersCol()
	_, err := col.InsertOne(ctx, req.Order)
	return err
}

func (r *Repository) GetOrder(ctx context.Context, req outbound.GetOrderRequest) (domain.Order, error) {
	col := r.ordersCol()
	res := col.FindOne(ctx, bson.D{{Key: "id", Value: req.OrderID}})
	order := domain.Order{}
	if err := res.Decode(&order); err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

func (r *Repository) UpdateOrder(ctx context.Context, req outbound.UpdateOrderRequest) error {
	col := r.ordersCol()
	_, err := col.ReplaceOne(ctx, bson.D{{Key: "id", Value: req.Order.ID}}, req.Order)
	return err
}

func (r *Repository) followsCol() *mongo.Collection {
	return r.c.Database(r.dbName).Collection(followsColName)
}

func (r *Repository) ordersCol() *mongo.Collection {
	return r.c.Database(r.dbName).Collection(ordersColName)
}
