package main

import (
	"errors"
	"fmt"
	"strings"

	getopt "github.com/pborman/getopt/v2"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

/*
	Examples:
		,ban -soft -user @schism#1384 -c "comment"
			^-> ,user -ban -soft -user @schism#1384 -c "comment"
		,xfer -n 100 -user @user
			^-> ,user -x -n 100 -user @user
*/

// Unique errors for Aliases.
var (
	ErrNoAliases = errors.New("no aliases found")
)

// Syntax constants for Alias Commands.
const (
	aliasSyntaxAdd    = ",alias   --add      -i ub   -o \"user -ban\"\n"
	aliasSyntaxRemove = ",alias   --remove   -i ub\n"
	aliasSyntaxList   = ",alias   --list\n"
	aliasSyntaxAll    = "\n\n" + aliasSyntaxAdd + aliasSyntaxRemove + aliasSyntaxList
)

// CoreAlias processes creating and destroying new aliases.
func (dat *IOdata) CoreAlias() error {
	u := dat.user
	var help, list, add, remove bool
	var caller, linked string

	fl := getopt.New()

	fl.FlagLong(&help, "help", 'h', "Help Text")
	fl.FlagLong(&list, "list", 'l', "List all Aliases")
	fl.FlagLong(&add, "add", 'a', "Add")
	fl.FlagLong(&remove, "remove", 'r', "Remove")
	fl.Flag(&caller, 'i', "Input (Alias) text")
	fl.Flag(&linked, 'o', "Original (What it is referring to)")

	if err := fl.Getopt(dat.io, nil); err != nil {
		return err
	}
	if fl.NArgs() > 0 {
		if err := fl.Getopt(fl.Args(), nil); err != nil {
			return err
		}
	}

	// Return if the user does not have the role
	if ok := dat.user.HasRoleType(dat.guildConfig, rolePermissionMod); !ok {
		return ErrBadPermissions
	}

	// Empty help to skip to end of script to print.
	if add || remove {
		if caller == "" {
			return errors.New("bad alias name")
		}
		alias := AliasNew(caller, linked, dat.guild.ID, u)
		if add {
			if linked == "" {
				return errors.New("bad original command")
			}
			if err := alias.Update(); err != nil {
				return err
			}
			// Alias added at this point.
			msg := fmt.Sprintf("%s added an alias. \n**%s** -> **%s**", u.StringPretty(), caller, linked)
			dat.msgEmbed = embedCreator(msg, ColorGreen)
			return nil
		} else if remove {
			if err := alias.Remove(); err != nil {
				return err
			}
			msg := fmt.Sprintf("%s removed the **%s** alias.", u.StringPretty(), caller)
			dat.msgEmbed = embedCreator(msg, ColorMaroon)
			return nil
		}
	} else if list {
		var err error
		alias := AliasNew("", "", dat.guild.ID, u)
		dat.output, err = alias.List()
		if err != nil {
			return err
		}
		return nil
	}

	dat.output = Help(fl, "", aliasSyntaxAll)
	return nil
}

// AliasNew returns a new Alias Object.
func AliasNew(caller, link, serverID string, user *User) *Alias {
	return &Alias{
		Caller:   caller,
		Linked:   link,
		ServerID: serverID,
		AddedBy:  user.Basic(),
	}
}

// Update an alias into the database.
func (a *Alias) Update() error {
	var q = make(map[string]interface{})
	var c = make(map[string]interface{})
	q["caller"] = a.Caller
	c["$set"] = bson.M{
		"linked":  a.Linked,
		"addedby": a.AddedBy,
	}

	dbdat := DBdataCreate(a.ServerID, CollectionAlias, a, q, c)
	err := dbdat.dbEdit(Alias{})
	if err != nil {
		if err == mgo.ErrNotFound {
			err := dbdat.dbInsert()
			return err
		}
		return err
	}

	return nil
}

// Get an alias from database.
func (a *Alias) Get() error {
	var q = make(map[string]interface{})
	q["caller"] = a.Caller

	dbdat := DBdataCreate(a.ServerID, CollectionAlias, Alias{}, q, nil)
	err := dbdat.dbGet(Alias{})
	if err != nil {
		return err
	}

	var alias Alias
	alias = dbdat.Document.(Alias)

	a.ID = alias.ID
	a.Caller = alias.Caller
	a.Linked = alias.Linked
	a.AddedBy = alias.AddedBy

	return nil
}

// GetAll aliases from database.
func (a *Alias) GetAll() ([]Alias, error) {
	db := DBdataCreate(a.ServerID, CollectionAlias, Alias{}, nil, nil)
	if err := db.dbGetAll(Alias{}); err != nil {
		return nil, err
	}

	if len(db.Documents) == 0 {
		return nil, ErrNoAliases
	}

	var aliases []Alias
	var alias Alias
	for _, a := range db.Documents {
		alias = a.(Alias)
		aliases = append(aliases, alias)
	}
	return aliases, nil
}

// Remove an alias from the database.
func (a *Alias) Remove() error {
	if err := a.Get(); err != nil {
		return nil
	}

	db := DBdataCreate(a.ServerID, CollectionAlias, a, nil, nil)
	err := db.dbDeleteID(a.ID)
	return err
}

// Check if an alias exists in a database.
func (a *Alias) Check() (string, error) {
	if err := a.Get(); err != nil {
		return "", err
	}
	return a.Linked, nil
}

// List prints out all currently accessible links.
func (a *Alias) List() (string, error) {
	aliases, err := a.GetAll()
	if err != nil {
		return "", err
	}

	var msg = "Current aliases:\n"
	for _, a := range aliases {
		msg += fmt.Sprintf("\n\t[%s]   ->   [%s]", a.Caller, a.Linked)
	}
	msg += "\n\nUse [left] text to perform [right] text."
	return "```" + msg + "```", nil
}

// Convert a Caller to a Link and return new io.io
func aliasConv(dat *IOdata, linked string) []string {
	newTxt := strings.Replace(dat.input, dat.io[0], linked, 1)
	_, cmds := strToCommands(newTxt, dat.cmdPrefix)
	return cmds
}
