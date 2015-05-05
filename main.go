package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/golang/glog"
)

const (
	Breakfast int = iota
	Lunch
	Dinner
)

type ClientEvent interface {
	Valid() error
}

type Event interface {
	Handle()
}

type MSI map[string]interface{}
type ClientMeal struct {
	Client      uuid.UUID `json:"id""`
	Location    string    `json:"location"`
	Date        time.Time `json:"date"`
	Meal        int       `json:"meal"`
	Description string    `json:"description"`
}

func ParseMSI(b []byte) (cm ClientMeal, err error) {
	var msi MSI
	if err := json.Unmarshal(b, &msi); err != nil {
		return cm, err
	}
	cm.Client = uuid.Parse(msi["id"].(string))
	cm.Location = msi["location"].(string)
	if cm.Date, err = time.Parse(time.RFC3339Nano, msi["date"].(string)); err != nil {
		return cm, err
	}
	switch msi["meal"].(string) {
	case "Breakfast":
		cm.Meal = Breakfast
	case "Lunch":
		cm.Meal = Lunch
	case "Dinner":
		cm.Meal = Dinner
	default:
		return cm, errors.New("invalid meal")
	}
	cm.Description = msi["description"].(string)
	return cm, nil
}

func ParseClientMeal(b []byte) (cm ClientMeal, err error) {
	err = json.Unmarshal(b, &cm)
	return cm, err
}

func (cm ClientMeal) Valid() (err error) {
	if cm.Client != nil && len(cm.Client) == 16 &&
		cm.Location != "" &&
		cm.Meal >= Breakfast && cm.Meal <= Dinner &&
		cm.Description != "" {
		return nil
	}
	return errors.New("Client Meal Validation Error")
}

func NewClientID(b []byte) (c ClientID) {
	if len(b) != 16 {
		return c
	}
	for i := range b {
		c[i] = b[i]
	}
	return c
}

type Meal struct {
	TimeStamp   time.Time
	Client      ClientID
	Location    string
	Date        time.Time
	Meal        int
	Description string
}

func NewMeal(cm ClientMeal) Meal {
	return Meal{
		TimeStamp:   time.Now(),
		Client:      NewClientID(cm.Client),
		Location:    cm.Location,
		Date:        cm.Date,
		Meal:        cm.Meal,
		Description: cm.Description,
	}
}

type Badge struct {
	Name string `json:"name"`
}

func RecomputeBadges(c *Client) {
}

func (m Meal) Handle() {
	client, ok := clients[m.Client]
	if !ok {
		return
	}
	client.meals = append(client.meals, m)
	RecomputeBadges(&client)
}

var events = make(chan Event)

func EventHandler() {
	for {
		e := <-events
		e.Handle()
	}
}

func WriteEvent(e Event) error {
	f, err := os.OpenFile("event.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	return enc.Encode(e)
}

type Client struct {
	ID     ClientID
	meals  []Meal
	badges []Badge
}

type ClientID [16]byte

var clients map[ClientID]Client

func meals(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Error(err)
		return
	}
	var cm ClientMeal
	if cm, err = ParseClientMeal(body); err != nil {
		glog.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err = cm.Valid(); err != nil {
		glog.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	m := NewMeal(cm)
	WriteEvent(m)
	events <- m
}

type GetBadges struct {
	ID    ClientID
	reply chan []Badge
}

func (g GetBadges) Handle() {
	var b []Badge
	if c, ok := clients[g.ID]; ok {
		b = c.badges
	}
	g.reply <- b
}

func badges(w http.ResponseWriter, r *http.Request) {
	uuid := uuid.Parse(r.URL.Path)
	reply := make(chan []Badge)
	events <- GetBadges{
		ID:    NewClientID(uuid),
		reply: reply,
	}
	badges := <-reply
	b, err := json.Marshal(badges)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(b); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func main() {
	go EventHandler()
	// read and replay  events
	http.HandleFunc("/meals", meals)
	http.Handle("/badges/", http.StripPrefix("/badges/", http.HandlerFunc(badges)))
	glog.Fatal(http.ListenAndServe(":8080", nil))
}
