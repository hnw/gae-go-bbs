package main

import (
	"errors"
	"net/http"
	"time"

	"google.golang.org/appengine/datastore"
	//aelog "google.golang.org/appengine/log"

	"github.com/mjibson/goon"
)

type post struct {
	ID        int64     `datastore:"-" goon:"id"`
	Bbs       *bbs      `datastore:"-"`
	BbsID     int64     `datastore:"bbs_id"`
	UserName  string    `datastore:"user_name,noindex"`
	Subject   string    `datastore:"subject,noindex"`
	Message   string    `datastore:"message,noindex"`
	IPAddr    string    `datastore:"ip_addr,noindex"`
	CreatedAt time.Time `datastore:"created_at,noindex"`
	UpdatedAt time.Time `datastore:"updated_at"`
}

type posts []*post

func newPostFromKey(k *datastore.Key) *post {
	p := &post{
		ID: k.IntID(),
	}
	return p
}

func newPostFromRequest(r *http.Request, b *bbs) (*post, error) {
	now := time.Now()
	p := &post{
		BbsID:     b.ID,
		UserName:  r.PostFormValue("user_name"),
		Subject:   r.PostFormValue("subject"),
		Message:   r.PostFormValue("message"),
		IPAddr:    r.RemoteAddr,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return p, nil
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

func (p *post) fetchBbs(g *goon.Goon) {
	b := &bbs{ID: p.BbsID}
	if err := g.Get(b); err == nil {
		p.Bbs = b
	}
}

func (b *bbs) getPostsKeys(g *goon.Goon, ps []*post, from time.Time, limit int, descOrder bool) error {
	q := datastore.NewQuery("post").KeysOnly().Filter("bbs_id =", b.ID)

	if descOrder {
		if !from.IsZero() {
			q = q.Filter("updated_at <=", from)
		}
		q = q.Order("-updated_at")
	} else {
		if !from.IsZero() {
			q = q.Filter("updated_at >=", from)
		}
		q = q.Order("updated_at")
	}
	if limit != 0 {
		q = q.Limit(limit)
	}
	if _, err := g.GetAll(q, ps); err != nil {
		return err
	}
	return nil
}

func (ptr *posts) getAll(g *goon.Goon, b *bbs, limit int, encodedCursor string) (string, error) {
	ps := *ptr
	encCur := ""

	q := datastore.NewQuery("post").KeysOnly().Filter("bbs_id =", b.ID).Order("-updated_at")
	if encodedCursor != "" {
		cursor, err := datastore.DecodeCursor(encodedCursor)
		if err == nil {
			q = q.Start(cursor)
		}
	}
	// Iterate over the results.
	t := g.Run(q)
	//aelog.Infof(g.Context, "g.Run() finished")
	cnt := 0
	for {
		k, err := t.Next(nil)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return "", err
		}
		ps = append(ps, newPostFromKey(k))
		cnt++
		if cnt >= limit {
			cur, err := t.Cursor() // 内部的にAPIに問い合わせるため遅い
			if err != nil {
				return "", err
			}
			encCur = cur.String()
			break
		}
	}
	if err := g.GetMulti(ps); err != nil {
		return "", err
	}

	*ptr = ps

	return encCur, nil
}

func getRecentPosts(g *goon.Goon, ptr *[]*post, limit int, distinctOnBbs bool) error {
	ps := make([]*post, 0, limit)
	q := datastore.NewQuery("post").Project("bbs_id", "updated_at").Order("-updated_at")

	if !distinctOnBbs {
		q = q.Limit(limit)
	}

	t := q.Run(g.Context)
	cnt := 0
	occurred := make(map[int64]bool)
	for {
		p := &post{}
		k, err := t.Next(p)
		p.ID = k.IntID()
		if err == datastore.Done {
			break
		}
		if err != nil {
			return err
		}
		if distinctOnBbs && occurred[p.BbsID] {
			continue
		}
		//aelog.Infof(g.Context, "p=%v", p)
		ps = append(ps, p)
		cnt++
		occurred[p.BbsID] = true
		if cnt >= limit {
			break
		}
	}
	if err := g.GetMulti(ps); err != nil {
		return err
	}

	*ptr = ps
	return nil
}
