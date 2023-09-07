package recommendation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
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

type Handler struct {
	svc Service
}

func NewHandler(svc Service) (*Handler, error) {
	if svc == (Service{}) {
		return nil, errors.New("svc cannot be empty")
	}
	return &Handler{svc: svc}, nil
}

type GetRecommendationResponse struct {
	HotelName string `json:"hotelName"`
	TotalCost struct {
		Cost     int64  `json:"cost"`
		Currency string `json:"currency"`
	} `json:"totalCost"`
}

func (handler Handler) GetRecommendation(responseWriter http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	location, ok := query["location"]
	if !ok {
		responseWriter.WriteHeader(http.StatusBadRequest)
		responseWriter.Write([]byte("location is required"))
		return
	}
	from, ok := query["from"]
	if !ok {
		responseWriter.WriteHeader(http.StatusBadRequest)
		responseWriter.Write([]byte("from is required"))
		return
	}
	to, ok := query["to"]
	if !ok {
		responseWriter.WriteHeader(http.StatusBadRequest)
		responseWriter.Write([]byte("to is required"))
		return
	}
	budget, ok := query["budget"]
	if !ok {
		responseWriter.WriteHeader(http.StatusBadRequest)
		responseWriter.Write([]byte("budget is required"))
		return
	}
	const expectedDateFormat = "2006-01-02"
	tripStart, err := time.Parse(expectedDateFormat, from[0])
	if err != nil {
		responseWriter.WriteHeader(http.StatusBadRequest)
		responseWriter.Write([]byte("invalid from date"))
		return
	}
	tripEnd, err := time.Parse(expectedDateFormat, to[0])
	if err != nil {
		responseWriter.WriteHeader(http.StatusBadRequest)
		responseWriter.Write([]byte("invalid to date"))
		return
	}
	budgetValue, err := strconv.Atoi(budget[0])
	if err != nil {
		responseWriter.WriteHeader(http.StatusBadRequest)
		responseWriter.Write([]byte("invalid budget"))
		return
	}
	budgetMoney := money.New(int64(budgetValue), "USD")

	recommendation, err := handler.svc.Get(request.Context(), tripStart, tripEnd, location[0], *budgetMoney)
	if err != nil {
		responseWriter.WriteHeader(http.StatusInternalServerError)
		responseWriter.Write([]byte(err.Error()))
		return
	}
	response, err := json.Marshal(GetRecommendationResponse{
		HotelName: recommendation.HotelName,
		TotalCost: struct {
			Cost     int64  `json:"cost"`
			Currency string `json:"currency"`
		}{
			Cost:     recommendation.TripPrice.Amount(),
			Currency: "USD",
		},
	})
	if err != nil {
		responseWriter.WriteHeader(http.StatusInternalServerError)
		responseWriter.Write([]byte(err.Error()))
		return
	}
	responseWriter.WriteHeader(http.StatusOK)
	_, _ = responseWriter.Write(response)
}
