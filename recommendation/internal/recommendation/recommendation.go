package recommendation

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/Rhymond/go-money"
)

type Recommendation struct {
	TripStart time.Time
	TripEnd   time.Time
	Location  string
	HotelName string
	TripPrice money.Money
}

type Option struct {
	Location      string
	HotelName     string
	PricePerNight money.Money
}

type AvailabilityGetter interface {
	GetAvailability(ctx context.Context, tripStart time.Time, tripEnd time.Time, location string) ([]Option, error)
}

type Service struct {
	avaiability AvailabilityGetter
}

func NewService(availability AvailabilityGetter) (*Service, error) {
	if availability == nil {
		return nil, errors.New("availability cannot be nil")
	}
	return &Service{avaiability: availability}, nil
}

func (svc *Service) Get(ctx context.Context, tripStart time.Time, tripEnd time.Time, location string, budget money.Money) (*Recommendation, error) {
	switch {
	case tripStart.IsZero():
		return nil, errors.New("tripStart cannot be zero")
	case tripEnd.IsZero():
		return nil, errors.New("tripEnd cannot be zero")
	case location == "":
		return nil, errors.New("location cannot be empty")
	}
	opts, err := svc.avaiability.GetAvailability(ctx, tripStart, tripEnd, location)
	if err != nil {
		return nil, fmt.Errorf("failed to get availability: %w", err)
	}
	tripDuration := math.Round(tripEnd.Sub(tripStart).Hours() / 24)
	lowestPrice := money.New(999999999, "USD")
	var lowestOption *Option
	for _, opt := range opts {
		totalPrice := opt.PricePerNight.Multiply(int64(tripDuration))
		if ok, _ := totalPrice.GreaterThan(&budget); ok {
			continue
		}
		if ok, _ := totalPrice.LessThan(lowestPrice); ok {
			lowestPrice = totalPrice
			lowestOption = &opt
		}
	}
	if lowestOption == nil {
		return nil, errors.New("no options available")
	}
	return &Recommendation{
		TripStart: tripStart,
		TripEnd:   tripEnd,
		Location:  location,
		HotelName: lowestOption.HotelName,
		TripPrice: *lowestPrice,
	}, nil
}
