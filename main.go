package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang/glog"
)

const (
	Breakfast int = iota
	Lunch
	Dinner
)

type ClientMeal struct {
	Location    string    `json:"location"`
	Date        time.Time `json:"date"`
	Meal        int       `json:"meal"`
	Description string    `json:"description"`
}

type Meal struct {
	TimeStamp time.Time
	Location    string
	Date        time.Time
	Meal        int
	Description string
}

func meals(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Error(err)
		return
	}
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		glog.Error(err)
		return
	}
	fmt.Fprintf(w, "POST: %#v\n", m)
}

func badges(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Find badges for %s\n", r.URL.Path)
}

func main() {
	http.HandleFunc("/meals", meals)
	http.Handle("/badges/", http.StripPrefix("/badges/", http.HandlerFunc(badges)))
	glog.Fatal(http.ListenAndServe(":8080", nil))
}
