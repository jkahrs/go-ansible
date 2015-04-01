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
		log.Fatalln(err)
		return
	}
}

type Host struct {
	Name string
	ID   string
	Data interface{}
}

/*
	Basic request validation
*/
func (h Host) validateReq() (err error) {
	if (len(h.ID) == 0) && (len(h.Name) == 0) {
		err := errors.New("function requires h.Name or h.ID (string)")
		return err
	}
	return nil
}

func (h Host) getID() (hostID string, err error) {
	err = h.validateReq()
	if err != nil {
		return "", err
	}
	// ID is already set..
	if len(h.ID) > 0 {
		return h.ID, nil
	}
	// get the ID by h.Name
	rows, err := r.Table("hosts").Filter(r.Row.Field("name").Eq(h.Name)).Pluck("id").Run(session)
	if err != nil {
		return "", err
	}
	type response struct {
		Id string `gorethink:"id"`
	}
	var IDs []response
	err = rows.All(&IDs)
	if err != nil {
		log.Println(err)
	}
	if IDs == nil {
		err = errors.New("host not found")
		return "", err
	}
	h.ID = IDs[0].Id

	return h.ID, nil
}

/*
	Get JSON hash/dict of the host_vars for h.Name/ID

	fulfills inventory --list {{ host_name }} requirements see:
	http://docs.ansible.com/developing_inventory.html#script-conventions
*/
func (h Host) ListVars() (vars []byte, err error) {
	err = h.validateReq()
	if err != nil {
		return nil, err
	}
	// I prefer searching by ID
	if len(h.ID) == 0 {
		h.ID, err = h.getID()
		if err != nil {
			err = errors.New("{}")
			return nil, err
		}
	}

	// get the Host from the DB eithber by name or by ID
	rows, err := r.Table("hosts").Get(h.ID).Run(session)
	if err != nil {
		return nil, err
	}

	var HostInfo []interface{}
	rows.All(&HostInfo)

	if HostInfo == nil {
		err := errors.New("{}")
		return nil, err
	}

	JSON, err := json.Marshal(HostInfo[0])
	if err != nil {
		return nil, err
	}

	return JSON, nil
}

/*
	Update a Host / Vars by h.Name / h.ID

	ToDo: data interface{} seems kind of ugly here.. I need to think about this
*/
func (h Host) Update(data interface{}) (err error) {
	err = h.validateReq()
	if err != nil {
		return err
	}
	// get the ID by Name
	if len(h.ID) == 0 {
		h.ID, err = h.getID()
		if err != nil {
			return err
		}
	}

	_, err = r.Table("hosts").Get(h.ID).Update(data).RunWrite(session)
	if err != nil {
		return err
	}

	return nil
}

/*
	(Try to..) Delete a Host by h.Name / h.ID
*/
func (h Host) Delete() (err error) {
	err = h.validateReq()
	if err != nil {
		return err
	}
	// get the ID by Name
	if len(h.ID) == 0 {
		h.ID, err = h.getID()
		if err != nil {
			return err
		}
	}

	_, err = r.Table("hosts").Get(h.ID).Delete().Run(session)
	if err != nil {
		return err
	}

	return nil
}

func (h Host) Add() (err error) {
	if len(h.Name) == 0 {
		err = errors.New("create failed. Method requires h.Name")
		return err
	}
	// check for preexisting host with this name
	id, _ := h.getID()
	if len(id) > 0 {
		log.Println(id)
		err = errors.New("add failed. Host already exists.")
		return err
	}

	// create a new entry object
	type newHost struct {
		Name string
	}
	var entry newHost
	entry.Name = h.Name
	_, err = r.Table("hosts").Insert(entry).Run(session)
	if err != nil {
		return err
	}
	return nil
}

func prettyPrintVars(vars []byte) (err error) {
	// make the fields more accessible
	jsonParsed, err := gabs.ParseJSON(vars)
	if err != nil {
		return err
	}
	// debug: prettyprint
	fmt.Println(jsonParsed.Path("vars").StringIndent("", "  "))
	return nil
}

func main() {
	// ex. Update a Host
	var req2 Host
	req2.Name = "localhost3"
	//req2.ID = "727403fb-7930-4b89-a6fd-81b81e30eb4e"

	// create a data Object and provide the changes

	//Data := gabs.New()
	//Data.SetP("localhost", "vars.ansible_host_name")
	//Data.SetP("localhost", "name")
	//req2.Data = Data.Data()
	//err := req2.Update(req2.Data)
	//if err != nil {
	//	log.Println(err)
	//}

	//req2.Delete()

	err := req2.Add()
	if err != nil {
		log.Println(err)
	}

	vars, err := req2.ListVars()
	if err != nil {
		fmt.Println(err)
	}
	prettyPrintVars(vars)
}
