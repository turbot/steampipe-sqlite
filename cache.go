package main

import (
	"log"
	"os"
	"time"

	"github.com/turbot/go-kit/types"
)

func cacheEnabled() bool {
	log.Println("[DEBUG] cacheEnabled")
	defer log.Println("[DEBUG] end cacheEnabled")
	if envStr, ok := os.LookupEnv(EnvCacheEnabled); ok {
		toBool, err := types.ToBool(envStr)
		if err == nil {
			return toBool
		}
	}
	return true
}

func cacheTTL() int64 {
	log.Println("[DEBUG] cacheTTL")
	defer log.Println("[DEBUG] end cacheTTL")
	if envStr, ok := os.LookupEnv(EnvCacheMaxTTL); ok {
		i64, err := types.ToInt64(envStr)
		if err == nil {
			return i64
		}
	}
	return (int64((10 * time.Hour).Seconds()))
}
