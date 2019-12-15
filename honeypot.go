package main

import (
  "fmt"
  "log"
  "context"
  "time"
  "net/http"
  "errors"
  "os"
  "io/ioutil"
  "regexp"
  "go.mongodb.org/mongo-driver/mongo"
  "go.mongodb.org/mongo-driver/mongo/options"
  "go.mongodb.org/mongo-driver/mongo/readpref"
)

var mongoCollection *mongo.Collection

type RequestData struct {
  Ip string
  IsTor bool
  UserAgent string
  TriggeredUrl string
  Time string
  Data string
}

func checkTor(ip string) (bool) {
  client := &http.Client{}

  req, err := http.NewRequest("GET", "https://check.torproject.org/exit-addresses", nil)
  if err != nil {
    log.Fatal(err)
    return false
  }
  req.Header.Add("user-agent", `go-honeypress/(https://github.com/karuko24/go-honeypress)`)
  resp, err := client.Do(req)
  if err != nil {
    log.Fatal(err)
    return false
  }
  if resp.StatusCode == http.StatusOK {
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
      log.Fatal(err)
      return false
    }
    bodyString := string(bodyBytes)

    match, err := regexp.MatchString(ip, bodyString)
    if err != nil {
      log.Fatal(err)
      return false
    }

    if match {
      return true
    }
  }
  return false
}

func logPOST(mongoCollection *mongo.Collection, ip string, useragent string, triggeredUrl string, payload string) {
  isTor := checkTor(ip)
  data := RequestData{ip, isTor, useragent, triggeredUrl, time.Now().String(), payload}
  ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
  _, err := mongoCollection.InsertOne(ctx, data)
  if err != nil {
    log.Fatal(err)
  }
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
  applyHeaders(w)
  if r.Method == "POST" {
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
      log.Fatal(err)
    }
    logPOST(mongoCollection, r.RemoteAddr, r.UserAgent(), r.RequestURI, string(body))
  }
  if r.URL.Path != "/" {
    if r.Method == "POST" {
      body, err := ioutil.ReadAll(r.Body)
      if err != nil {
        log.Fatal(err)
      }
      logPOST(mongoCollection, r.RemoteAddr, r.UserAgent(), r.RequestURI, string(body))
    }
    fmt.Fprintf(w, "")
    return
  }
  http.ServeFile(w, r, "templates/index.php")
}

func srdbHandler(w http.ResponseWriter, r *http.Request) {
  applyHeaders(w)
  if r.Method == "POST" {
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
      log.Fatal(err)
    }
    logPOST(mongoCollection, r.RemoteAddr, r.UserAgent(), r.RequestURI, string(body))
  }
  http.ServeFile(w, r, "templates/searchreplacedb2.php")
}

func debugLogHandler(w http.ResponseWriter, r *http.Request) {
  applyHeaders(w)
  fmt.Fprintf(w, "aaa")
}

func adminAjaxHandler(w http.ResponseWriter, r *http.Request) {
  applyHeaders(w)
  if r.Method == "POST" {
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
      log.Fatal(err)
    }
    logPOST(mongoCollection, r.RemoteAddr, r.UserAgent(), r.RequestURI, string(body))
  }
  fmt.Fprintf(w, "0")
}

func xmlrpcHandler(w http.ResponseWriter, r *http.Request) {
  applyHeaders(w)
  if r.Method == "POST" {
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
      log.Fatal(err)
    }
    logPOST(mongoCollection, r.RemoteAddr, r.UserAgent(), r.RequestURI, string(body))
  }
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
  if r.Method == "POST" {
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
      log.Fatal(err)
    }
    logPOST(mongoCollection, r.RemoteAddr, r.UserAgent(), r.RequestURI, string(body))
  }
  fmt.Fprintf(w, "")
}

func wpadminHandler(w http.ResponseWriter, r *http.Request) {
  applyHeaders(w)
  http.Redirect(w, r, "/wp-login.php", http.StatusFound)
}

func wploginHandler(w http.ResponseWriter, r *http.Request) {
  applyHeaders(w)
  if r.Method == "POST" {
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
      log.Fatal(err)
    }
    logPOST(mongoCollection, r.RemoteAddr, r.UserAgent(), r.RequestURI, string(body))
  }
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

func connectMongo() (*mongo.Collection, error){
  ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
  client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
  ctx, _ = context.WithTimeout(context.Background(), 2*time.Second)
  err = client.Ping(ctx, readpref.Primary())
  if err == nil {
    log.Print("Connection established.")
    collection := client.Database("honeypot").Collection("honeypot")
    return collection, nil
  }
  log.Fatal(err)
  return nil, errors.New("Couldn't connect to MongoDB-Server.")
}

func main() {
  log.Print("Starting Wordpress-Honeypot...")
  log.Print("Trying to connect to MongoDB-Server...")
  var err error
  mongoCollection, err = connectMongo()
  if err != nil {
    log.Fatal(err)
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
  log.Fatal(http.ListenAndServe(":3000", nil))
}
