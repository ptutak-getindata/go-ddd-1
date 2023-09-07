package recommendation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Rhymond/go-money"
)

type PartnershipAdaptor struct {
	client *http.Client
	url    string
}

func NewPartnershipAdaptor(client *http.Client, url string) (*PartnershipAdaptor, error) {
	if client == nil {
		return nil, errors.New("client cannot be nil")
	}
	if url == "" {
		return nil, errors.New("url cannot be empty")
	}

	return &PartnershipAdaptor{client: client, url: url}, nil
}

type partnershipResponse struct {
	AvailableHotels []struct {
		Name               string `json:"name"`
		PriceInUSDPerNight int    `json:"priceInUSDPerNight"`
	} `json:"availableHotels"`
}

func (pa PartnershipAdaptor) GetAvailability(ctx context.Context, tripStart time.Time, tripEnd time.Time, location string) ([]Option, error) {
	from := fmt.Sprintf("%d-%d-%d", tripStart.Year(), tripStart.Month(), tripStart.Day())
	to := fmt.Sprintf("%d-%d-%d", tripEnd.Year(), tripEnd.Month(), tripEnd.Day())
	url := fmt.Sprintf("%s/partnerships?from=%s&to=%s&location=%s", pa.url, from, to, location)
	res, err := pa.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get availability: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad request to partnerships: %d", res.StatusCode)
	}
	var response partnershipResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	var opts []Option = make([]Option, len(response.AvailableHotels))
	for i, hotel := range response.AvailableHotels {
		opts[i] = Option{
			Location:      location,
			HotelName:     hotel.Name,
			PricePerNight: *money.New(int64(hotel.PriceInUSDPerNight), "USD"),
		}
	}
	return opts, nil
}
