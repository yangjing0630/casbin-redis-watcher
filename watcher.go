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
	options  WatcherOptions
	pubConn  *redis.Client //生产消息
	subConn  *redis.Client //订阅消息
	callback func(string)
	closed   chan struct{}
	once     sync.Once
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
func NewWatcher(addr string, setters ...WatcherOption) (persist.Watcher, error) {
	w := &Watcher{
		closed: make(chan struct{}),
	}

	w.options = WatcherOptions{
		Channel:  "/casbin",
		Protocol: "tcp",
	}
	// call destructor when the object is released
	runtime.SetFinalizer(w, finalizer)

	for _, setter := range setters {
		setter(&w.options)
	}

	if err := w.connect(addr); err != nil {
		return nil, err
	}

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
	if _, err := w.pubConn.Publish(w.options.Channel, "casbin rules updated").Result(); err != nil {
		return err
	}
	return nil
	//if _, err := w.pubConn.Do("PUBLISH", w.options.Channel, "casbin rules updated").Result(); err != nil {
	//	return err
	//}
	//return nil
}

// Close disconnects the watcher from redis
func (w *Watcher) Close() {
	finalizer(w)
}

func (w *Watcher) connect(addr string) error {
	if err := w.connectPub(addr); err != nil {
		return err
	}

	if err := w.connectSub(addr); err != nil {
		return err
	}

	return nil
}

func (w *Watcher) connectPub(addr string) error {
	if w.options.PubConn != nil {
		w.pubConn = w.options.PubConn
		return nil
	}
	client := redis.NewClient(
		&redis.Options{
			Addr:     addr,
			Password: w.options.Password,
			DB:       0,
		})

	if _, err := client.Ping().Result(); err != nil {
		fmt.Printf("connectPub error[%s]\n", err.Error())
		return err
	}
	w.pubConn = client
	return nil
}

func (w *Watcher) connectSub(addr string) error {
	if w.options.SubConn != nil {
		w.subConn = w.options.SubConn
		return nil
	}
	client := redis.NewClient(
		&redis.Options{
			Addr:     addr,
			Password: w.options.Password,
			DB:       0,
		})

	if _, err := client.Ping().Result(); err != nil {
		fmt.Printf("connectSub error[%s]\n", err.Error())
		return err
	}

	w.subConn = client
	return nil
}

func (w *Watcher) subscribe() error {
	psc := w.subConn.Subscribe(w.options.Channel)
	//if _, err := psc.Receive(); err != nil {
	//	fmt.Printf("try subscribe channel[%s] error[%s]\n", w.options.Channel, err.Error())
	//	return nil
	//}
	defer psc.Unsubscribe()
	for {
		receive, err := psc.Receive()
		fmt.Println(reflect.TypeOf(receive))
		if err != nil {
			fmt.Printf("try subscribe channel[%s] error[%s]\n", w.options.Channel, err.Error())
			return nil
		}
		switch n := receive.(type) {
		case error:
			return n
		case *redis.Message:
			fmt.Printf("有没有进到这儿呢：%v \n", w.callback)
			if w.callback != nil {
				w.callback("success")
			}
		case *redis.Subscription:
			if n.Count == 0 {
				return nil
			}
		}
	}
	return nil
}

func finalizer(w *Watcher) {
	w.once.Do(func() {
		close(w.closed)
		w.subConn.Close()
		w.pubConn.Close()
	})
}
