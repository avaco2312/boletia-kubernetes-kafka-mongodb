package pcKafka

import (
	"context"
	"net"
	"strconv"
	"strings"

	kafka "github.com/segmentio/kafka-go"
)

func CreaKafkaWriter(kafkaURL string) *kafka.Writer {
	return &kafka.Writer{
		Addr:     kafka.TCP(kafkaURL),
		Balancer: &kafka.LeastBytes{},
	}
}

func EnviaMensaje(kafkaWriter *kafka.Writer, topic, key, message string) error {
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: []byte(message),
	}
	return kafkaWriter.WriteMessages(context.Background(), msg)
}

func CreaTopico(kafkaURL string, topico string, particiones, replicas int) error {
	conn, err := kafka.Dial("tcp", kafkaURL)
	if err != nil {
		return err
	}
	defer conn.Close()
	controller, err := conn.Controller()
	if err != nil {
		return err
	}
	var controllerConn *kafka.Conn
	controllerConn, err = kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return err
	}
	defer controllerConn.Close()
	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topico,
			NumPartitions:     particiones,
			ReplicationFactor: replicas,
		},
	}
	err = controllerConn.CreateTopics(topicConfigs...)
	return err
}

type RecibeTopico struct {
	Mensaje *kafka.Message
	Reader  *kafka.Reader
}

func RecibeMensajes(kafkaURL, consumergroup string, topicos []string, particiones int, replicas int, mensajes chan<- RecibeTopico, errores chan<- error) error {
	brokers := strings.Split(kafkaURL, ",")
	for _, topico := range topicos {
		err := CreaTopico(kafkaURL, topico, particiones, replicas)
		if err != nil {
			return err
		}
	}
	go func() {
		reader := kafka.NewReader(
			kafka.ReaderConfig{
				Brokers:     brokers,
				GroupTopics: topicos,
				GroupID:     consumergroup,
			},
		)
		for {
			msg, err := reader.FetchMessage(context.Background())
			if err != nil {
				errores <- err
			} else {
				mensajes <- RecibeTopico{Mensaje: &msg, Reader: reader}
			}
		}
	}()
	return nil
}
