package transport

import (
	"recommsystem/recommendation/internal/recommendation"

	"github.com/gorilla/mux"
)

func NewMux(recommendationHandler recommendation.Handler) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/recommendation", recommendationHandler.GetRecommendation).Methods("GET")
	return r
}
