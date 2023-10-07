package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	mynats "nats-test/my-nats"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {

	urls := "nats://127.0.0.1:4223"
	cmds := "4"
	for idx := range os.Args {
		switch idx {
		case 1:
			cmds = os.Args[idx]
		case 2:
			urls = os.Args[idx]
		}
	}

	fmt.Println(cmds, urls)

	switch cmds {
	case "1":
		pub(urls)
	case "2":
		sub(urls)
	case "3":
		jet_pub(urls)
	case "4":
		jet_sub(urls)
	}
}

func pub(urls string) {
	conn, err := mynats.New(urls)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	_ = conn.Publish("hello1", []byte(time.Now().Format(time.RFC3339)))
}

func sub(urls string) {
	conn, err := mynats.New(urls)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	// Subscribe
	sub, err := conn.SubscribeSync("hello1")
	if err != nil {
		log.Fatal(err)
	}

	for {
		// Wait for a message
		msg, err := sub.NextMsg(time.Minute)

		if errors.Is(nats.ErrTimeout, err) {
			continue
		} else if err != nil {
			log.Fatalln(err)
		}

		// Use the response
		log.Printf("Reply: %s", msg.Data)
	}
}

func jet_pub(urls string) {
	// 連線
	natsConn, err := nats.Connect(urls)
	if err != nil {
		log.Fatal("連不上 NATS")
	}
	defer natsConn.Drain()
	// defer natsConn.Close()

	// 取得 JetStream 的 Context
	js, err := natsConn.JetStream()
	if err != nil {
		log.Fatalf("取得 JetStream 的 Context 失敗: %v", err)
	}

	_, _ = js.Publish("hello1", []byte(time.Now().Format(time.RFC3339)))
}

func jet_sub(urls string) {
	// 連線
	natsConn, err := nats.Connect(urls)
	if err != nil {
		log.Fatal("連不上 NATS")
	}
	defer natsConn.Drain()

	// 新版API
	{
		newJS, _ := jetstream.New(natsConn)

		// 設定連線timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 新 Stream 設定
		// cfg := jetstream.StreamConfig{
		// 	Name:      "HELLO_ALL",
		// 	Retention: jetstream.WorkQueuePolicy,
		// 	Storage:   jetstream.FileStorage,
		// 	MaxMsgs:   20,
		// 	Subjects:  []string{"hello*"},
		// }

		// 取得 stream 物件
		stream, err := newJS.Stream(ctx, "HELLO1-3")
		if err != nil {
			panic(err)
		}

		// 建立臨時消費者
		// cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		// 	InactiveThreshold: 10 * time.Millisecond,
		// })
		// if err != nil {
		// 	panic(err)
		// }

		// fmt.Println("Created consumer", cons.CachedInfo().Name)
		// fmt.Println("# Consume messages using Consume()")

		// // 設定消費者訊息讀取
		// consumeContext, _ := cons.Consume(func(msg jetstream.Msg) {
		// 	fmt.Printf("received %s %s\n", msg.Subject(), string(msg.Data()))
		// 	_ = msg.Ack()
		// })
		// defer consumeContext.Stop()

		// 建立持久化消費者
		// consumerName := "pull-1"
		// cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		// 	Name: consumerName,
		// 	// InactiveThreshold: 10 * time.Minute,
		// 	FilterSubject: "HELLO1-3",
		// })
		// if err != nil {
		// 	panic(err)
		// }
		// fmt.Println("Created consumer", cons.CachedInfo().Name)
		// fmt.Println("# Consume messages using Messages()")

		// 訊號 loop
		// it, err := cons.Messages()
		// if err != nil {
		// 	it.Stop()
		// 	panic(err)
		// }
		// for {
		// 	msg1, err := it.Next()
		// 	if err != nil {
		// 		it.Stop()
		// 		panic(err)
		// 	}
		// 	fmt.Printf("received %q %s\n", msg1.Subject(), string(msg1.Data()))
		// }

		cons, _ := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
			InactiveThreshold: 10 * time.Millisecond,
			FilterSubject:     "hello1",
		})
		fetchResult, _ := cons.Fetch(1, jetstream.FetchMaxWait(100*time.Millisecond))
		for msg := range fetchResult.Messages() {
			fmt.Printf("received %q\n", msg.Subject())
			msg.Ack()
		}

	}

	// 舊版API
	{
		// 取得 JetStream 的 Context
		// js, err := natsConn.JetStream()
		// if err != nil {
		// 	log.Fatalf("取得 JetStream 的 Context 失敗: %v", err)
		// }
		// // 建立 Stream
		// _, err = js.AddStream(&nats.StreamConfig{
		// 	Name: "STEAM_hello",
		// 	Subjects: []string{
		// 		"hello*", // 支援 wildcard
		// 	},
		// 	Storage:   nats.FileStorage,  // 儲存的方式 (預設 FileStorage)
		// 	Retention: nats.LimitsPolicy, // 保留的策略
		// 	Discard:   nats.DiscardOld,   // 丟棄的策略
		// 	MaxMsgs:   20,                // 保留訊息數量
		// 	// ...
		// })
		// if err != nil {
		// 	log.Fatalf("建立 Stream 失敗: %v", err)
		// }
		// _, err = js.Subscribe("hello*", func(msg *nats.Msg) {
		// 	fmt.Println("收到了", string(msg.Data))
		// })
		// if err != nil {
		// 	log.Fatal("訂閱失敗", err)
		// }
	}

	select {}
}

func jet_pull_sub(urls string) {
	// 連線
	natsConn, err := nats.Connect(urls)
	if err != nil {
		log.Fatal("連不上 NATS")
	}
	defer natsConn.Drain()
	// defer natsConn.Close()

	// 取得 JetStream 的 Context
	js, err := natsConn.JetStream()
	if err != nil {
		log.Fatalf("取得 JetStream 的 Context 失敗: %v", err)
	}

	// 建立 Stream
	_, err = js.AddStream(&nats.StreamConfig{
		Name: "STEAM_hello",
		Subjects: []string{
			"hello*", // 支援 wildcard
		},
		Storage:   nats.FileStorage,  // 儲存的方式 (預設 FileStorage)
		Retention: nats.LimitsPolicy, // 保留的策略
		Discard:   nats.DiscardOld,   // 丟棄的策略
		MaxMsgs:   20,                // 保留訊息數量
		// ...
	})
	if err != nil {
		log.Fatalf("建立 Stream 失敗: %v", err)
	}

	_, err = js.PullSubscribe("hello*", "", nats.Durable(""))
	if err != nil {
		log.Fatal("訂閱失敗", err)
	}

	select {}
}
