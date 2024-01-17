package binancefutures

import (
	"context"
	"reflect"
	"testing"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/Plotrader/core/outbound"
	"github.com/H3Cki/go-binance/v2/futures"
	"go.uber.org/zap"
)

func TestExchange_ModifyOrder(t *testing.T) {
	type fields struct {
		logger *zap.SugaredLogger
		client *futures.Client
		ei     ExchangeInfo
		eier   outbound.FileLoader[ExchangeInfo]
	}
	type args struct {
		ctx context.Context
		req outbound.ModifyExchangeOrderRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domain.ExchangeOrder
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Exchange{
				logger: tt.fields.logger,
				client: tt.fields.client,
				ei:     tt.fields.ei,
				eier:   tt.fields.eier,
			}
			got, err := f.ModifyOrder(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Exchange.ModifyOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Exchange.ModifyOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExchange_modifyOrder(t *testing.T) {
	type fields struct {
		logger *zap.SugaredLogger
		client *futures.Client
		ei     ExchangeInfo
		eier   outbound.FileLoader[ExchangeInfo]
	}
	type args struct {
		ctx context.Context
		req outbound.ModifyExchangeOrderRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domain.ExchangeOrder
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Exchange{
				logger: tt.fields.logger,
				client: tt.fields.client,
				ei:     tt.fields.ei,
				eier:   tt.fields.eier,
			}
			got, err := f.modifyOrder(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Exchange.modifyOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Exchange.modifyOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExchange_CancelOrder(t *testing.T) {
	type fields struct {
		logger *zap.SugaredLogger
		client *futures.Client
		ei     ExchangeInfo
		eier   outbound.FileLoader[ExchangeInfo]
	}
	type args struct {
		ctx context.Context
		req outbound.CancelExchangeOrdersRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domain.ExchangeOrder
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Exchange{
				logger: tt.fields.logger,
				client: tt.fields.client,
				ei:     tt.fields.ei,
				eier:   tt.fields.eier,
			}
			got, err := f.CancelOrder(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Exchange.CancelOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Exchange.CancelOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExchange_cancelOrder(t *testing.T) {
	type fields struct {
		logger *zap.SugaredLogger
		client *futures.Client
		ei     ExchangeInfo
		eier   outbound.FileLoader[ExchangeInfo]
	}
	type args struct {
		ctx context.Context
		req outbound.CancelExchangeOrdersRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domain.ExchangeOrder
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Exchange{
				logger: tt.fields.logger,
				client: tt.fields.client,
				ei:     tt.fields.ei,
				eier:   tt.fields.eier,
			}
			got, err := f.cancelOrder(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Exchange.cancelOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Exchange.cancelOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}
