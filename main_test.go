package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/alertmanager/template"
)

var (
	alertmanagerMessage = `	{
  "version": "4",
  "groupKey": "",    
  "status": "firing",
  "receiver": "megafon-sms",
	"groupLabels": {"alertname":"DenyOfService", "env":"prod"},
  "commonLabels": {"alertname":"DenyOfService", "env":"prod"},
  "commonAnnotations": {},
	"externalURL": "http://localhost:9093",
  "alerts": [
    {
      "status": "firing",
      "labels": {"alertname":"DenyOfService", "env":"prod"},
      "annotations": {},
			"startsAt": "2019-01-04T11:08:54.016165421+03:00",
			"endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": ""
    }
  ]
}
`
)

func TestMakeMessage(t *testing.T) {
	var data template.Data
	if err := json.NewDecoder(bytes.NewBufferString(alertmanagerMessage)).Decode(&data); err != nil {
		t.Fatalf("%v", err)
	}
	expected := "firing. alertname:DenyOfService"
	msg := makeMessage(data)
	if strings.Compare(msg, expected) != 0 {
		t.Fatalf("Expected: \"%v\", got: \"%v\"", expected, msg)
	}
}

func TestSendSms(t *testing.T) {
	megafonServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			t.Error(err)
		}
		megafonReply := []byte(`{"result": {"status": {"code": 0, "description": "ok"},"msg_id": "124343"}}`)
		w.Write(megafonReply)
	}))
	defer megafonServer.Close()
	gwUrl = megafonServer.URL
	ch := make(chan int)
	go sendSms(79261238212, "Это тест", ch)
	if ok := <-ch; ok != 0 {
		t.Fail()
	}
}

func TestWebhookHandler(t *testing.T) {
	megafonService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			t.Error(err)
		}
		megafonReply := []byte(`{"result": {"status": {"code": 0, "description": "ok"},"msg_id": "124343"}}`)
		w.Write(megafonReply)
	}))
	defer megafonService.Close()
	gwUrl = megafonService.URL
	smsTo = "79261238212"

	r := httptest.NewRequest(http.MethodPost, "/sms", bytes.NewBufferString(alertmanagerMessage))
	w := httptest.NewRecorder()
	webhookHandler(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fail()
	}

}
