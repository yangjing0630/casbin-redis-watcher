package rediswatcher

import (
	"fmt"
	"github.com/casbin/casbin/v2/persist"
	"reflect"
	"runtime"
	"sync"

	"github.com/go-redis/redis"
)

type Watcher struct {
	options     WatcherOptions
	pubConn redis.Client
	subConn redis.Client
	callback    func(string)
	closed      chan struct{}
	once        sync.Once
	redisClient *redis.Client
}

//NewWatcher creates a new Watcher to be used with a Casbin enforcer
//addr is a redis target string in the format "host:port"
//setters allows for inline WatcherOptions
//
//Example:
//w, err := rediswatcher.NewWatcher("127.0.0.1:6379", rediswatcher.Password("pass"), rediswatcher.Channel("/yourchan"))
//
//A custom redis.Conn can be provided to NewWatcher
//
//Example:
//c, err := redis.Dial("tcp", ":6379")
//w, err := rediswatcher.NewWatcher("", rediswatcher.WithRedisConnection(c)
////
func NewWatcher(redisClient *redis.Client) (persist.Watcher, error) {
	w := &Watcher{
		closed: make(chan struct{}),
	}

	w.options = WatcherOptions{
		Channel:  "/casbin",
		Protocol: "tcp",
	}
	// call destructor when the object is released
	runtime.SetFinalizer(w, finalizer)

	go func() {
		for {
			select {
			case <-w.closed:
				return
			default:
				err := w.subscribe()
				if err != nil {
					fmt.Printf("Failure from Redis subscription: %v", err)
				}
			}
		}
	}()

	return w, nil
}

// SetUpdateCallBack sets the update callback function invoked by the watcher
// when the policy is updated. Defaults to Enforcer.LoadPolicy()
func (w *Watcher) SetUpdateCallback(callback func(string)) error {
	w.callback = callback
	return nil
}

// Update publishes a message to all other casbin instances telling them to
// invoke their update callback
func (w *Watcher) Update() error {
	return Publish(w.redisClient, w.options.Channel, "casbin rules updated")
	//if _, err := w.pubConn.Do("PUBLISH", w.options.Channel, "casbin rules updated").Result(); err != nil {
	//	return err
	//}
	//
	//return nil
}

// Close disconnects the watcher from redis
func (w *Watcher) Close() {
	finalizer(w)
}

func (w *Watcher) subscribe() error {
	psc := w.redisClient.Subscribe(w.options.Channel)
	receive, err := psc.Receive()
	if err != nil {
		return err
	}
	defer psc.Unsubscribe()

	for {
		fmt.Println(reflect.TypeOf(receive))
		//switch n := receive.(type) {
		//
		//case error:
		//	return n
		//case redis.Message:
		//	if w.callback != nil {
		//		w.callback(string(n.Data))
		//	}
		//case redis.Subscription:
		//	if n.Count == 0 {
		//		return nil
		//	}
		//}
	}
	return nil
}

func finalizer(w *Watcher) {
	w.once.Do(func() {
		close(w.closed)
		w.redisClient.Close()
		//w.subConn.Close()
		//w.pubConn.Close()
	})
}

func Publish(redisClient *redis.Client, channel string, msg string) error {
	var err error
	fmt.Printf("Will publish message [%v] to channel [%v]\n", msg, channel)

	err = redisClient.Publish(channel, msg).Err()
	if err != nil {
		fmt.Printf("try publish message to channel[test_channel] error[%s]\n",
			err.Error())
		return err
	}
	return nil
}
