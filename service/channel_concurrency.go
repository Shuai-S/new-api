package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const (
	ginKeyChannelConcurrencyLease = "channel_concurrency_lease"

	channelConcurrencyNamespace    = "new-api:channel_concurrency:v1"
	channelConcurrencyPollInterval = 100 * time.Millisecond
	channelConcurrencyRedisKeyTTL  = 24 * time.Hour
)

const channelConcurrencyAcquireScript = `
local current = redis.call("GET", KEYS[1])
local limit = tonumber(ARGV[1])
local ttl_ms = tonumber(ARGV[2])
if not current then
	redis.call("SET", KEYS[1], 1, "PX", ttl_ms)
	return 1
end
current = tonumber(current)
if current < limit then
	current = redis.call("INCR", KEYS[1])
	redis.call("PEXPIRE", KEYS[1], ttl_ms)
	return 1
end
return 0
`

const channelConcurrencyReleaseScript = `
local current = redis.call("GET", KEYS[1])
local ttl_ms = tonumber(ARGV[1])
if not current then
	return 0
end
current = tonumber(current)
if current <= 1 then
	redis.call("DEL", KEYS[1])
	return 0
end
current = redis.call("DECR", KEYS[1])
redis.call("PEXPIRE", KEYS[1], ttl_ms)
return current
`

var (
	channelConcurrencyMemoryMu sync.Mutex
	channelConcurrencyMemory   = make(map[int]int)
)

type channelConcurrencyLease struct {
	channelID   int
	releaseOnce sync.Once
	releaseFunc func()
}

func (l *channelConcurrencyLease) Release() {
	if l == nil {
		return
	}
	l.releaseOnce.Do(func() {
		if l.releaseFunc != nil {
			l.releaseFunc()
		}
	})
}

func ReserveChannelForRequest(c *gin.Context, channel *model.Channel, retryParam *RetryParam, selectedGroup string, waitTimeout time.Duration) (*model.Channel, string, error) {
	if channel == nil {
		return nil, selectedGroup, nil
	}
	currentChannel := channel
	currentGroup := selectedGroup
	currentWait := waitTimeout

	for {
		acquired, err := acquireChannelConcurrencySlot(c, currentChannel, currentWait)
		if err != nil {
			return nil, currentGroup, err
		}
		if acquired {
			return currentChannel, currentGroup, nil
		}

		maxConcurrency := currentChannel.GetSetting().GetMaxConcurrency()
		logger.LogInfo(c, fmt.Sprintf("channel #%d max concurrency reached (%d), switching to another channel", currentChannel.Id, maxConcurrency))

		if retryParam == nil {
			return nil, currentGroup, nil
		}
		retryParam.ExcludeChannel(currentChannel.Id)
		currentWait = 0

		nextChannel, nextGroup, err := CacheGetRandomSatisfiedChannel(retryParam)
		if err != nil {
			return nil, nextGroup, err
		}
		if nextChannel == nil {
			return nil, nextGroup, nil
		}

		currentChannel = nextChannel
		currentGroup = nextGroup
	}
}

func ReleaseCurrentChannelConcurrency(c *gin.Context) {
	if c == nil {
		return
	}
	value, ok := c.Get(ginKeyChannelConcurrencyLease)
	if !ok || value == nil {
		return
	}
	lease, ok := value.(*channelConcurrencyLease)
	if !ok {
		return
	}
	lease.Release()
}

func acquireChannelConcurrencySlot(c *gin.Context, channel *model.Channel, waitTimeout time.Duration) (bool, error) {
	if channel == nil {
		return false, nil
	}
	maxConcurrency := channel.GetSetting().GetMaxConcurrency()
	if maxConcurrency <= 0 {
		return true, nil
	}

	deadline := time.Now().Add(waitTimeout)
	for {
		lease, acquired, err := tryAcquireChannelConcurrency(channel.Id, maxConcurrency)
		if err != nil {
			return false, err
		}
		if acquired {
			if c != nil {
				c.Set(ginKeyChannelConcurrencyLease, lease)
			}
			return true, nil
		}
		if waitTimeout <= 0 || time.Now().After(deadline) {
			return false, nil
		}
		if c != nil && c.Request != nil {
			select {
			case <-c.Request.Context().Done():
				return false, c.Request.Context().Err()
			case <-time.After(channelConcurrencyPollInterval):
			}
		} else {
			time.Sleep(channelConcurrencyPollInterval)
		}
	}
}

func tryAcquireChannelConcurrency(channelID int, maxConcurrency int) (*channelConcurrencyLease, bool, error) {
	if channelID <= 0 || maxConcurrency <= 0 {
		return nil, true, nil
	}
	if common.RedisEnabled && common.RDB != nil {
		return tryAcquireChannelConcurrencyRedis(channelID, maxConcurrency)
	}
	lease := tryAcquireChannelConcurrencyMemory(channelID, maxConcurrency)
	if lease == nil {
		return nil, false, nil
	}
	return lease, true, nil
}

func tryAcquireChannelConcurrencyMemory(channelID int, maxConcurrency int) *channelConcurrencyLease {
	channelConcurrencyMemoryMu.Lock()
	defer channelConcurrencyMemoryMu.Unlock()

	current := channelConcurrencyMemory[channelID]
	if current >= maxConcurrency {
		return nil
	}
	channelConcurrencyMemory[channelID] = current + 1
	return &channelConcurrencyLease{
		channelID: channelID,
		releaseFunc: func() {
			channelConcurrencyMemoryMu.Lock()
			defer channelConcurrencyMemoryMu.Unlock()

			currentCount := channelConcurrencyMemory[channelID]
			if currentCount <= 1 {
				delete(channelConcurrencyMemory, channelID)
				return
			}
			channelConcurrencyMemory[channelID] = currentCount - 1
		},
	}
}

func tryAcquireChannelConcurrencyRedis(channelID int, maxConcurrency int) (*channelConcurrencyLease, bool, error) {
	key := channelConcurrencyKey(channelID)
	result, err := common.RDB.Eval(
		context.Background(),
		channelConcurrencyAcquireScript,
		[]string{key},
		maxConcurrency,
		channelConcurrencyRedisKeyTTL.Milliseconds(),
	).Int64()
	if err != nil {
		return nil, false, err
	}
	if result == 0 {
		return nil, false, nil
	}
	return &channelConcurrencyLease{
		channelID: channelID,
		releaseFunc: func() {
			if _, releaseErr := common.RDB.Eval(
				context.Background(),
				channelConcurrencyReleaseScript,
				[]string{key},
				channelConcurrencyRedisKeyTTL.Milliseconds(),
			).Result(); releaseErr != nil {
				common.SysError(fmt.Sprintf("release channel concurrency failed: channel_id=%d, err=%v", channelID, releaseErr))
			}
		},
	}, true, nil
}

func channelConcurrencyKey(channelID int) string {
	return fmt.Sprintf("%s:%d", channelConcurrencyNamespace, channelID)
}
