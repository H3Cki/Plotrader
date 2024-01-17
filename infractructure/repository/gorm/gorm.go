package gormrepo

import (
	"context"
	"errors"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/Plotrader/core/outbound"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {

}

type Repository struct {
	dbName string
	db     *gorm.DB
}

func New(logger *zap.SugaredLogger, dbName string) *Repository {
	return &Repository{}
}

func (r *Repository) Connect(context.Context) error {
	db, err := gorm.Open(sqlite.Open(r.dbName), &gorm.Config{})
	if err != nil {
		return err
	}
	r.db = db
	return nil
}

// Follow
func (r *Repository) CreateFollow(ctx context.Context, req outbound.CreateFollowRequest) error {
	db := r.db.WithContext(ctx)
	db.Create(followFromDomain(req.Follow)).Commit()
	return nil
}

func (r *Repository) GetFollow(ctx context.Context, req outbound.GetFollowRequest) (domain.Follow, error) {
	var follow *Follow
	db := r.db.WithContext(ctx)
	db.First(follow, "ID = ?", req.FollowID)
	if follow == nil {
		return domain.Follow{}, errors.New("follow not found")
	}
	return follow.domain(), nil
}

func (r *Repository) UpdateFollow(ctx context.Context, req outbound.UpdateFollowRequest) error {
	f := followFromDomain(req.Follow)
	db := r.db.WithContext(ctx)
	db.Model(f).Where("ID == ?", f.ID).Updates(*f).Commit()
	return nil
}

// Order
func (r *Repository) CreateOrder(ctx context.Context, req outbound.CreateOrderRequest) error {
	db := r.db.WithContext(ctx)
	db.Create(orderFromDomain(req.Order)).Commit()
	return nil
}

func (r *Repository) GetOrder(ctx context.Context, req outbound.GetOrderRequest) (domain.Order, error) {
	var order *Order
	db := r.db.WithContext(ctx)
	db.First(order, "ID = ?", req.OrderID)
	if order == nil {
		return domain.Order{}, errors.New("follow not found")
	}
	return order.domain(), nil
}

func (r *Repository) UpdateOrder(ctx context.Context, req outbound.UpdateOrderRequest) error {
	o := orderFromDomain(req.Order)
	db := r.db.WithContext(ctx)
	db.Model(o).Where("ID == ?", o.ID).Updates(*o).Commit()
	return nil
}
