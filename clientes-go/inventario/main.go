package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"

	"github.com/avaco/clientes/contratos"
	"github.com/avaco/clientes/pcKafka"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var sep *mgo.Session
var kafkaURL string

func init() {
	kafkaURL = os.Getenv("KAFKA_URL")
	mongoURL := os.Getenv("MONGO_URL")
	var err error
	sep, err = mgo.Dial(mongoURL)
	if err != nil {
		log.Fatal(err)
	}
	err = sep.DB("boletia").C("inventario").EnsureIndex(
		mgo.Index{
			Key:    []string{"nombre"},
			Unique: true,
		},
	)
	if err != nil {
		log.Fatal(err)
	}	
	err = sep.DB("boletia").C("reservas").EnsureIndex(
		mgo.Index{
			Key:    []string{"evento", "email"},
			Unique: false,
		},
	)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	par, _ := strconv.Atoi(os.Getenv("KAFKA_TOPIC_PARTITIONS"))
	rep, _ := strconv.Atoi(os.Getenv("KAFKA_TOPIC_REPLICAS"))
	mensajes := make(chan pcKafka.RecibeTopico)
	errores := make(chan error)
	err := pcKafka.RecibeMensajes(kafkaURL, "inventario", []string{"boletia.inventario", "boletia.reservas"}, par, rep, mensajes, errores)
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
	switch topico {
	case "boletia.inventario":
		reserva := contratos.DetReserva{}
		err = json.Unmarshal([]byte(b), &reserva)
		if err != nil {
			return err
		}
		ses := sep.Copy()
		defer ses.Close()
		switch reserva.Estado {
		case "A":
			switch {
			case reserva.Cantidad == 0:
				return nil
			case reserva.Cantidad > 0:
				err = ses.DB("boletia").C("reservas").Insert(&reserva)
				if err != nil {
					if mgo.IsDup(err) {
						return nil
					}
				}
				return err
			default:
				return nil
			}
		case "C":
			_, err = ses.DB("boletia").C("reservas").UpdateAll(
				bson.D{{Name: "evento", Value: reserva.Evento}, {Name: "estado", Value: "A"}},
				bson.D{{Name: "$set", Value: bson.D{{Name: "estado", Value: "C"}}}})
			return err
		}
	case "boletia.reservas":
		reserva := contratos.Reserva{}
		err = json.Unmarshal([]byte(b), &reserva)
		if err != nil {
			return err
		}
		if reserva.Estado != "X" {
			return nil
		}
		ses := sep.Copy()
		defer ses.Close()
		change := mgo.Change{
			Update: bson.D{
				{Name: "$inc", Value: bson.D{{Name: "capacidad", Value: reserva.Cantidad}}},
				{Name: "$set", Value: bson.D{{Name: "idres", Value: reserva.Id},
					{Name: "email", Value: reserva.Email},
					{Name: "canres", Value: -1},
				}},				
			},
		}
		_, err = ses.DB("boletia").C("inventario").
			Find(bson.D{
				{Name: "nombre", Value: reserva.Evento},
				{Name: "estado", Value: "A"},
			}).Apply(change, &reserva)
		if err == mgo.ErrNotFound {
			return nil
		}
		return err
	}
	log.Printf("mensaje imposible %s del topico %s\n", msg, topico)
	return nil
}
