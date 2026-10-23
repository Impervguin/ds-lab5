package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Impervguin/ds-lab2/library/internal/domain"
)

func TestConditionRankOrdersFromBestToWorst(t *testing.T) {
	assert.Less(t, domain.ConditionExcellent.Rank(), domain.ConditionGood.Rank())
	assert.Less(t, domain.ConditionGood.Rank(), domain.ConditionBad.Rank())
	assert.Greater(t, domain.BookCondition("DESTROYED").Rank(), domain.ConditionBad.Rank())
}

func TestConditionValid(t *testing.T) {
	for _, condition := range domain.Conditions {
		assert.True(t, condition.Valid(), "%q is not recognised", condition)
	}

	assert.False(t, domain.BookCondition("DESTROYED").Valid())
	assert.False(t, domain.BookCondition("excellent").Valid(), "conditions are case sensitive in the contract")
	assert.False(t, domain.BookCondition("").Valid())
}
