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

func (h Host) validateReq() (err error) {
	if (len(h.ID) == 0) && (len(h.Name) == 0) {
		err := errors.New("function requires h.Name or h.ID (string)")
		return err
	}
	return nil
}

// fulfills inventory --list {{ host_name }} requrements
func (h Host) ListVars() (err error) {
	err = h.validateReq()
	if err != nil {
		return err
	}
	// prefer searching by ID
	var MatchBy, Field string
	if len(h.ID) > 0 {
		MatchBy = h.ID
		Field = "id"
	} else {
		MatchBy = h.Name
		Field = "name"
	}
	// get the Host from the DB eithber by name or by ID
	rows, err := r.Table("hosts").Filter(r.Row.Field(Field).Eq(MatchBy)).Run(session)
	if err != nil {
		return err
	}

	var HostInfo []interface{}
	rows.All(&HostInfo)

	if HostInfo == nil {
		err := errors.New("{}")
		return err
	}

	JSON, err := json.Marshal(HostInfo[0])
	if err != nil {
		return err
	}

	// make the fields more accessible
	jsonParsed, err := gabs.ParseJSON(JSON)
	if err != nil {
		return err
	}

	fmt.Println(jsonParsed.Path("vars").StringIndent("", "  "))
	return nil
}

func (h Host) Update(data interface{}) (err error) {
	err = h.validateReq()
	if err != nil {
		return err
	}
	// get the ID by Name
	if len(h.ID) == 0 {
		rows, err := r.Table("hosts").Filter(r.Row.Field("name").Eq(h.Name)).Pluck("id").Run(session)
		if err != nil {
			return err
		}
		var IDs []string
		rows.All(&IDs)
		h.ID = IDs[0]
	}

	_, err = r.Table("hosts").Get(h.ID).Update(data).RunWrite(session)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	// ex. Update a Host by ID
	var req2 Host
	req2.Name = "asdsdd"
	//req2.ID = "fcb8a618-92c8-4e53-a670-8b61b98a9c2f"
	// create a data Object and provide the changes
	Data := gabs.New()
	Data.SetP("local2", "vars.ansible_host_name")
	Data.SetP("local2", "name")
	req2.Data = Data.Data()

	//err := req2.Update(req2.Data)
	//if err != nil {
	//	log.Println(err)
	//}
	err := req2.ListVars()
	if err != nil {
		fmt.Println(err)
	}
}
