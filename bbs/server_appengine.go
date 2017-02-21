// How to deploy:
//   $ goapp deploy -application <application_id> [-version <version_number>]

// +build appengine

package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/appengine"
	aelog "google.golang.org/appengine/log"

	"github.com/gorilla/mux"
	"github.com/mjibson/goon"
)

func parseHTML(wr io.Writer, filename string, data interface{}) error {
	// TemplateFuncs are stolen from revel framework.
	// See: https://github.com/revel/revel/blob/master/template.go
	TemplateFuncs := map[string]interface{}{
		// Replaces newlines with <br>
		"nl2br": func(text string) template.HTML {
			return template.HTML(strings.Replace(template.HTMLEscapeString(text), "\n", "<br>", -1))
		},

		// Skips sanitation on the parameter.  Do not use with dynamic data.
		"raw": func(text string) template.HTML {
			return template.HTML(text)
		},
	}
	tmpl, err := template.New("base").Funcs(TemplateFuncs).ParseFiles("tmpl/layout.html", filename)
	if err != nil {
		return err
	}
	return tmpl.ExecuteTemplate(wr, "layout", data)
}

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
	r.HandleFunc(`/bbs{_dummy:/?}`, newBbsHandler)
	r.HandleFunc(`/bbs/{bbs_id:[0-9]+}/posts`, listPostsHandler).Methods(`GET`)
	r.HandleFunc(`/bbs/{bbs_id:[0-9]+}/posts`, newPostHandler).Methods(`POST`)
	r.PathPrefix(`/assets/`).Handler(http.FileServer(http.Dir(`./`)))
	r.NotFoundHandler = http.HandlerFunc(notFoundHandler)
	http.Handle(`/`, r)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	if err := parseHTML(w, "tmpl/error.html", "404 Not Found"); err != nil {
		aelog.Errorf(ctx, "%v", err)
	}
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

	// output HTML
	vars := map[string]interface{}{
		"err": "err",
	}
	if err := parseHTML(w, "tmpl/new_bbs.html", vars); err != nil {
		aelog.Errorf(ctx, "%v", err)
	}
}

func listPostsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	g := goon.NewGoon(r)

	b := new(bbs)
	if err := b.fromString(mux.Vars(r)["bbs_id"]); err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	} else if err := b.get(g); err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	}

	limit := 10
	ps := make(posts, 0, limit)
	nextCur, err := ps.getAll(g, b, limit, r.FormValue("offset"))
	if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	}

	// output HTML
	vars := map[string]interface{}{
		"bbs":      b,
		"bbs_id":   mux.Vars(r)["bbs_id"],
		"posts":    ps,
		"next_cur": nextCur,
	}
	if err := parseHTML(w, "tmpl/list_posts.html", vars); err != nil {
		aelog.Errorf(ctx, "%v", err)
	}
}

func newPostHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	g := goon.NewGoon(r)

	b := new(bbs)
	if err := b.fromString(mux.Vars(r)["bbs_id"]); err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	} else if err := b.get(g); err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	}

	p := new(post)
	if err := p.fromRequest(r, b); err != nil {
		aelog.Errorf(ctx, "%v", err)
	} else if _, err = g.Put(p); err != nil {
		aelog.Errorf(ctx, "%v", err)
	} else {
		aelog.Infof(ctx, "p=%v", p)
		aelog.Infof(ctx, "post succeeded")
	}

	http.Redirect(w, r, fmt.Sprintf("/bbs/%d/posts", b.ID), http.StatusSeeOther)
}
