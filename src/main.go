package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var gbif = "http://api.gbif.org/v1/"
var restCountry = "https://restcountries.eu/rest/v2/"
var tmp = "https://restcountries.eu/rest/v2/alpha/col"
var Version = "v1"
var startTime time.Time

type diagStruct struct {
	Gbif string				`json:"gbif"`
	Restcountries string 	`json:"restcountries"`
	Version string 			`json:"version"`
	Uptime int 				`json:"uptime"`
}

type countryInfo struct {
	Code string				`json:"countryCode"`
	CountryName string		`json:"countryName"`
	CountryFlag string		`json:"countryFlag"`
	Species string			`json:"species"`
	SpeciesKey int			`json:"speciesKey"`
}

type speciesResponse struct {
	Offset int		`json:"offset"`
	Limit int		`json:"limit"`
	EndOfRecords bool	`json:"endOfRecords"`
	Count int		`json:"count"`
	Results []countryInfo	`json:"results"`
	Facets []string		`json:"facets"`
}

type speciesSpecific struct {
	Key	int 				`json:"speciesKey"`
	Kingdom string 			`json:"kingdom"`
	Phylum string 			`json:"phylum"`
	Order string 			`json:"order"`
	Family string 			`json:"family"`
	Genus string 			`json:"genus"`
	ScientificName string	`json:"scientificName"`
	CanonicalName string	`json:"canonicalName"`
	Year string				`json:"year"`
}

type speciesYear struct {
	Bracketyear string `json:"bracketyear"`
	Year string 		`json:"year"`
}

type tmpCountry struct {
	Flag string			`json:"flag"`
	Name string			`json:"name"`
	alpha2Code string 	`json:"alpha2Code"`
}

func speciesLink(w http.ResponseWriter, r *http.Request){
	vars := mux.Vars(r)
	key := vars["speciesKey"]
	response, err := http.Get(gbif + "species/" + key)
	if err != nil{
		fmt.Print(err)
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		var species speciesSpecific
		err := json.Unmarshal(data, &species)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			yearResponse, err := http.Get(gbif + "species/" + key + "/name")
			var yearStruct speciesYear
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println(err)
			} else {
				yearData, _ := ioutil.ReadAll(yearResponse.Body)
				err = json.Unmarshal(yearData, &yearStruct)
				if err != nil {
					fmt.Print(err)
				}
				if yearStruct.Bracketyear != "" {
					species.Year = yearStruct.Bracketyear
				} else if yearStruct.Year != "" {
					species.Year = yearStruct.Year
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(species)
		}
	}
}

func countryLink(w http.ResponseWriter, r *http.Request){
	vars := mux.Vars(r)
	country := vars["countryIdentifier"]
	var limit string
	limitParam := strings.Split(r.URL.RequestURI(), "?")
	if len(limitParam) > 1 {
		limit = strings.Split(limitParam[1], "=")[1]
	}
	species, err := http.Get(gbif + "occurrence/search?country=" + country + "&limit=" + limit)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else {
		data, _ := ioutil.ReadAll(species.Body)
		var spec speciesResponse
		error2 := json.Unmarshal(data, &spec)
		if error2 != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			for i := 0; i < len(spec.Results); i++ {
				var tmp countryInfo
				tmp = spec.Results[i]
				flagInfo, error3 := http.Get(restCountry + "alpha/" + tmp.Code + "?fields=flag;alpha2Code;name")
				if error3 != nil{
					w.WriteHeader(http.StatusBadRequest)
				} else {
					var tmp tmpCountry
					data2, _ := ioutil.ReadAll(flagInfo.Body)
					error4 := json.Unmarshal(data2, &tmp)
					if error4 != nil{
						w.WriteHeader(http.StatusInternalServerError)
					} else {
						spec.Results[i].CountryFlag = tmp.Flag
						spec.Results[i].CountryName = tmp.Name
					}
				}
			}

		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(spec.Results)
	}
}

func diagLink(w http.ResponseWriter, r *http.Request){
	var status diagStruct
	gbiff, err := http.Get(gbif)
	if err != nil {
		fmt.Print(err)
		//w.WriteHeader(http.StatusBadRequest)
	}
	status.Gbif = gbiff.Status
	restcountries, err2 := http.Get(restCountry)
	if err2 != nil {
		fmt.Print(err2)
	}
	status.Restcountries = restcountries.Status
	status.Version = Version
	elapsed := time.Since(startTime)
	status.Uptime = int(elapsed.Seconds())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

func init() {
	startTime = time.Now()
}

func main(){
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("No Port is set")
	}
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/conservation/v1/country/{countryIdentifier}", countryLink).Methods("GET")
	router.HandleFunc("/conservation/v1/species/{speciesKey}", speciesLink).Methods("GET")
	router.HandleFunc("/conservation/v1/diag/", diagLink).Methods("GET")
	http.ListenAndServe(":" + port, router)
}
