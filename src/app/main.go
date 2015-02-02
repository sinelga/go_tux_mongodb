package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"log"
	"net/http"
	//	"strconv"
)

// error response contains everything we need to use http.Error
type handlerError struct {
	Error   error
	Message string
	Code    int
}

type book struct {
	Title  string        `json:"title"`
	Author string        `json:"author"`
	Id     bson.ObjectId `json:"id" bson:"_id,omitempty"`
}

// list of all of the books
var books = make([]book, 0)

var session *mgo.Session
var c *mgo.Collection

var site string

// a custom type that we can use for handling errors and formatting responses
type handler func(w http.ResponseWriter, r *http.Request) (interface{}, *handlerError)

// attach the standard ServeHTTP method to our handler so the http library can call it
func (fn handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// here we could do some prep work before calling the handler if we wanted to
	
	fmt.Println(mux.Vars(r))
	fmt.Println(r.Header)
	fmt.Println(r.Header.Get("Access-Control-Request-Headers"))

	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Content-Type", "application/json")
	
	site = r.Host
	
	session, _ = mgo.Dial("localhost")

	defer session.Close()
	//
	session.SetMode(mgo.Monotonic, true)
		
	c = session.DB("en_USseo").C(site)
	merr := c.Find(nil).All(&books)
	//	err := c.Insert(payload)
	if merr != nil {
		log.Println(merr.Error()) //		return
		//		golog.Err(err.Error())
	}

	response, err := fn(w, r)

	// check for errors
	if err != nil {
		log.Printf("ERROR: %v\n", err.Error)
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Message), err.Code)
		return
	}
	if response == nil {
		log.Printf("ERROR: response from method is nil\n")
		http.Error(w, "Internal server error. Check the logs.", http.StatusInternalServerError)
		return
	}

	// turn the response into JSON
	bytes, e := json.Marshal(response)
	if e != nil {
		http.Error(w, "Error marshalling JSON", http.StatusInternalServerError)
		return
	}
	w.Write(bytes)
	log.Printf("%s %s %s %d", r.RemoteAddr, r.Method, r.URL, 200)
}

func listBooks(w http.ResponseWriter, r *http.Request) (interface{}, *handlerError) {
	fmt.Println(r.Header)
	
	fmt.Println(r.Header.Get("Access-Control-Request-Headers"))

	return books, nil
}

func getBook(w http.ResponseWriter, r *http.Request) (interface{}, *handlerError) {

	//	fmt.Println(mux.Vars(r)["id"])
	var result book
//	c := session.DB("fi_FIporno").C("www.test.com")
	err := c.FindId(bson.ObjectIdHex(mux.Vars(r)["id"])).One(&result)

	if err != nil {
		//		return nil, &handlerError{nil, "Could not find book " + param, http.StatusNotFound}

	}

	return result, nil
}

func parseBookRequest(r *http.Request) (book, *handlerError) {

	data, e := ioutil.ReadAll(r.Body)
	if e != nil {
		return book{}, &handlerError{e, "Could not read request", http.StatusBadRequest}
	}

	// turn the request body (JSON) into a book object
	var payload book
	e = json.Unmarshal(data, &payload)
	if e != nil {
		return book{}, &handlerError{e, "Could not parse JSON", http.StatusBadRequest}
	}

	return payload, nil
}

func addBook(w http.ResponseWriter, r *http.Request) (interface{}, *handlerError) {

	payload, e := parseBookRequest(r)
	if e != nil {
		return nil, e
	}

	// it's our job to assign IDs, ignore what (if anything) the client sent
	//	payload.Id = getNextId()
	books = append(books, payload)

	err := c.Insert(payload)
	if err != nil {

		//		return
		//		golog.Err(err.Error())
	}

	// we return the book we just made so the client can see the ID if they want
	return payload, nil
}

func updateBook(w http.ResponseWriter, r *http.Request) (interface{}, *handlerError) {
	payload, e := parseBookRequest(r)
	if e != nil {
		return nil, e
	}

	change := bson.M{"title": payload.Title, "autor": payload.Author}

	err := c.UpdateId(payload.Id, change)
	if err != nil {
		fmt.Println(err.Error())
		//		return nil, &handlerError{nil, "Could not find book " + param, http.StatusNotFound}

	}

	return make(map[string]string), nil
}

func removeBook(w http.ResponseWriter, r *http.Request) (interface{}, *handlerError) {

	var index int

	for i, book := range books {

		if book.Id == bson.ObjectIdHex(mux.Vars(r)["id"]) {
			index = i

		}

	}

	books = append(books[:index], books[index+1:]...)

	err := c.RemoveId(bson.ObjectIdHex(mux.Vars(r)["id"]))

	if err != nil {
		fmt.Println(err.Error())
		//		return nil, &handlerError{nil, "Could not find book " + param, http.StatusNotFound}

	}

	return make(map[string]string), nil
}


func corOptions(w http.ResponseWriter, r *http.Request) (interface{}, *handlerError) {

	return make(map[string]string), nil

}


func main() {
	// command line flags
	port := flag.Int("port", 80, "port to serve on")
	dir := flag.String("directory", "web/", "directory of web files")
	flag.Parse()

	// handle all requests by serving a file of the same name
	fs := http.Dir(*dir)
	fileHandler := http.FileServer(fs)

	// setup routes
	router := mux.NewRouter()

	router.Handle("/", http.RedirectHandler("/static/", 302))
	router.Handle("/books", handler(listBooks)).Methods("GET")
	router.Handle("/books", handler(addBook)).Methods("POST")
	router.Handle("/books/{id}", handler(getBook)).Methods("GET")
	router.Handle("/books/{id}", handler(updateBook)).Methods("POST")
	router.Handle("/books/{id}", handler(removeBook)).Methods("DELETE")
	router.Handle("/books/{id}", handler(corOptions)).Methods("OPTIONS")
	router.Handle("/books", handler(corOptions)).Methods("OPTIONS")
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static", fileHandler))
	http.Handle("/", router)



	log.Printf("Running on port %d\n", *port)

	addr := fmt.Sprintf("104.236.237.125:%d", *port)
	// this call blocks -- the progam runs here forever
	err := http.ListenAndServe(addr, nil)
	fmt.Println(err.Error())
}
