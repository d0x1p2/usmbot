package main

import (
	"errors"

	"github.com/d0x1p2/godbot"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Error constants for database issues.
var (
	ErrNilInterface = errors.New("nil interface provided")
	ErrUnknownType  = errors.New("unknown type")
)

// DBdatCreate creates a database object used to get exchange information with mongodb
func DBdatCreate(db, coll string, doc interface{}, q bson.M) *DBdat {
	return &DBdat{Database: db, Collection: coll, Document: doc, Query: q}
}

func (d *DBdat) dbInsert() error {

	if d.Document == nil {
		return ErrNilInterface
	}

	mdb, err := mgo.Dial(envDBUrl)
	if err != nil {
		return err
	}
	defer mdb.Close()

	//mdb.SetMode(mgo.Monotonic, true)

	c := mdb.DB(d.Database).C(d.Collection)
	err = c.Insert(d.Document)
	if err != nil {
		return err
	}
	return nil
}

func (d *DBdat) dbGet(i interface{}) error {

	if d.Query == nil {
		return ErrNilInterface
	}

	mdb, err := mgo.Dial(envDBUrl)
	if err != nil {
		return err
	}
	defer mdb.Close()

	c := mdb.DB(d.Database).C(d.Collection)
	err = c.Find(d.Query).One(&d.Document)
	if err != nil {
		return err
	}

	d.Document = i
	return nil
}

func (io *IOdat) dbCore() (err error) {
	var dbdat *DBdat
	var guild *godbot.Guild
	if Bot != nil {
		guild = Bot.GetGuild(Bot.GetChannel(io.msg.ChannelID).GuildID)
	}

	switch io.io[1] {
	case "event":
		var e = Event{Day: io.io[2], Time: io.io[3], Description: io.io[4], AddedBy: io.user}
		dbdat = DBdatCreate(guild.Name, "events", e, nil)
	}
	switch io.io[0] {
	case "add":
		err = dbdat.dbInsert()
	case "edit":
	case "del":
	}

	return nil
}