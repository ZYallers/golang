package redis

import (
	"fmt"
	"github.com/ZYallers/golib/types"
	"github.com/go-redis/redis"
	"sync/atomic"
	"time"
)

const (
	retryMaxTimes  = 3
	retrySleepTime = 100 * time.Millisecond
)

type Redis struct {
	Client func() *redis.Client
}

func (r *Redis) NewRedis(rdc *types.RedisCollector, cli *types.RedisClient, options func() *redis.Options) (*redis.Client, error) {
	var err error
	for i := 1; i <= retryMaxTimes; i++ {
		if atomic.LoadUint32(&rdc.Done) == 0 {
			rdc.M.Lock()
			if rdc.Done == 0 {
				opts := &redis.Options{}
				if options != nil {
					opts = options()
				}
				opts.Addr = cli.Host + ":" + cli.Port
				opts.Password = cli.Pwd
				opts.DB = cli.Db
				rdc.Pointer = redis.NewClient(opts)
				atomic.StoreUint32(&rdc.Done, 1)
			}
			rdc.M.Unlock()
		}
		if rdc.Pointer == nil {
			err = fmt.Errorf("new redis(%s:%s) is nil", cli.Host, cli.Port)
		} else {
			err = rdc.Pointer.Ping().Err()
		}
		if err != nil {
			if i < retryMaxTimes {
				rdc.M.Lock()
				if rdc.Done == 1 {
					time.Sleep(retrySleepTime)
					atomic.StoreUint32(&rdc.Done, 0)
				}
				rdc.M.Unlock()
				continue
			} else {
				return nil, fmt.Errorf("new redis(%s:%s) error: %v", cli.Host, cli.Port, err)
			}
		}
		break
	}
	return rdc.Pointer, nil
}
