package data

import (
	"encoding/json"
	"fmt"

	"example.com/fuel/types"
	"github.com/go-redis/redis"
)

func Connect() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       1,
	})

	return rdb
}

var redisClient = Connect()

// var ctx = context.Background()

func GetClient() *redis.Client {
	return redisClient
}

func marshal(item types.Item) ([]byte, error) {
	return json.Marshal(item)
}

func InsertOrUpdate(key string, item types.Item) bool {
	result, err := marshal(item)

	fmt.Println(key)

	fmt.Println(len(result))

	if err == nil {
		str, err := redisClient.Set(key, "test", 0).Result()
		fmt.Println(str)
		fmt.Println("Error: ", err)
	} else {
		return false
	}

	return true
}
