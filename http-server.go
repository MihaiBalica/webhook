package main

import (
	"fmt"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	b64 "encoding/base64"
	"time"
	"strings"
	"unicode"
)
type TransactionID struct {
    ID string `json:"id"`
}

type Tag struct {
	ID   		string `json:"trId"`
	datetime 	string `json:"datetime"`
	source		string `json:"source"`
	header		string `json:"header"`
	Body		string `json:"body"`
}

type webhook struct {
	WebHook	[]RawContent `json:"webhooks"`
}

type RawContent struct {
	Headers map[string]interface{} `json:"headers"`
	Bodies	map[string]interface{} `json:"body"`
}

func isDigit(s string) bool {
    for _, c := range s {
        if !unicode.IsDigit(c) {
            return false
        }
    }
    return true
}

func hello (w http.ResponseWriter, req *http.Request){
	

	switch req.Method {
	case "POST":

		origin := req.Header.Get("x-axia-origin-system")
		fmt.Fprintf(w,"The origin of POST request is: %v\n", string(origin))

		headers := req.Header
		header, err := json.Marshal(headers)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(w,"The headers of POST request into JSON: %v\n", string(header))

		// fmt.Fprintf("Header: %v\n",  headers)
		for name, value := range req.Header {
			fmt.Printf("%v: %v\n", name, value)
		}

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			panic(err)
		}
		log.Println(string(body))

		var msg TransactionID
		err = json.Unmarshal(body,&msg)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		output, err := json.Marshal(msg.ID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		fmt.Fprintf(w, "Transaction ID is %v\n", string(output))
		fmt.Fprintf(w, "Post from website! r.PostFrom = %v\n", string(body))

		//let's populate db
		db, err := sql.Open("mysql", "axiamed:axiamed@tcp(db:3306)/")
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println("Connected to DB successfully")
		}

		_,err = db.Exec("USE webhook")
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println("DB selected successfully..")
		}

		var insertQuery string = "INSERT INTO calls ( trId , datetime,  source, header, body ) values ( " + 
				string(output) + ", \"" + 
				time.Now().Format("2006-01-02 15:04:05") + "\", \"" +
				origin + "\", \"" + 
				string(b64.StdEncoding.EncodeToString([]byte(header))) + "\", \"" + 
				string(b64.StdEncoding.EncodeToString([]byte(body))) + "\"  );"

		fmt.Println(insertQuery)
		stmt, err := db.Prepare(insertQuery)
		if err != nil {
			fmt.Println(err.Error())
		}
		
		_, err = stmt.Exec()
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println("Insert request successfully executed..")
		}

		defer db.Close()
		defer req.Body.Close()
	case "GET":
		log.Println("Received " + req.Method + " request from",req.RemoteAddr, req.RequestURI)
		log.Println("Detected this path: ",req.URL)

		trID := req.URL.String()

		if strings.Contains(trID[0:1],"/") {
			log.Println("First character in URL checked: '/'")
			if strings.Contains(trID[len(trID)-1:len(trID)],"?") {
				log.Println("Received a transaction id request")
				trID = strings.Trim(trID,"/")
				trID = strings.Trim(trID,"?")
				if isDigit(trID) {
					log.Println ("Received a valid transaction Id: " , trID)
				}
			}
		}

		//let's populate db
		db, err := sql.Open("mysql", "axiamed:axiamed@tcp(db:3306)/")
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println("Connected to DB successfully")
		}

		_,err = db.Exec("USE webhook")
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println("DB selected successfully..")
		}

		fmt.Println("SELECT trId, datetime, source, header, body from calls where trId like '" + trID + "';") 
			
		result, err := db.Query("SELECT trId, datetime, source, header, body from calls where trId like '" + trID + "';")
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println("Select request successfully executed..")
		}
		
		mainJson := webhook{}

		for result.Next() {
			var tag Tag

			err = result.Scan(&tag.ID, &tag.datetime, &tag.source, &tag.header, &tag.Body)

			if err != nil {
				panic(err.Error()) // proper error handling instead of panic in your app
			}
			
			fmt.Println(tag.source)
			sHeader, err1 := b64.StdEncoding.DecodeString(tag.header)
    		// fmt.Println(sHeader)
    		fmt.Println(err1)
			sBody, err1 := b64.StdEncoding.DecodeString(tag.Body)
    		// fmt.Println(string(sBody))
    		fmt.Println(err1)


			var resultH map[string]interface{}
			json.Unmarshal([]byte(sHeader), &resultH)
			fmt.Println(resultH)

			var resultB map[string]interface{}
			json.Unmarshal([]byte(sBody), &resultB)
			fmt.Println(resultB)

			raw := RawContent{resultH, resultB}
			mainJson.WebHook = append(mainJson.WebHook,raw)
		}

		js, _ := json.MarshalIndent(mainJson,"", "    ")
		fmt.Printf("%s\n",js)
		fmt.Fprintf(w, "%s\n", js)

	default:
		fmt.Fprintf(w, "Sorry, only POST and GET methods are supported.")
	}

}

func initialize (w http.ResponseWriter, req *http.Request){
	for name, headers := range req.Header {
		for _, h := range headers {
			fmt.Fprintf(w, "%v: %v\n", name, h)
		}
	}

	db, err := sql.Open("mysql", "axiamed:axiamed@tcp(db:3306)/")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Connected to DB successfully")
	}

	_,err = db.Exec("USE webhook")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("DB selected successfully..")
	}

	stmt, err := db.Prepare("create table IF NOT EXISTS calls( id int not null auto_increment, trId varchar(10) not null, datetime TIMESTAMP not null, source varchar(32) not null, header varchar(10000) not null, body varchar(15000) not null, primary key ( id ));")
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
	http.HandleFunc("/", hello)
	http.HandleFunc("/initializeAxia", initialize)

	fmt.Printf("Starting server for testing HTTP POST...\n")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatal(err)
    }
}