package jobs

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	structs "taas/structs"

	"sort"

	_ "github.com/lib/pq"
)

func GetVars(job string, jobspace string) (v []structs.EnvironmentVariable, e error) {
	var envvars []structs.EnvironmentVariable

	req, err := http.NewRequest("GET", os.Getenv("AKKERIS_API_URL")+"/v1/config/set/"+job+"-"+jobspace+"-cs", nil)
	if err != nil {
		fmt.Println(err)
		return envvars, err
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return envvars, err
	}
	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return envvars, err
	}

	var configset structs.ConfigSpec
	err = json.Unmarshal(bodybytes, &configset)
	if err != nil {
		fmt.Println(err)
		return envvars, err
	}

	for _, element := range configset {
		var envvar structs.EnvironmentVariable
		envvar.Name = element.Varname
		envvar.Value = element.Varvalue
		envvars = append(envvars, envvar)
	}
	secretenvvars, err := getSecretVars(job, jobspace)
	if err != nil {
		fmt.Println(err)
		return envvars, err
	}
	envvars = append(envvars, secretenvvars...)

	sort.Slice(envvars, func(i, j int) bool {
		return envvars[i].Name < envvars[j].Name
	})

	return envvars, nil
}

func getSecretVars(job string, jobspace string) (v []structs.EnvironmentVariable, e error) {
	var envvars []structs.EnvironmentVariable
	uri := os.Getenv("PITDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
		return envvars, dberr
	}
	defer db.Close()
	stmt, err := db.Prepare("select bindname from appbindings where appname = $1 and space=$2 and bindtype='vault'")
	if err != nil {
		fmt.Println(err)
		return envvars, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(job, jobspace)
	var secrets []string
	for rows.Next() {
		var bindname string
		err := rows.Scan(&bindname)
		if err != nil {
			fmt.Println(err)
			return envvars, err
		}
		secrets = append(secrets, bindname)
	}
	db.Close()

	for _, secret := range secrets {
		vars, err := getSecret(secret)
		if err != nil {
			fmt.Println(err)
			return envvars, err
		}
		envvars = append(envvars, vars...)
	}
	return envvars, nil
}

func getSecret(secret string) (v []structs.EnvironmentVariable, e error) {
	var envvars []structs.EnvironmentVariable
	var secretvars []structs.KeyValuePair
	req, err := http.NewRequest("GET", os.Getenv("AKKERIS_API_URL")+"/v1/service/vault/credentials/"+secret, nil)
	if err != nil {
		fmt.Println(err)
		return envvars, err
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return envvars, err
	}
	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(bodybytes, &secretvars)
	if err != nil {
		fmt.Println(err)
		return envvars, err
	}
	for _, element := range secretvars {
		var envvar structs.EnvironmentVariable
		envvar.Name = element.Key
		envvar.Value = element.Value
		envvars = append(envvars, envvar)
	}

	return envvars, nil

}
