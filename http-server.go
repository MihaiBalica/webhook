package main

import (
	"database/sql"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

/*
TransactionID is used to extract the transaction id from JSON
ID contains that transaction id
*/
type TransactionID struct {
	ID string `json:"id"`
}

/*
Tag 		- used for manipulating info read from mysql db
ID			- contains the transaction ID, column in db
Datetime 	- when was the POST request received, column in db
Source		- this info, extracted from Header, saved in separate column in db
Header		- entire Header of the POST request, saved as Base64 encoded string in db
Body		- entire JSON from POST request, saved as Base64 encoded string in db
*/
type Tag struct {
	ID       string `json:"trId"`
	Datetime string `json:"Datetime"`
	Source   string `json:"Source"`
	Header   string `json:"Header"`
	Body     string `json:"body"`
}

type webhook struct {
	WebHook []RawContent `json:"webhooks"`
}

//RawContent struct stores the raw headers and bodyes stored in db, Base64 decoded.
type RawContent struct {
	Headers map[string]interface{} `json:"Headers"`
	Bodies  map[string]interface{} `json:"body"`
}

func isDigit(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

func getEnv(key string, defaultVal string) string {
    if value, exists := os.LookupEnv(key); exists {
	return value
    }

    return defaultVal
}

// MySQLUsername - username to connect to mysql db
var MySQLUsername = getEnv("MYSQL_USER","axiamed")
// MySQLPassword - le password
var MySQLPassword = getEnv("MYSQL_PASSWORD","axiamed")
// MySQLHost - le mysql host - see docker-compose.yml, overthere it is named 'db'
var MySQLHost = getEnv("MYSQL_HOSTNAME","db")
// MySQLPort - the mysql db port. 3306 is the default one but you never know.
var MySQLPort = getEnv("MYSQL_PORT","3306")
// Verbose - if "true" then return some info
var Verbose = getEnv("verbose","false")

func processRequest(w http.ResponseWriter, req *http.Request) {

	v := false

	if Verbose == "true" {v = true}
		
	method := strings.ToUpper(req.Method)
	fmt.Println ("Received method: ", method)
	switch method {
	case "POST":

		if v { fmt.Println(w, "POST request received!") }

		contentType := req.Header.Get("content-type")
		fmt.Println( "Content type: ",string(contentType))
		if strings.Contains(contentType, "application/json") {

			origin := req.Header.Get("x-axia-origin-system")
			if v { fmt.Println(w, "POST request origin : %v\n", string(origin)) }
			
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				panic(err)
			}
			if len(string(body)) < 10 {
				if v { fmt.Println("Empty body!") }
				return
			}
			log.Println(string(body))

			Headers := req.Header
			Header, err := json.Marshal(Headers)
			if err != nil {
				panic(err)
			}
			if v { fmt.Println("Header to JSON      : ", string(Header)) }

			// fmt.Fprintf("Header: %v\n",  Headers)
			for name, value := range req.Header {
				if v { fmt.Printf("%v: %v\n", name, value) }
			}


			var msg TransactionID
			err = json.Unmarshal(body, &msg)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			output, err := json.Marshal(msg.ID)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if v { fmt.Println( "Transaction ID is   : ", string(output)) }
			if v { fmt.Println( "POST request body is: ", string(body)) }

			//let's populate db
			db, err := sql.Open("mysql", MySQLUsername + ":" + MySQLPassword + "@tcp(" + MySQLHost + ":" + MySQLPort + ")/")
			if err != nil {
				fmt.Println(err.Error())
			} else {
				if v { fmt.Println("Connected to DB successfully") }
			}

			_, err = db.Exec("USE webhook")
			if err != nil {
				fmt.Println(err.Error())
			} else {
				if v { fmt.Println("DB selected successfully..") }
			}

			var insertQuery string = "INSERT INTO calls ( trId , Datetime,  Source, Header, body ) values ( " +
				string(output) + ", \"" +
				time.Now().Format("2006-01-02 15:04:05") + "\", \"" +
				origin + "\", \"" +
				string(b64.StdEncoding.EncodeToString([]byte(Header))) + "\", \"" +
				string(b64.StdEncoding.EncodeToString([]byte(body))) + "\"  );"

			if v { fmt.Println(insertQuery) }
			stmt, err := db.Prepare(insertQuery)
			if err != nil {
				fmt.Println(err.Error())
			}

			_, err = stmt.Exec()
			if err != nil {
				fmt.Println(err.Error())
			} else {
				if v { fmt.Println("Insert request successfully executed..") }
			}

			defer db.Close()
			defer req.Body.Close()
		}
	case "GET":
		log.Println("Received "+req.Method+" request from", req.RemoteAddr, req.RequestURI)
		log.Println("Detected this path: ", req.URL)

		validID := false
		trID := req.URL.String()

		if len(trID) <= 10 {

			if strings.Contains(trID[0:1], "/") {
				log.Println("First character in URL checked: '/'")

				if strings.Contains(trID[len(trID)-1:len(trID)], "?") {
					log.Println("Received a transaction id request")
					trID = strings.Trim(trID, "/")
					trID = strings.Trim(trID, "?")
					if isDigit(trID) {
						log.Println("Received a valid transaction Id: ", trID)
						validID = true
					}
				}
			}
		}

		//the transaction ID requested has valid format so proceeding to looking for it in DB
		if validID {
			//let's search db
			db, err := sql.Open("mysql", MySQLUsername + ":" + MySQLPassword + "@tcp(" + MySQLHost + ":" + MySQLPort + ")/")
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			fmt.Println("Connected to DB successfully")
			

			_, err = db.Exec("USE webhook")
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			fmt.Println("DB selected successfully..")
			

			fmt.Println("SELECT trId, Datetime, Source, Header, body from calls where trId = '" + trID + "';")

			result, err := db.Query("SELECT trId, Datetime, Source, Header, body from calls where trId = '" + trID + "';")
			if err != nil {
				fmt.Println(err.Error())
				return
			} 
			fmt.Println("Select request successfully executed..")
			

			mainJSON := webhook{}

			for result.Next() {
				var tag Tag

				err = result.Scan(&tag.ID, &tag.Datetime, &tag.Source, &tag.Header, &tag.Body)

				if err != nil {
					panic(err.Error()) // proper error handling instead of panic in your app
				}

				fmt.Println(tag.Source)
				sHeader, err1 := b64.StdEncoding.DecodeString(tag.Header)
				// fmt.Println(sHeader)
				if err != nil {
					fmt.Println(err1)
				}
				sBody, err1 := b64.StdEncoding.DecodeString(tag.Body)
				// fmt.Println(string(sBody))
				if err != nil {
					fmt.Println(err1)
				}
				var resultH map[string]interface{}
				json.Unmarshal([]byte(sHeader), &resultH)
				// fmt.Println(resultH)

				var resultB map[string]interface{}
				json.Unmarshal([]byte(sBody), &resultB)
				// fmt.Println(resultB)

				raw := RawContent{resultH, resultB}
				mainJSON.WebHook = append(mainJSON.WebHook, raw)
			}

			js, _ := json.MarshalIndent(mainJSON, "", "    ")
			// fmt.Printf("%s\n", js)
			fmt.Fprintf(w, "%s\n", js)
			defer db.Close()
		}
	default:
		fmt.Fprintf(w, "Sorry, only POST and GET methods are supported.")
	}

}

func initialize(w http.ResponseWriter, req *http.Request) {
	
	for name, Headers := range req.Header {
		for _, h := range Headers {
			fmt.Fprintf(w, "%v: %v\n", name, h)
		}
	}

	db, err := sql.Open("mysql", MySQLUsername + ":" + MySQLPassword + "@tcp(" + MySQLHost + ":" + MySQLPort + ")/")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Connected to DB successfully")
	}

	_, err = db.Exec("USE webhook")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("DB selected successfully..")
	}

	stmt, err := db.Prepare("create table IF NOT EXISTS calls( id int not null auto_increment, trId varchar(10) not null, Datetime TIMESTAMP not null, Source varchar(32) not null, Header varchar(15000) not null, body varchar(21844) not null, primary key ( id ));")
	if err != nil {
		fmt.Println(err.Error())
	}

	_, err = stmt.Exec()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Table created successfully..")
	}

	defer db.Close()
}

func main() {
	http.HandleFunc("/", processRequest)
	http.HandleFunc("/initializeAxia", initialize)

	fmt.Printf("Starting server for testing HTTP POST...\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
