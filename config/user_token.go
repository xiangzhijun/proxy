package config

import (
	"encoding/json"
	"io/ioutil"
)

type User struct {
	UserTokenMap map[string]string
}

func (u *User) ReadUserTokenMap(file_name string) (err error) {
	data, err := ioutil.ReadFile(file_name)
	if err != nil {
		return err
	}

	var temp map[string]string
	err = json.Unmarshal(data, &temp)
	u.UserTokenMap = temp
	return

}

func (u *User) WriteUserTokenMap(file_name string) (err error) {
	data, err := json.Marshal(u.UserTokenMap)
	if err != nil {
		return err
	}

	backup, _ := ioutil.ReadFile(file_name)
	ioutil.WriteFile(file_name+".backup", backup, 0661)

	err = ioutil.WriteFile(file_name, data, 0661)
	return
}
