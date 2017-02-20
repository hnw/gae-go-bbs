// How to deploy:
//   $ appcfg.py update . -A [application_id]

// +build appengine

package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"

	"google.golang.org/appengine"
	aelog "google.golang.org/appengine/log"

	"github.com/gorilla/mux"
	"github.com/mjibson/goon"
)

func init() {
	timezone := os.Getenv("TIMEZONE")
	if timezone != "" {
		loc, err := time.LoadLocation(timezone)
		if err == nil {
			timezoneOffset, err := strconv.ParseInt(os.Getenv("TIMEZONE_OFFSET"), 10, 0)

			if err == nil {
				loc = time.FixedZone(timezone, int(timezoneOffset))
			}

			timezone = ""
		}
		time.Local = loc
	}

	r := mux.NewRouter()
	r.HandleFunc("/bbs", newBbsHandler)
	r.HandleFunc("/bbs/{bbs_id:[0-9]+}/posts", listPostsHandler).Methods("GET")
	r.HandleFunc("/bbs/{bbs_id:[0-9]+}/posts", newPostHandler).Methods("POST")
	http.Handle("/", r)
}

func newBbsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	g := goon.NewGoon(r)

	if r.Method == "POST" {
		b := new(bbs)
		if err := b.fromRequest(r); err != nil {
			aelog.Errorf(ctx, "%v", err)
		} else if k, err := b.put(g); err != nil {
			aelog.Errorf(ctx, "%v", err)
		} else {
			http.Redirect(w, r, fmt.Sprintf("/bbs/%d/posts", k.IntID()), http.StatusSeeOther)
			return
		}
	}

	tmpl := template.Must(template.ParseFiles("tmpl/layout.html", "tmpl/new_bbs.html"))
	vars := map[string]interface{}{
		"err": "err",
	}
	if err := tmpl.ExecuteTemplate(w, "layout", vars); err != nil {
		aelog.Errorf(ctx, "%v", err)
	}
	w.WriteHeader(200)
}

func listPostsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	g := goon.NewGoon(r)

	b := new(bbs)
	if err := b.fromRequest(r); err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	} else if err := b.get(g); err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	}
	aelog.Errorf(ctx, "%v", b)

	tmpl := template.Must(template.ParseFiles("tmpl/layout.html", "tmpl/list_posts.html"))

	limit := 20
	ps := make(posts, 0, limit)
	var err error
	if err = ps.getAll(g, b, limit, ""); err != nil {
		aelog.Errorf(ctx, "%v", err)
	}
	vars := map[string]interface{}{
		"bbs":    b,
		"bbs_id": mux.Vars(r)["bbs_id"],
		"posts":  ps,
	}
	if err := tmpl.ExecuteTemplate(w, "layout", vars); err != nil {
		aelog.Errorf(ctx, "%v", err)
	}
}

func newPostHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	g := goon.NewGoon(r)

	b := new(bbs)
	if err := b.fromRequest(r); err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	} else if err := b.get(g); err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	}

	p := new(post)
	if err := p.fromRequest(r); err != nil {
		aelog.Errorf(ctx, "%v", err)
	} else if _, err = g.Put(p); err != nil {
		aelog.Errorf(ctx, "%v", err)
	} else {
		aelog.Infof(ctx, "p=%v", p)
		aelog.Infof(ctx, "post succeeded")
	}

	http.Redirect(w, r, fmt.Sprintf("/bbs/%d/posts", b.ID), http.StatusSeeOther)
}
