package main

import (
	"log"
	"net/http"

	"strconv"

	"github.com/gorilla/websocket"

	"github.com/RafaySystems/rafay-prompt/pkg/ptyio"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	//Subprotocols:    []string{"binary"},
}

func main() {

	fs := http.FileServer(http.Dir("../dev"))
	http.Handle("/", fs)

	// websocket handler
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {

		rows := r.URL.Query().Get("rows")
		cols := r.URL.Query().Get("cols")

		rowsUint, err := strconv.ParseUint(rows, 10, 16)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		colsUint, err := strconv.ParseUint(cols, 10, 16)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil) // error ignored for sake of simplicity
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ptyio.New(r.Context(), conn, uint16(rowsUint), uint16(colsUint))

	})

	log.Println("Listening on :7009...")
	err := http.ListenAndServe(":7009", nil)
	if err != nil {
		log.Fatal(err)
	}

}
