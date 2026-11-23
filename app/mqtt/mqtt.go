// Package mqtt is the gateway's MQTT connection. It replaces the connection
// handling of mqtt-gateway, which exits the process when the broker is not
// reachable at startup and backs off up to 10 minutes between reconnects.
//
// Here the broker is retried every RetryInterval until it is reachable, and a
// broken connection is re-established automatically with the same interval
// as the upper bound for paho's backoff.
package mqtt

import (
	"context"
	"math/rand"
	"sync"
	"time"
	"unicode/utf8"

	PAHO "github.com/eclipse/paho.mqtt.golang"
	"github.com/philipparndt/go-logger"
	gwconfig "github.com/philipparndt/mqtt-gateway/config"
)

const (
	statusOnline  = "online"
	statusOffline = "offline"
)

// RetryInterval is the delay between connection attempts, both while waiting
// for the broker at startup and while reconnecting after a lost connection.
var RetryInterval = 5 * time.Second

// publishTimeout bounds how long a publish may block while the broker is gone,
// so callers (the polling loop, SSE handlers) are never stuck on an outage.
const publishTimeout = 10 * time.Second

var (
	mu          sync.Mutex
	client      PAHO.Client
	cfg         gwconfig.MQTTConfig
	onReconnect func()
)

// SetOnReconnect registers a callback that runs (on its own goroutine) every
// time the connection is re-established after it was lost. Use it to republish
// state that may have been dropped during the outage.
func SetOnReconnect(f func()) {
	mu.Lock()
	defer mu.Unlock()
	onReconnect = f
}

// Start connects to the broker and blocks until the first connection succeeds,
// retrying every RetryInterval. It returns ctx.Err() if ctx is cancelled before
// a connection could be made.
func Start(ctx context.Context, config gwconfig.MQTTConfig, clientIdPrefix string) error {
	statusTopic := config.Topic + "/bridge/state"
	clientID := clientIdPrefix + "_" + generateRandomClientID(10)
	logger.Debug("Generated client ID", "clientID", clientID)

	connected := false

	opts := PAHO.NewClientOptions().
		AddBroker(config.URL).
		SetClientID(clientID).
		SetWill(statusTopic, statusOffline, 1, true).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(RetryInterval).
		SetConnectTimeout(RetryInterval)
	opts.Username = config.Username
	opts.Password = config.Password

	opts.SetOnConnectHandler(func(_ PAHO.Client) {
		mu.Lock()
		reconnect := connected
		connected = true
		cb := onReconnect
		mu.Unlock()

		// Off the handler goroutine: waiting for publish/subscribe tokens
		// inside a paho callback can block the acks they wait for.
		go func() {
			if reconnect {
				logger.Info("Reconnected to MQTT broker", "url", config.URL)
			} else {
				logger.Info("Connected to MQTT broker", "url", config.URL)
			}
			// Always retained, whatever cfg.Retain says: the will is retained.
			PublishAbsolute(statusTopic, statusOnline, true)
			guardStatus(statusTopic)
			if reconnect && cb != nil {
				cb()
			}
		}()
	})
	opts.SetConnectionLostHandler(func(_ PAHO.Client, err error) {
		logger.Error("MQTT connection lost, reconnecting", "error", err)
	})
	opts.SetReconnectingHandler(func(_ PAHO.Client, _ *PAHO.ClientOptions) {
		logger.Debug("Trying to reconnect to MQTT broker", "url", config.URL)
	})

	c := PAHO.NewClient(opts)
	mu.Lock()
	client = c
	cfg = config
	mu.Unlock()

	for {
		token := c.Connect()
		token.Wait()
		if token.Error() == nil {
			return nil
		}
		logger.Error("MQTT broker not available, retrying",
			"url", config.URL, "error", token.Error(), "retryIn", RetryInterval)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(RetryInterval):
		}
	}
}

// Stop ends the MQTT session cleanly, waiting up to 250 ms for in-flight work.
// A clean DISCONNECT discards the will, so a planned shutdown does not
// overwrite the status topic of a successor that already announced "online".
func Stop() {
	c := getClient()
	if c == nil || !c.IsConnectionOpen() {
		return
	}
	logger.Info("Disconnecting from MQTT broker")
	c.Disconnect(250)
}

// guardStatus re-asserts "online" whenever our own retained status topic says
// "offline" while we are connected (e.g. the will of a previous instance that
// died after we announced ourselves). Subscriptions do not survive a clean
// session reconnect, so this runs on every connect.
func guardStatus(statusTopic string) {
	c := getClient()
	c.Subscribe(statusTopic, 1, func(_ PAHO.Client, message PAHO.Message) {
		if string(message.Payload()) != statusOffline {
			return
		}
		go func() {
			logger.Info("Status topic says offline while we are connected, correcting", "topic", statusTopic)
			PublishAbsolute(statusTopic, statusOnline, true)
		}()
	})
}

// PublishAbsolute publishes message to topic. While the broker is unreachable
// the message is dropped (and logged) instead of blocking the caller.
func PublishAbsolute(topic string, message any, retained bool) {
	c := getClient()
	if c == nil {
		logger.Warn("MQTT not started, dropping message", "topic", topic)
		return
	}
	if !c.IsConnectionOpen() {
		logger.Debug("MQTT not connected, dropping message", "topic", topic)
		return
	}

	token := c.Publish(topic, cfg.QoS, retained, message)
	if !token.WaitTimeout(publishTimeout) {
		logger.Error("Timeout publishing message", "topic", topic)
		return
	}
	if token.Error() != nil {
		logger.Error("Error publishing message", "topic", topic, "error", token.Error())
		return
	}

	if size, ok := payloadSize(message); ok && size > 100 {
		logger.Debug("Published message", "topic", topic, "bytes", size)
	} else {
		logger.Debug("Published message", "topic", topic, "message", logMessage(message))
	}
}

func getClient() PAHO.Client {
	mu.Lock()
	defer mu.Unlock()
	return client
}

func generateRandomClientID(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(result)
}

func logMessage(message any) any {
	if b, ok := message.([]byte); ok && utf8.Valid(b) {
		return string(b)
	}
	return message
}

func payloadSize(message any) (int, bool) {
	switch v := message.(type) {
	case []byte:
		return len(v), true
	case string:
		return len(v), true
	default:
		return 0, false
	}
}
