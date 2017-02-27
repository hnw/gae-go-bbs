package main

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/appengine/datastore"
	//aelog "google.golang.org/appengine/log"

	"github.com/mjibson/goon"
)

type bbs struct {
	ID        int64     `datastore:"-" goon:"id"`
	Name      string    `datastore:"name,noindex"`
	Descr     string    `datastore:"descr,noindex"`
	Theme     string    `datastore:"theme,noindex"`
	CreatedAt time.Time `datastore:"created_at,noindex"`
	UpdatedAt time.Time `datastore:"updated_at,noindex"`
}

func newBbsFromString(s string) (*bbs, error) {
	bbsID, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, err
	}
	return &bbs{ID: bbsID}, nil
}

func newBbsFromRequest(r *http.Request) (b *bbs, err error) {
	now := time.Now()
	b = &bbs{
		Name:      r.PostFormValue("bbs_name"),
		Descr:     r.PostFormValue("bbs_descr"),
		Theme:     r.PostFormValue("theme"),
		CreatedAt: now,
		UpdatedAt: now,
	}
	return b, nil
}

func (b *bbs) get(g *goon.Goon) error {
	if b.ID == 0 {
		return errors.New("b.ID == 0")
	} else if b.Name != "" {
		return errors.New(`b.Name != ""`)
	} else if err := g.Get(b); err != nil {
		return err
	}
	return nil
}

func (b *bbs) put(g *goon.Goon) (*datastore.Key, error) {
	var k *datastore.Key
	var err error
	if b.ID != 0 {
		return nil, errors.New("b.ID != 0")
	} else if b.Name == "" {
		return nil, errors.New(`b.Name == ""`)
	} else if k, err = g.Put(b); err != nil {
		return nil, err
	}
	return k, nil
}
