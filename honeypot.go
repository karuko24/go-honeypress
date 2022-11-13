package main

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var mongoCollection *mongo.Collection

type RequestData struct {
	Ip           string
	UserAgent    string
	TriggeredUrl string
	Time         string
	Data         string
	Country      string
}

func logPOST(mongoCollection *mongo.Collection, r *http.Request) {
	if r.Method == "POST" {
		var ip string = strings.Split(r.RemoteAddr, ":")[0]
		resp, err := http.Get(fmt.Sprintf("http://www.geoplugin.net/json.gp?ip=%s", ip))
		if err != nil {
			log.Print("Error while getting geolocation data")
		}

		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Print(err)
		}

		var country string = string(bytes)
		var useragent string = r.UserAgent()
		var triggeredUrl string = r.RequestURI

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
		}
		var payload string = string(body)

		data := RequestData{ip, useragent, triggeredUrl, time.Now().String(), payload, country}
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		_, err = mongoCollection.InsertOne(ctx, data)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	applyHeaders(w)
	logPOST(mongoCollection, r)
	if r.URL.Path != "/" {
		logPOST(mongoCollection, r)
		fmt.Fprintf(w, "")
		return
	}
	http.ServeFile(w, r, "templates/index.php")
}

func srdbHandler(w http.ResponseWriter, r *http.Request) {
	applyHeaders(w)
	logPOST(mongoCollection, r)
	http.ServeFile(w, r, "templates/searchreplacedb2.php")
}

func debugLogHandler(w http.ResponseWriter, r *http.Request) {
	applyHeaders(w)
	fmt.Fprintf(w, "aaa")
}

func adminAjaxHandler(w http.ResponseWriter, r *http.Request) {
	applyHeaders(w)
	logPOST(mongoCollection, r)
	fmt.Fprintf(w, "0")
}

func xmlrpcHandler(w http.ResponseWriter, r *http.Request) {
	applyHeaders(w)
	logPOST(mongoCollection, r)
	if r.Method == "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("XML-RPC server accepts POST requests only."))
	}
	fmt.Fprintf(w, "")
}

func readmeHandler(w http.ResponseWriter, r *http.Request) {
	applyHeaders(w)
	http.ServeFile(w, r, "templates/readme.html")
}

func wpconfigHandler(w http.ResponseWriter, r *http.Request) {
	applyHeaders(w)
	logPOST(mongoCollection, r)
	fmt.Fprintf(w, "")
}

func wpadminHandler(w http.ResponseWriter, r *http.Request) {
	applyHeaders(w)
	http.Redirect(w, r, "/wp-login.php", http.StatusFound)
}

func wploginHandler(w http.ResponseWriter, r *http.Request) {
	applyHeaders(w)
	logPOST(mongoCollection, r)
	http.ServeFile(w, r, "templates/wp-login.php")
}

func applyHeaders(w http.ResponseWriter) {
	w.Header().Set("Server", "nginx")
	w.Header().Set("Content", "text/html; charset=UTF-8")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Keep-Alive", "timeout=20")
	w.Header().Set("Link", "<http://wordpress.com/wp-json/>; rel=\"https://api.w.org/\"")
	w.Header().Set("Set-Cookie", "wordpress_test_cookie=WP+Cookie+check; path=/")
}

func connectMongo() (*mongo.Collection, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	var client *mongo.Client
	var err error

	if os.Getenv("MONGO_URL") == "" {
		client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	} else {
		client, err = mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URL")))
	}

	ctx, _ = context.WithTimeout(context.Background(), 2*time.Second)
	err = client.Ping(ctx, readpref.Primary())
	if err == nil {
		log.Print("Connection established.")
		collection := client.Database("honeypot").Collection("honeypot")
		return collection, nil
	}
	fmt.Println(err)
	return nil, errors.New("Couldn't connect to MongoDB-Server.")
}

func main() {
	log.Print("Starting Wordpress-Honeypot...")
	log.Print("Trying to connect to MongoDB-Server...")
	var err error
	mongoCollection, err = connectMongo()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/searchreplacedb2.php", srdbHandler)
	http.HandleFunc("/wp-content/debug.log", debugLogHandler)
	http.HandleFunc("/wp-admin/admin-ajax.php", adminAjaxHandler)
	http.HandleFunc("/xmlrpc.php", xmlrpcHandler)
	http.HandleFunc("/readme.html", readmeHandler)
	http.HandleFunc("/wp-config.php", wpconfigHandler)
	http.HandleFunc("/wp-admin", wpadminHandler)
	http.HandleFunc("/wp-admin/", wpadminHandler)
	http.HandleFunc("/wp-login.php", wploginHandler)
	if os.Getenv("HONEYPOT_PORT") == "" {
		fmt.Println(http.ListenAndServe(":3000", nil))
	} else {
		fmt.Println(http.ListenAndServe(os.Getenv("HONEYPOT_PORT"), nil))
	}
}
