package test1interestbasedstreaminjetstream

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestMain(m *testing.T) {

	url := "nats://127.0.0.1:4223"

	nc, _ := nats.Connect(url)
	defer nc.Drain()

	js, _ := jetstream.New(nc)

	head := "events1.12"
	cfg := jetstream.StreamConfig{
		Name:      "EVENTS12",
		Retention: jetstream.LimitsPolicy,
		MaxMsgs:   20,
		Subjects:  []string{head + ".>"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	stream, err := js.CreateStream(ctx, cfg)
	if err != nil {
		panic(err)
	}
	fmt.Println("created the stream")

	data := []byte(time.Now().Format(time.RFC3339))
	js.Publish(ctx, head+".1.page_loaded", data)
	js.Publish(ctx, head+".1.mouse_clicked", data)
	ack, err := js.Publish(ctx, head+".1.input_focused", nil)
	if err != nil {
		panic(err)
	}
	fmt.Println("published 3 messages")

	fmt.Printf("last message seq: %d\n", ack.Sequence)

	fmt.Println("# Stream info without any consumers")
	printStreamState(ctx, stream)

	// 加入消費者
	cons, _ := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "processor-1",
		AckPolicy: jetstream.AckExplicitPolicy,
	})

	data = []byte(time.Now().Format(time.RFC3339))
	js.Publish(ctx, head+".1.mouse_clicked", data)
	js.Publish(ctx, head+".1.input_focused", data)

	fmt.Println("\n# Stream info with one consumer")
	printStreamState(ctx, stream)

	msgs, _ := cons.Fetch(2)
	for msg := range msgs.Messages() {
		msg.DoubleAck(ctx)
	}

	// 更多消費者
	cons2, _ := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       "processor-2",
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: head + ".2.>",
	})

	data = []byte(time.Now().Format(time.RFC3339))
	js.Publish(ctx, head+".2.input_focused", data)
	js.Publish(ctx, head+".2.mouse_clicked", data)

	msgs, _ = cons2.Fetch(2)
	var msgsMeta []*jetstream.MsgMetadata
	for msg := range msgs.Messages() {
		msg.DoubleAck(ctx)
		meta, _ := msg.Metadata()
		msgsMeta = append(msgsMeta, meta)
	}

	fmt.Printf("msg seqs %d and %d", msgsMeta[0].Sequence.Stream, msgsMeta[1].Sequence.Stream)

	fmt.Println("\n# Stream info with two consumers, but only one set of acked messages")
	printStreamState(ctx, stream)

	msgs, _ = cons.Fetch(2)
	for msg := range msgs.Messages() {
		msg.DoubleAck(ctx)
	}

	fmt.Println("\n# Stream info with two consumers having both acked")
	printStreamState(ctx, stream)

	cons3, _ := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       "processor-3",
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: head + ".*.mouse_clicked",
	})

	data = []byte(time.Now().Format(time.RFC3339))
	js.Publish(ctx, head+".3.input_focused", data)

	for {
		msgs, _ = cons3.Fetch(1)
		msg := <-msgs.Messages()
		msg.DoubleAck(ctx)
	}

	msgs, _ = cons.Fetch(1)
	msg := <-msgs.Messages()
	msg.Term()
	msgs, _ = cons2.Fetch(1)
	msg = <-msgs.Messages()
	msg.DoubleAck(ctx)

	fmt.Println("\n# Stream info with three consumers with interest from two")
	printStreamState(ctx, stream)
}

func printStreamState(ctx context.Context, stream jetstream.Stream) {
	info, _ := stream.Info(ctx)
	b, _ := json.MarshalIndent(info.State, "", " ")
	fmt.Println(string(b))
}
