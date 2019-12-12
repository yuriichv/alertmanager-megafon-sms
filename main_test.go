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

	megafonGoodReply      = `{"result": {"status": {"code": 0, "description": "ok"},"msg_id": "124343"}}`
	megafonBadCodeReply   = `{"result": {"status": {"code": 2, "description": "code 2 test reply"},"msg_id": "124344"}}`
	megafonBadFormatReply = `{"result": {"status": {"cod": 1, "description": "corrupted code"},"msge_id": "124345"}}`
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

func DoTestSendSms() {

}

func TestSendSms(t *testing.T) {
	var tests = []struct {
		args string
		want int
	}{
		{args: "megafonGoodReply", want: 0},
		{args: "megafonBadFormatReply", want: 1},
		{args: "megafonBadCodeReply", want: 1},
	}
	megafonServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		megafonReplies := make(map[string]string)
		megafonReplies["megafonGoodReply"] = megafonGoodReply
		megafonReplies["megafonBadCodeReply"] = megafonBadCodeReply
		megafonReplies["megafonBadFormatReply"] = megafonBadFormatReply

		var req Request
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			t.Error(err)
		}
		w.Write([]byte(megafonReplies[req.Message]))
	}))
	defer megafonServer.Close()
	gwUrl = megafonServer.URL
	ch := make(chan int)

	for _, v := range tests {
		go sendSms(79261238212, v.args, ch)
		if ok := <-ch; ok != v.want {
			t.Errorf("sendSms(%s) returned %v, expected %v", v.args, ok, v.want)
			//t.Fail()
		}
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
