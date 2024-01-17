package domain

import (
	"crypto/md5"
	"fmt"
	"time"
)

type FollowStatus string

var (
	FollowStatusPending FollowStatus = "PENDING"
	FollowStatusActive  FollowStatus = "ACTIVE"
	FollowStatusStopped FollowStatus = "STOPPED"
)

type ExchangeOrder struct {
	ID           any
	Status       OrderStatus
	Type         string
	Symbol       string
	Side         string
	Price        float64
	BaseQuantity float64
}

type Follow struct {
	ID           string        `json:"id"`
	Status       FollowStatus  `json:"status"`
	ExchangeHash string        `json:"exchangeHash"`
	Pair         Pair          `json:"pair"`
	Interval     time.Duration `json:"interval"`
	WebhookURL   string        `json:"webhookURL"`
	OrderIDs     []string      `json:"orderIDs"`
}

type Pair struct {
	Base  string `json:"base"`
	Quote string `json:"quote"`
}

func Hash(v any) (string, error) {
	hash := md5.New()
	_, err := hash.Write([]byte(fmt.Sprint(v)))
	if err != nil {
		return "", err
	}
	return string(hash.Sum([]byte{})), nil
}
