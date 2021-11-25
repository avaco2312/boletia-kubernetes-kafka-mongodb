package contratos

import (
	"gopkg.in/mgo.v2/bson"
)

type Evento struct {
	Id        bson.ObjectId `bson:"_id"`
	Nombre    string
	Capacidad int
	Categoria string
	Estado    string
}

type Inventario struct {
	Id         bson.ObjectId `bson:"_id"`
	Nombre     string
	Disponible int `bson:"capacidad"`
	Categoria  string
	Estado     string
}

type Reserva struct {
	Id       bson.ObjectId `bson:"_id" json:"_id"`
	Evento   string
	Estado   string
	Email    string
	Cantidad int
}

type DetReserva struct {
	Id       bson.ObjectId `bson:"_id" json:"idres"`
	Evento   string        `json:"nombre"`
	Estado   string
	Email    string
	Cantidad int `json:"canres"`
}
