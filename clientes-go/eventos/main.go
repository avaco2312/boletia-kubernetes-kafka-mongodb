package main

import (
	"bytes"
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
	err = sep.DB("boletia").C("eventos").EnsureIndex(
		mgo.Index{
			Key:    []string{"nombre"},
			Unique: true,
		},
	)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/eventos", getEventos).Methods("GET")
	r.HandleFunc("/eventos/{nombre}", getEvento).Methods("GET")
	r.HandleFunc("/eventos", postEvento).Methods("POST")
	r.HandleFunc("/eventos/{nombre}", deleteEvento).Methods("DELETE")
	log.Fatal(http.ListenAndServe(":8070", r))
}

func deleteEvento(w http.ResponseWriter, r *http.Request) {
	ses := sep.Copy()
	defer ses.Close()
	nombre := mux.Vars(r)["nombre"]
	err := ses.DB("boletia").C("eventos").Update(bson.M{"nombre": nombre}, bson.D{{Name: "$set", Value: bson.D{{Name: "estado", Value: "C"}}}})
	if err != nil {
		if err == mgo.ErrNotFound {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Evento: "+nombre+" cancelado"))
}

func postEvento(w http.ResponseWriter, r *http.Request) {
	ses := sep.Copy()
	defer ses.Close()
	evento := contratos.Evento{}
	err := json.NewDecoder(r.Body).Decode(&evento)
	if err != nil {
		http.Error(w, "JSON no válido", http.StatusBadRequest)
		return
	}
	evento.Id = bson.NewObjectId()
	evento.Estado = "A"
	sEvento, _ := json.Marshal(&evento)
	var resp bytes.Buffer
	_ = json.Indent(&resp, sEvento, "", "  ")
	err = ses.DB("boletia").C("eventos").Insert(&evento)
	if err != nil {
		if mgo.IsDup(err) {
			http.Error(w, "Evento "+evento.Nombre+" ya existente", http.StatusBadRequest)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Write(resp.Bytes())
}

func getEvento(w http.ResponseWriter, r *http.Request) {
	ses := sep.Copy()
	defer ses.Close()
	nombre := mux.Vars(r)["nombre"]
	evento := contratos.Evento{}
	err := ses.DB("boletia").C("eventos").Find(bson.M{"nombre": nombre}).One(&evento)
	if err != nil {
		if err == mgo.ErrNotFound {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=utf8")
	json.NewEncoder(w).Encode(&evento)
}

func getEventos(w http.ResponseWriter, r *http.Request) {
	ses := sep.Copy()
	defer ses.Close()
	eventos := []contratos.Evento{}
	err := ses.DB("boletia").C("eventos").Find(nil).All(&eventos)
	if err != nil {
		if err == mgo.ErrNotFound {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=utf8")
	json.NewEncoder(w).Encode(&eventos)
}

