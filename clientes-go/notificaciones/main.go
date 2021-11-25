package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"context"
	"encoding/json"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/avaco/clientes/contratos"
	"github.com/avaco/clientes/pcKafka"
)

var svc *ses.SES

var Sender string

func init() {
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)
	svc = ses.New(sess)
}

func main() {
	Sender = os.Getenv("SENDER_EMAIL")
	kafkaURL := os.Getenv("KAFKA_URL")
	par, _ := strconv.Atoi(os.Getenv("KAFKA_TOPIC_PARTITIONS"))
	rep, _ := strconv.Atoi(os.Getenv("KAFKA_TOPIC_REPLICAS"))
	mensajes := make(chan pcKafka.RecibeTopico)
	errores := make(chan error)
	err := pcKafka.RecibeMensajes(kafkaURL, "notificaciones", []string{"boletia.reservas"}, par, rep, mensajes, errores)
	if err != nil {
		log.Fatal(err)
	}
	for {
		select {
		case m := <-mensajes:
			err = procesaMensajes(m.Mensaje.Topic, m.Mensaje.Value)
			if err == nil {
				m.Reader.CommitMessages(context.Background(), *m.Mensaje)
			}
		case err = <-errores:
			log.Printf("error recibiendo: %s\n", err)
		}
	}
}

func procesaMensajes(topico string, msg []byte) error {
	b, err := strconv.Unquote(string(msg))
	if err != nil {
		return err
	}
	reserva := contratos.Reserva{}
	err = json.Unmarshal([]byte(b), &reserva)
	if err != nil {
		return err
	}
	return enviaEmail(reserva.Id.Hex(), reserva.Evento, reserva.Estado, reserva.Email, reserva.Cantidad)
}

const 	CharSet = "UTF-8"

type mensaje struct {
	subject  string
	textbody string
}

var mensajes = [...]mensaje{
	{
		subject:  "Confirmación de reserva",
		textbody: "Su reserva %s de %d boletos para el evento %s está confirmada",
	},
	{
		subject:  "Cancelación de reserva",
		textbody: "Su reserva %s de %d boletos para el evento %s fue cancelada, el evento fue suspendido por los organizadores",
	},
	{
		subject:  "Cancelación de reserva",
		textbody: "Su reserva %s de %d boletos para el evento %s fue cancelada a petición suya",
	},
}

func enviaEmail(id, evento, estado, email string, cantidad int) error {
	tipo := strings.Index("ACX", estado)
	if tipo == -1 {
		return errors.New("estado de la reserva no valido")
	}
	msg := fmt.Sprintf(mensajes[tipo].textbody, id, cantidad, evento)
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{
				aws.String(email),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Text: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(msg),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(CharSet),
				Data:    aws.String(mensajes[tipo].subject),
			},
		},
		Source: aws.String(Sender),
	}
	_, err := svc.SendEmail(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ses.ErrCodeMessageRejected:
				log.Println(ses.ErrCodeMessageRejected, aerr.Error())
			case ses.ErrCodeMailFromDomainNotVerifiedException:
				log.Println(ses.ErrCodeMailFromDomainNotVerifiedException, aerr.Error())
			case ses.ErrCodeConfigurationSetDoesNotExistException:
				log.Println(ses.ErrCodeConfigurationSetDoesNotExistException, aerr.Error())
			default:
				log.Println(aerr.Error())
			}
			return nil
		} else {
			return err
		}
	}
	return nil
}
