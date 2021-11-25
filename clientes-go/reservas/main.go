package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/avaco/clientes/contratos"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var sep *mgo.Session

func init() {
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
	r := mux.NewRouter()
	r.HandleFunc("/reservas/eventos", getEventos).Methods("GET")
	r.HandleFunc("/reservas/eventos/{nombre}", getEvento).Methods("GET")
	r.HandleFunc("/reservas/{evento}/{email}", getReservasCliente).Methods("GET")
	r.HandleFunc("/reservas/{id}", getReservaId).Methods("GET")
	r.HandleFunc("/reservas", postReserva).Methods("POST")
	r.HandleFunc("/reservas/{id}", deleteReservaId).Methods("DELETE")
	log.Fatal(http.ListenAndServe(":8071", r))
}

func getEventos(w http.ResponseWriter, r *http.Request) {
	ses := sep.Copy()
	defer ses.Close()
	inventarios := []contratos.Inventario{}
	err := ses.DB("boletia").C("inventario").Find(nil).All(&inventarios)
	if err != nil {
		if err == mgo.ErrNotFound {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=utf8")
	json.NewEncoder(w).Encode(&inventarios)
}

func getEvento(w http.ResponseWriter, r *http.Request) {
	ses := sep.Copy()
	defer ses.Close()
	nombre := mux.Vars(r)["nombre"]
	inventario := contratos.Inventario{}
	err := ses.DB("boletia").C("inventario").Find(bson.M{"nombre": nombre}).One(&inventario)
	if err != nil {
		if err == mgo.ErrNotFound {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=utf8")
	json.NewEncoder(w).Encode(&inventario)
}

func getReservasCliente(w http.ResponseWriter, r *http.Request) {
	ses := sep.Copy()
	defer ses.Close()
	evento := mux.Vars(r)["evento"]
	email := mux.Vars(r)["email"]
	reservas := []contratos.DetReserva{}
	err := ses.DB("boletia").C("reservas").Find(bson.D{{Name: "evento", Value: evento}, {Name: "email", Value: email}}).All(&reservas)
	if err != nil {
		if err == mgo.ErrNotFound {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=utf8")
	json.NewEncoder(w).Encode(&reservas)
}

func getReservaId(w http.ResponseWriter, r *http.Request) {
	id, err := hex.DecodeString(mux.Vars(r)["id"])
	if err != nil || len(id) != 12 {
		http.Error(w, "id incorrecta, el formato es /id/(12 bytes hex)", http.StatusBadRequest)
		return
	}
	ses := sep.Copy()
	defer ses.Close()
	reserva := contratos.Reserva{}
	err = ses.DB("boletia").C("reservas").FindId(bson.ObjectId(id)).One(&reserva)
	if err != nil {
		if err == mgo.ErrNotFound {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=utf8")
	json.NewEncoder(w).Encode(&reserva)
}

func postReserva(w http.ResponseWriter, r *http.Request) {
	ses := sep.Copy()
	defer ses.Close()
	reserva := contratos.Reserva{}
	err := json.NewDecoder(r.Body).Decode(&reserva)
	if err != nil {
		http.Error(w, "JSON no válido", http.StatusBadRequest)
		return
	}
	if reserva.Cantidad <= 0 {
		http.Error(w, "Cantidad incorrecta", http.StatusBadRequest)
		return
	}
	reserva.Id = bson.NewObjectId()
	reserva.Estado = "A"
	sInventario, _ := json.Marshal(&reserva)
	var resp bytes.Buffer
	_ = json.Indent(&resp, sInventario, "", "  ")
	change := mgo.Change{
		Update: bson.D{
			{Name: "$inc", Value: bson.D{{Name: "capacidad", Value: -reserva.Cantidad}}},
			{Name: "$set", Value: bson.D{{Name: "idres", Value: reserva.Id},
				{Name: "email", Value: reserva.Email},
				{Name: "canres", Value: reserva.Cantidad},
			}},
		},
	}
	_, err = ses.DB("boletia").C("inventario").
		Find(bson.D{
			{Name: "nombre", Value: reserva.Evento},
			{Name: "estado", Value: "A"},
			{Name: "capacidad", Value: bson.D{{Name: "$gt", Value: reserva.Cantidad - 1}}},
		}).Apply(change, &reserva)
	if err != nil {
		if err == mgo.ErrNotFound {
			http.Error(w, "evento "+reserva.Evento+" no encontrado o sin capacidad en este momento", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(resp.Bytes())
}

func deleteReservaId(w http.ResponseWriter, r *http.Request) {
	ids := mux.Vars(r)["id"]
	id, err := hex.DecodeString(ids)
	if err != nil || len(id) != 12 {
		http.Error(w, "id incorrecta, el formato es /id/(12 bytes hex)", http.StatusBadRequest)
		return
	}
	ses := sep.Copy()
	defer ses.Close()
	reserva := contratos.Reserva{}
	change := mgo.Change{
		Update: bson.D{
			{Name: "$set", Value: bson.D{{Name: "estado", Value: "X"}}},		
		},
	}
	_, err = ses.DB("boletia").C("reservas").
		Find(bson.D{
			{Name: "_id", Value: bson.ObjectId(id)},
			{Name: "estado", Value: "A"},
		}).Apply(change, &reserva)
	if err != nil {
		if err == mgo.ErrNotFound {
			http.Error(w, "reserva Id "+ids+" no encontrada o ya cancelada", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("reserva Id: " + ids + " Cliente: " + reserva.Email + " Evento: " + reserva.Evento + " cancelada"))
}
