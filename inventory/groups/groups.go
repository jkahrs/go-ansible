package main

import (
	"encoding/json"
	"errors"
	"fmt"
	r "github.com/dancannon/gorethink"
	"github.com/jeffail/gabs"
	"log"
)

var (
	DbServer = "127.0.0.1:8081"
	DataBase = "ansible_installer"
	session  *r.Session
)

func init() {
	var err error
	session, err = r.Connect(r.ConnectOpts{
		Address:  DbServer,
		Database: DataBase,
	})
	if err != nil {
		log.Println(err)
		return
	}
	err = r.DbCreate(DataBase).Exec(session)
	if err != nil {
		return
	}

	_, err = r.Db(DataBase).TableCreate("groups").RunWrite(session)
	if err != nil {
		return
	}
}

type Group struct {
	Name string      `gorethink:"name"`
	ID   string      `gorethink:"id,omitempty"`
	Data interface{} `gorethink:"data"`
}

type ListResult struct {
	Result []DataStruct
}

type DataStruct struct {
	Hosts    []string    `gorethink:"hosts"`
	Vars     interface{} `gorethink:"vars,omitempty"`
	Children []string    `gorethink:"children,omitempty"`
}

func (g Group) validateReq() (err error) {
	if (len(g.Name) == 0) && (len(g.ID) == 0) {
		err = errors.New("request failed. Method requires g.Name or g.ID")
		return err
	}
	return nil
}

/*
	Retrieves g.ID by g.Name
	Requires g.Name
*/
func (g Group) getID() (id string, err error) {
	rows, err := r.Table("groups").Filter(r.Row.Field("name").Eq(g.Name)).Map(r.Row.Field("id")).Run(session)
	if err != nil {
		return id, err
	}

	var IDs []string
	rows.All(&IDs)
	if len(IDs) == 0 {
		err = errors.New("request failed. Group not found")
		return id, err
	}
	// group names are uniqe, so I should be fine here
	return IDs[0], nil
}

/*
	Creates a new Group
*/
func (g Group) Add() (err error) {
	if len(g.Name) == 0 {
		err = errors.New("request failed. Method requires g.Name")
		return err
	}
	_, nameIsFree := g.getID()
	if nameIsFree == nil {
		err = errors.New("request failed. Group already exists")
		return err
	}
	_, err = r.Table("groups").Insert(g).RunWrite(session)
	if err != nil {
		return err
	}

	return nil
}

func (g Group) Delete() (err error) {
	err = g.validateReq()
	if err != nil {
		return err
	}
	if len(g.ID) == 0 {
		g.ID, err = g.getID()
		if err != nil {
			err = errors.New("request failed. Group not found")
			return err
		}
	}
	_, err = r.Table("groups").Get(g.ID).Delete().Run(session)
	if err != nil {
		log.Fatalln(err)
	}
	return nil
}

func List() (err error) {
	rows, err := r.Table("groups").Pluck("data").Run(session)
	if err != nil {
		return err
	}
	var result []interface{}
	rows.All(&result)

	if result == nil {
		err = errors.New("{}")
		return err
	}

	var out []interface{}
	for _, elem := range result {
		JSON, err := json.Marshal(elem)
		if err != nil {
			return err
		}
		// make the fields more accessible
		jsonParsed, err := gabs.ParseJSON(JSON)
		if err != nil {
			return err
		}
		add := jsonParsed.Path("data")
		out = append(out, add)
	}
	fmt.Println(out)

	return nil
}

func main() {

	var req Group
	req.Name = "testgroup"
	Data := gabs.New()
	Data.SetP("localhost", "testgroup.hosts")
	Data.SetP("ssh", "testgroup.vars.ansible_ssh_connection")
	req.Data = Data.Data()
	err := req.Add()

	if err != nil {
		fmt.Println(err)
	}

	var req2 Group
	req2.Name = "testgroup2"

	req2.Add()

	List()
}
