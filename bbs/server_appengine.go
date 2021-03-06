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
	"google.golang.org/appengine/internal"
	aelog "google.golang.org/appengine/log"

	"github.com/gorilla/mux"
	"github.com/mjibson/goon"
)

func parseHTML(wr io.Writer, filename string, data interface{}) error {
	// TemplateFuncs are stolen from revel framework.
	// See: https://github.com/revel/revel/blob/master/template.go
	TemplateFuncs := map[string]interface{}{
		"set": func(renderArgs map[string]interface{}, key string, value interface{}) template.JS {
			renderArgs[key] = value
			return template.JS("")
		},
		"default": func(defVal interface{}, args ...interface{}) (interface{}, error) {
			if len(args) >= 2 {
				return nil, fmt.Errorf("wrong number of args for default: want 2 got %d", len(args)+1)
			}
			args = append(args, defVal)
			for _, val := range args {
				switch val.(type) {
				case nil:
					continue
				case string:
					if val == "" {
						continue
					}
					return val, nil
				default:
					return val, nil
				}
			}
			return nil, nil
		},
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
	if data == nil {
		data = map[string]interface{}{}
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
	r.HandleFunc(`/`, topHandler)
	r.HandleFunc(`/about{_dummy:/?}`, aboutHandler)
	r.HandleFunc(`/contact{_dummy:/?}`, contactHandler)
	r.HandleFunc(`/bbs{_dummy:/?}`, newBbsHandler)
	r.HandleFunc(`/bbs/{bbs_id:[0-9]+}/posts`, listPostsHandler).Methods(`GET`)
	r.HandleFunc(`/bbs/{bbs_id:[0-9]+}/posts`, newPostHandler).Methods(`POST`)
	r.PathPrefix(`/assets/`).Handler(http.FileServer(http.Dir(`./`)))
	r.NotFoundHandler = http.HandlerFunc(notFoundHandler)
	http.Handle(`/`, r)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	vars := map[string]interface{}{
		"error": "404 Not Found",
	}
	if err := parseHTML(w, "tmpl/error.html", vars); err != nil {
		aelog.Errorf(ctx, "%v", err)
	}
}

func topHandler(w http.ResponseWriter, r *http.Request) {
	rawCtx := appengine.NewContext(r)
	aelog.Errorf(rawCtx, "rawCtx=%v", rawCtx)
	ctx, err := appengine.Namespace(rawCtx, "hogehoge")
	if err != nil {
		aelog.Errorf(ctx, "%v", err)
	}
	aelog.Errorf(ctx, "ctx=%v", ctx)

	g := goon.FromContext(ctx)
	limit := 5

	aelog.Infof(ctx, "ns=%v", internal.NamespaceFromContext(ctx))

	var bs []*bbs
	if err := getNewBbss(g, &bs, limit); err != nil {
		aelog.Errorf(ctx, "%v", err)
	}
	var ps []*post
	if err := getRecentPosts(g, &ps, limit, true); err != nil {
		aelog.Errorf(ctx, "%v", err)
	}
	for _, p := range ps {
		p.fetchBbs(g)
	}
	//aelog.Infof(ctx, "ps=%v", ps)
	//aelog.Infof(ctx, "bs=%v", bs)

	// output HTML
	vars := map[string]interface{}{
		"recentPosts": ps,
		"newBbss":     bs,
	}
	if err := parseHTML(w, "tmpl/top.html", vars); err != nil {
		aelog.Errorf(ctx, "%v", err)
	}
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	if err := parseHTML(w, "tmpl/about.html", nil); err != nil {
		aelog.Errorf(ctx, "%v", err)
	}
}

func contactHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	if err := parseHTML(w, "tmpl/contact.html", nil); err != nil {
		aelog.Errorf(ctx, "%v", err)
	}
}

func newBbsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	g := goon.NewGoon(r)

	if r.Method == "POST" {
		b, err := newBbsFromRequest(r)
		if err != nil {
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

	b, err := newBbsFromString(mux.Vars(r)["bbs_id"])
	if err != nil {
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

	b, err := newBbsFromString(mux.Vars(r)["bbs_id"])
	if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	} else if err := b.get(g); err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	}

	p, err := newPostFromRequest(r, b)
	if err != nil {
		aelog.Errorf(ctx, "%v", err)
	} else if _, err := g.Put(p); err != nil {
		aelog.Errorf(ctx, "%v", err)
	} else {
		aelog.Infof(ctx, "p=%v", p)
		aelog.Infof(ctx, "post succeeded")
	}

	http.Redirect(w, r, fmt.Sprintf("/bbs/%d/posts", b.ID), http.StatusSeeOther)
}
