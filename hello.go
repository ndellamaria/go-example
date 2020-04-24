package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

var tpl = template.Must(template.ParseFiles("index.html"))

type Search struct {
	SearchKey	string
	NextPage	int
	TotalPages	int
	Results 	Results
}

type Results struct {
	FullStudiesResponse FullStudiesResponse `json:"FullStudiesResponse"`
}

type FullStudiesResponse struct {
	NStudiesReturned int `json:"NStudiesReturned"`
	FullStudies []SingleStudy `json:"FullStudies"`
}


type SingleStudy struct {
	Study struct {
			ProtocolSection struct {
				IdentificationModule struct {
					Organization struct {
						OrgFullName string `json:"OrgFullName"`
					} `json:"Organization"`
					BriefTitle string `json:"BriefTitle"`
				} `json:"IdentificationModule"`
				StatusModule struct {
					OverallStatus string `json:"OverallStatus"`
					StartDateStruct struct {
						StartDate string `json:"StartDate"`
					} `json:"StartDateStruct"`
				} `json:"StatusModule"`
			} `json:"ProtocolSection"`
	} `json:"Study"`
}

func (s *Search) IsLastPage() bool {
	return s.NextPage >= s.TotalPages
}

func (s *Search) PreviousPage() int {
	return s.CurrentPage() - 1
}

func (s *Search) CurrentPage() int {
	if s.NextPage == 1 {
		return s.NextPage
	}

	return s.NextPage - 1
}

type TrialsAPIError struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tpl.Execute(w,nil)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(r.URL.String())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
		return
	}

	params := u.Query()
	searchKey := params.Get("expr")
	page := params.Get("page")
	if page == "" {
		page="1"
	}

	fmt.Println("Search Query is: ", searchKey)
	fmt.Println("Results page is: ", page)

	search := &Search{}
	search.SearchKey = searchKey

	next, err := strconv.Atoi(page)
	if err != nil {
		http.Error(w, "Unexpected server error", http.StatusInternalServerError)
		return
	}

	search.NextPage = next
	pageSize := 20 

	endpoint := fmt.Sprintf("https://clinicaltrials.gov/api/query/full_studies?expr=%s&max_rnk=30&fmt=JSON&pageSize=%d&page=%d&language=en", url.QueryEscape(search.SearchKey), pageSize, search.NextPage)
	resp, err := http.Get(endpoint)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		newError := &TrialsAPIError{}
		err := json.NewDecoder(resp.Body).Decode(newError)

		if err != nil {
			http.Error(w, "Unexpected server error", http.StatusInternalServerError)
			return
		}

		http.Error(w, newError.Message, http.StatusInternalServerError)
		return
	}



	err = json.NewDecoder(resp.Body).Decode(&search.Results)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	search.TotalPages = int(math.Ceil(float64(search.Results.FullStudiesResponse.NStudiesReturned / pageSize)))
	if ok := !search.IsLastPage(); ok {
		search.NextPage++
	}

	err = tpl.Execute(w, search)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	mux.HandleFunc("/search", searchHandler)
	mux.HandleFunc("/", indexHandler)
	http.ListenAndServe(":"+port,mux)
}