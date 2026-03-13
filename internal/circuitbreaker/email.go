package circuitbreaker

import (
	"context"
	"log"
	"time"

	"github.com/sony/gobreaker/v2"
)

type EmailService struct {
	cb *gobreaker.CircuitBreaker[struct{}]
}

func NewEmailService() *EmailService {
	settings := gobreaker.Settings{
		Name:        "email-service",
		MaxRequests: 3,
		Interval:    30 * time.Second,
		Timeout:     10 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Printf("[circuit-breaker] %s: %s -> %s", name, from, to)
		},
	}

	return &EmailService{
		cb: gobreaker.NewCircuitBreaker[struct{}](settings),
	}
}

func (s *EmailService) SendInvite(_ context.Context, email, teamName string) error {
	_, err := s.cb.Execute(func() (struct{}, error) {
		log.Printf("[email-service] sending invite to %s for team %q", email, teamName)
		return struct{}{}, nil
	})
	return err
}
