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
	CreatedAt time.Time `datastore:"created_at"`
	UpdatedAt time.Time `datastore:"updated_at"`
}

func newBbsFromKey(k *datastore.Key) *bbs {
	b := &bbs{
		ID: k.IntID(),
	}
	return b
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

func getNewBbss(g *goon.Goon, ptr *[]*bbs, limit int) error {
	bs := make([]*bbs, 0, limit)
	q := datastore.NewQuery("bbs").KeysOnly().Order("-created_at").Limit(limit)

	// Iterate over the results.
	t := g.Run(q)
	//aelog.Infof(g.Context, "g.Run() finished")
	for {
		k, err := t.Next(nil)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return err
		}
		bs = append(bs, newBbsFromKey(k))
	}
	if err := g.GetMulti(bs); err != nil {
		return err
	}

	*ptr = bs
	return nil
}
