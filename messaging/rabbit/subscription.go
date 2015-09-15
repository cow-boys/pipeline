package rabbit

import (
	"log"
	"strconv"

	"github.com/smartystreets/clock"
	"github.com/smartystreets/pipeline/messaging"
	"github.com/streadway/amqp"
)

type Subscription struct {
	channel       Consumer
	queue         string
	consumer      string
	bindings      []string
	deliveryCount uint64
	latestTag     uint64
	control       chan<- interface{}
	output        chan<- messaging.Delivery
}

func newSubscription(
	channel Consumer, queue string, bindings []string,
	control chan<- interface{}, output chan<- messaging.Delivery,
) *Subscription {
	return &Subscription{
		channel:  channel,
		queue:    queue,
		consumer: strconv.FormatInt(clock.UTCNow().UnixNano(), 10),
		bindings: bindings,
		control:  control,
		output:   output,
	}
}

func (this *Subscription) Listen() {
	input := this.open()
	this.listen(input)
	this.control <- subscriptionClosed{
		DeliveryCount:     this.deliveryCount,
		LatestDeliveryTag: this.latestTag,
		LatestConsumer:    this.channel,
	}
}
func (this *Subscription) listen(input <-chan amqp.Delivery) {
	if input == nil {
		return
	}

	for item := range input {
		this.deliveryCount++
		this.latestTag = item.DeliveryTag
		this.output <- fromAMQPDelivery(item, this.channel)
	}
}
func (this *Subscription) open() <-chan amqp.Delivery {
	this.channel.ConfigureChannelBuffer(cap(this.output))

	if len(this.queue) > 0 {
		if err := this.channel.DeclareQueue(this.queue); err != nil {
			log.Printf("[ERROR] Unable to declare queue [%s]: %s", this.queue, err)
		}

		this.bind(this.queue)
		return this.consume()
	} else {
		this.queue, _ = this.channel.DeclareTransientQueue()
		this.bind(this.queue)
		return this.exclusiveConsume()
	}
}
func (this *Subscription) bind(name string) {
	for _, exchange := range this.bindings {
		if err := this.channel.BindExchangeToQueue(name, exchange); err != nil {
			log.Printf("[ERROR] Unable to bind exchange [%s] to queue [%s]: %s", exchange, name, err)
		}
	}
}

func (this *Subscription) consume() <-chan amqp.Delivery {
	queue, _ := this.channel.Consume(this.queue, this.consumer)
	return queue
}
func (this *Subscription) exclusiveConsume() <-chan amqp.Delivery {
	queue, _ := this.channel.ExclusiveConsume(this.queue, this.consumer)
	return queue
}

func (this *Subscription) Close() {
	this.channel.CancelConsumer(this.consumer)
}
