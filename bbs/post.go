package main

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/appengine/datastore"

	"github.com/gorilla/mux"
	"github.com/mjibson/goon"
)

type post struct {
	ID        int64     `datastore:"-" goon:"id"`
	BbsID     int64     `datastore:"bbs_id"`
	UserName  string    `datastore:"user_name,noindex"`
	Subject   string    `datastore:"subject,noindex"`
	Message   string    `datastore:"message,noindex"`
	IPAddr    string    `datastore:"ip_addr,noindex"`
	CreatedAt time.Time `datastore:"created_at,noindex"`
	UpdatedAt time.Time `datastore:"updated_at"`
}

type posts []*post

func (p *post) fromKey(k *datastore.Key) error {
	p.ID = k.IntID()
	return nil
}

func (p *post) fromRequest(r *http.Request) error {
	now := time.Now()

	bbsID, err := strconv.ParseInt(mux.Vars(r)["bbs_id"], 10, 64)
	if err != nil {
		return err
	}
	p.BbsID = bbsID
	p.UserName = r.PostFormValue("user_name")
	p.Subject = r.PostFormValue("subject")
	p.Message = r.PostFormValue("message")
	p.IPAddr = r.RemoteAddr
	p.CreatedAt = now
	p.UpdatedAt = now

	return nil
}

func (p *post) get(g *goon.Goon) error {
	if p.ID == 0 {
		return errors.New("p.ID == 0")
	} else if p.Message != "" {
		return errors.New(`p.message != ""`)
	} else if err := g.Get(p); err != nil {
		return err
	}
	return nil
}

func (p *post) put(g *goon.Goon) (*datastore.Key, error) {
	var k *datastore.Key
	var err error
	if p.ID != 0 {
		return nil, errors.New("p.ID != 0")
	} else if p.Message == "" {
		return nil, errors.New(`p.Message == ""`)
	} else if k, err = g.Put(p); err != nil {
		return nil, err
	}
	return k, nil
}

func (ptr *posts) getAll(g *goon.Goon, b *bbs, limit int, encodedCursor string) error {
	ps := *ptr

	q := datastore.NewQuery("post").KeysOnly().Filter("bbs_id =", b.ID).Order("-updated_at")
	if encodedCursor != "" {
		cursor, err := datastore.DecodeCursor(encodedCursor)
		if err == nil {
			q = q.Start(cursor)
		}
	}
	// Iterate over the results.
	t := g.Run(q)
	for {
		k, err := t.Next(nil)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return err
		}
		p := new(post)
		if err := p.fromKey(k); err != nil {
			return err
		}
		ps = append(ps, p)
	}
	if err := g.GetMulti(ps); err != nil {
		return err
	}

	*ptr = ps
	return nil
}
