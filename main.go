package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"

	//	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/alertmanager/template"
	log "github.com/sirupsen/logrus"
)

var (
	port        = getEnv("SMS_PORT", "9097")                     //listern port
	gwUrl       = getEnv("SMS_GW_URL", "https://localhost:7443") //sms gateway url
	smsFrom     = getEnv("SMS_FROM", "VGR ID")                   //FROM
	smsTo       = getEnv("SMS_TO", "00000000000,11111111111")    //phone numbers, split by ","
	insecure    = getEnv("SMS_INSECURE", "false")                //disable tls cert check
	smsUser     = getEnv("SMS_USER", "")                         //username for basic auth
	smsPassword = getEnv("SMS_PASSWORD", "")                     //password for basic auth
	logLevel    = getEnv("SMS_LOG_LEVEL", "info")
)

func getEnv(key, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

type Request struct {
	From         string `json:"from"`
	To           int    `json:"to"`
	Message      string `json:"message"`
	Callback_url string `json:"callback_url,omitempty"`
}

type Payload struct {
	Description string `json:description`
	Code        string `json:description`
}
type Status struct {
	Payload     []Payload `json:payload`
	Code        int       `json:"code"`
	Description string    `json:"description"`
}

type Result struct {
	Status Status `json:"status"`
	Msg_id string `json:"msg_id"`
}

type Reply struct {
	Result Result `json:"result"`
}

func makeMessage(data template.Data) string {
	//TODO: можно добавить все CommonLables в формате key:value
	return "Alert: " + data.GroupLabels["alertname"]
}

func sendSms(smsTo int, smsMessage string, statusChan chan int) {
	request := Request{smsFrom, smsTo, smsMessage, ""}
	b, err := json.Marshal(request)
	req, err := http.NewRequest(http.MethodPost, gwUrl, bytes.NewBuffer(b))
	if err != nil {
		log.Errorf("%v", err)
		statusChan <- 1
		return
	}
	req.SetBasicAuth(smsUser, smsPassword)
	req.Header.Set("Content-type", "application/json")
	client := http.Client{}

	log.Debugf("Sending request to sms gateway: %v", req)
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("%v", err)
		statusChan <- 1
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Errorf("Server reply: %+v", resp)
		statusChan <- 1
		return
	}

	var reply Reply
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		log.Errorf("Error %v parsing reply %+v", err, resp)
		statusChan <- 1
		return
	}

	if reply.Result.Status.Code != 0 {
		r, _ := json.Marshal(reply)
		log.Errorf("Call to gateway fault with code: %v, reply: %+v", resp.StatusCode, string(r))
		statusChan <- 1
		return
	}
	log.Info("sms to %v sent. id %v", smsTo, reply.Result.Msg_id)
	statusChan <- 0
	return
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	var data template.Data
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		log.Error("%v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Infof("alert recieved: %v", data.GroupLabels["alertname"])
	if log.GetLevel() == log.DebugLevel {
		d, _ := json.Marshal(data)
		log.Debugf("full request: %v", string(d))
	}

	smsMessage := makeMessage(data)
	statusChan := make(chan int)
	returnStatus := http.StatusOK

	smsToSlice := strings.Split(smsTo, ",")
	for _, phone := range smsToSlice {
		if n, err := strconv.Atoi(phone); err != nil {
			log.Error("%v", err)
			returnStatus = http.StatusInternalServerError
		} else {
			go sendSms(n, smsMessage, statusChan)
		}
	}
	//wait for answers or timeout
	for i := 0; i < len(smsToSlice); i++ {
		select {
		case state := <-statusChan:
			if state != 0 {
				returnStatus = http.StatusInternalServerError
			}
		case <-time.After(time.Second * 60):
			log.Error("tiemout sending gateway requests")
			http.Error(w, "Timeout sending sms", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(returnStatus)

}

func main() {
	switch strings.ToLower(logLevel) {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)

	log.Infof("Init parameters: SMS_GW_URL=%v, SMS_FROM=%v, SMS_TO=%v, SMS_INSECURE=%v", gwUrl, smsFrom, smsTo, insecure)
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure == "true"}
	http.HandleFunc("/sms", webhookHandler)
	log.Printf("Listetning on port %v", port)
	log.Fatalln(http.ListenAndServe(":"+port, nil))
}
