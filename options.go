package rediswatcher

import (
	"github.com/go-redis/redis"
)

type WatcherOptions struct {
	Channel  string
	PubConn  *redis.Client
	SubConn  *redis.Client
	Password string
	Protocol string
}

type WatcherOption func(*WatcherOptions)

func Channel(subject string) WatcherOption {
	return func(options *WatcherOptions) {
		options.Channel = subject
	}
}

func Password(password string) WatcherOption {
	return func(options *WatcherOptions) {
		options.Password = password
	}
}

func Protocol(protocol string) WatcherOption {
	return func(options *WatcherOptions) {
		options.Protocol = protocol
	}
}

func WithRedisSubConnection(connection *redis.Client) WatcherOption {
	return func(options *WatcherOptions) {
		options.SubConn = connection
	}
}

func WithRedisPubConnection(connection *redis.Client) WatcherOption {
	return func(options *WatcherOptions) {
		options.PubConn = connection
	}
}
