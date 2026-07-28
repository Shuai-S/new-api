package controller

import (
	"errors"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldRetryByKeyword(t *testing.T) {
	origRetryKeywords := operation_setting.AutomaticRetryKeywords
	origRetryStatusCodes := operation_setting.AutomaticRetryStatusCodeRanges
	t.Cleanup(func() {
		operation_setting.AutomaticRetryKeywords = origRetryKeywords
		operation_setting.AutomaticRetryStatusCodeRanges = origRetryStatusCodes
	})

	operation_setting.AutomaticRetryKeywordsFromString("overloaded\nTemporarily Unavailable")
	operation_setting.AutomaticRetryStatusCodeRanges = nil

	c := &gin.Context{}
	err := types.NewErrorWithStatusCode(
		errors.New("upstream is temporarily unavailable"),
		types.ErrorCodeBadResponseStatusCode,
		http.StatusBadRequest,
	)

	require.True(t, shouldRetry(c, err, 1))
}

func TestShouldRetryByKeywordHonorsStopConditions(t *testing.T) {
	origRetryKeywords := operation_setting.AutomaticRetryKeywords
	origRetryStatusCodes := operation_setting.AutomaticRetryStatusCodeRanges
	t.Cleanup(func() {
		operation_setting.AutomaticRetryKeywords = origRetryKeywords
		operation_setting.AutomaticRetryStatusCodeRanges = origRetryStatusCodes
	})

	operation_setting.AutomaticRetryKeywordsFromString("temporary")
	operation_setting.AutomaticRetryStatusCodeRanges = nil

	testCases := []struct {
		name       string
		err        *types.NewAPIError
		retryTimes int
		setup      func(*gin.Context)
	}{
		{
			name: "skip retry error",
			err: types.NewErrorWithStatusCode(
				errors.New("temporary upstream failure"),
				types.ErrorCodeInvalidRequest,
				http.StatusBadRequest,
				types.ErrOptionWithSkipRetry(),
			),
			retryTimes: 1,
		},
		{
			name: "no remaining retries",
			err: types.NewErrorWithStatusCode(
				errors.New("temporary upstream failure"),
				types.ErrorCodeBadResponseStatusCode,
				http.StatusBadRequest,
			),
			retryTimes: 0,
		},
		{
			name: "specific channel",
			err: types.NewErrorWithStatusCode(
				errors.New("temporary upstream failure"),
				types.ErrorCodeBadResponseStatusCode,
				http.StatusBadRequest,
			),
			retryTimes: 1,
			setup: func(c *gin.Context) {
				c.Set("specific_channel_id", 1)
			},
		},
		{
			name: "2xx status",
			err: types.NewErrorWithStatusCode(
				errors.New("temporary upstream failure"),
				types.ErrorCodeBadResponseStatusCode,
				http.StatusOK,
			),
			retryTimes: 1,
		},
		{
			name: "always skip status 504",
			err: types.NewErrorWithStatusCode(
				errors.New("temporary upstream failure"),
				types.ErrorCodeBadResponseStatusCode,
				http.StatusGatewayTimeout,
			),
			retryTimes: 1,
		},
		{
			name: "always skip status 524",
			err: types.NewErrorWithStatusCode(
				errors.New("temporary upstream failure"),
				types.ErrorCodeBadResponseStatusCode,
				524,
			),
			retryTimes: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := &gin.Context{}
			if tc.setup != nil {
				tc.setup(c)
			}

			assert.False(t, shouldRetry(c, tc.err, tc.retryTimes))
		})
	}
}
