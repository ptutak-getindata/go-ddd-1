package main

import (
	"log"
	"net/http"
	"recommsystem/recommendation/internal/recommendation"
	"recommsystem/recommendation/internal/transport"

	"github.com/hashicorp/go-retryablehttp"
)

func main() {
	retryableClient := retryablehttp.NewClient()
	retryableClient.RetryMax = 5
	partnershipAdaptor, err := recommendation.NewPartnershipAdaptor(retryableClient.StandardClient(), "http://localhost:3031")
	if err != nil {
		log.Fatal("failed to create partnership adaptor: ", err)
	}
	recommendationService, err := recommendation.NewService(partnershipAdaptor)
	if err != nil {
		log.Fatal("failed to create recommendation service: ", err)
	}
	recommendationHandler, err := recommendation.NewHandler(*recommendationService)
	if err != nil {
		log.Fatal("failed to create recommendation handler: ", err)
	}
	mux := transport.NewMux(*recommendationHandler)
	if err := http.ListenAndServe(":4040", mux); err != nil {
		log.Fatal("failed to serve: ", err)
	}
}
