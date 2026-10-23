package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Impervguin/ds-lab2/gateway/internal/domain"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
)

func TestGetRatingReturnsStarsAndTheBookLimit(t *testing.T) {
	s := newSuite(t)

	s.ratings.On("GetRating", mock.Anything).Return(&domain.Rating{Stars: 75, MaxBooks: 25}, nil)

	rating, err := usecase.NewRatingUseCase(s.ratings).GetRating(context.Background())

	require.NoError(t, err)
	assert.Equal(t, domain.Rating{Stars: 75, MaxBooks: 25}, *rating)
}

func TestGetRatingReportsAnUnavailableService(t *testing.T) {
	s := newSuite(t)

	s.ratings.On("GetRating", mock.Anything).Return(nil, errService)

	_, err := usecase.NewRatingUseCase(s.ratings).GetRating(context.Background())

	assert.ErrorIs(t, err, errService)
}
