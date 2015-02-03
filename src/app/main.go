package main

import (
	"domains"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	//	"strconv"
)

// error response contains everything we need to use http.Error
type handlerError struct {
	Error   error
	Message string
	Code    int
}

var blogs = make([]domains.Blog, 0)

var session *mgo.Session
var c *mgo.Collection

var site string

// a custom type that we can use for handling errors and formatting responses
type handler func(w http.ResponseWriter, r *http.Request) (interface{}, *handlerError)

// attach the standard ServeHTTP method to our handler so the http library can call it
func (fn handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// here we could do some prep work before calling the handler if we wanted to

	//	fmt.Println(mux.Vars(r))
	//	fmt.Println(r.Header)
	//	fmt.Println(r.Header.Get("Access-Control-Request-Headers"))

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
	merr := c.Find(nil).All(&blogs)
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

func listBlogs(w http.ResponseWriter, r *http.Request) (interface{}, *handlerError) {

	return blogs, nil
}

func getBlog(w http.ResponseWriter, r *http.Request) (interface{}, *handlerError) {
	var result domains.Blog

	err := c.FindId(bson.ObjectIdHex(mux.Vars(r)["id"])).One(&result)

	if err != nil {
		//		return nil, &handlerError{nil, "Could not find book " + param, http.StatusNotFound}

	}

//	fmt.Println(result)

	return result, nil
}

func parseBookRequest(r *http.Request) (domains.Blog, *handlerError) {

	data, e := ioutil.ReadAll(r.Body)
	if e != nil {
		//		return book{}, &handlerError{e, "Could not read request", http.StatusBadRequest}
	}

	var payload domains.Blog
	e = json.Unmarshal(data, &payload)
	if e != nil {
		return domains.Blog{}, &handlerError{e, "Could not parse JSON", http.StatusBadRequest}
	}

	//	payload.Pubdate = time.Now()

	return payload, nil
}

func addBook(w http.ResponseWriter, r *http.Request) (interface{}, *handlerError) {

	payload, e := parseBookRequest(r)
	if e != nil {
		return nil, e
	}

	// it's our job to assign IDs, ignore what (if anything) the client sent
	//	payload.Id = getNextId()
	blogs = append(blogs, payload)

	payload.Pubdate = time.Now().Local()

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

	fmt.Println("updateBook", payload)
	
	
	change := bson.M{"title": payload.Title,
		"author":        payload.Author,
		"contents":      payload.Contents,
		"permanentlink": payload.Permanentlink,
		"imglink":       payload.Imglink,
		"extlink":       payload.Extlink,
		"pubdate":       payload.Pubdate,
		"keywords":      payload.Keywords,
		"tags":          payload.Tags,
	}

	err := c.UpdateId(payload.Id, change)
	if err != nil {
		fmt.Println(err.Error())
		//		return nil, &handlerError{nil, "Could not find book " + param, http.StatusNotFound}

	}

	return make(map[string]string), nil
}

func removeBook(w http.ResponseWriter, r *http.Request) (interface{}, *handlerError) {

	var index int

	for i, blog := range blogs {

		if blog.Id == bson.ObjectIdHex(mux.Vars(r)["id"]) {
			index = i

		}

	}

	blogs = append(blogs[:index], blogs[index+1:]...)

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
	router.Handle("/blogs", handler(listBlogs)).Methods("GET")
	router.Handle("/blogs", handler(addBook)).Methods("POST")
	router.Handle("/blogs/{id}", handler(getBlog)).Methods("GET")
	router.Handle("/blogs/{id}", handler(updateBook)).Methods("POST")
	router.Handle("/blogs/{id}", handler(removeBook)).Methods("DELETE")
	router.Handle("/blogs/{id}", handler(corOptions)).Methods("OPTIONS")
	router.Handle("/blogs", handler(corOptions)).Methods("OPTIONS")
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static", fileHandler))
	http.Handle("/", router)

	log.Printf("Running on port %d\n", *port)

	addr := fmt.Sprintf(":%d", *port)
	// this call blocks -- the progam runs here forever
	err := http.ListenAndServe(addr, nil)
	fmt.Println(err.Error())
}
