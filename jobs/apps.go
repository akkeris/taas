package jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	structs "taas/structs"

	vault "github.com/akkeris/vault-client"
)
func GetMostRecentReleaseID(diagnostic structs.DiagnosticSpec)(r string){
        req, err := http.NewRequest("GET", os.Getenv("APP_CONTROLLER_URL")+"/apps/"+diagnostic.App+"-"+diagnostic.Space+"/releases", nil)
        req.Header.Add("Authorization", vault.GetField(os.Getenv("APP_CONTROLLER_AUTH_SECRET"), "authorization"))
        if err != nil {
                fmt.Println(err)
                return ""
        }
        client := http.Client{}
        resp, err := client.Do(req)
        if err != nil {
                fmt.Println(err)
                return ""
        }
        defer resp.Body.Close()
        bodybytes, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                fmt.Println(err)
                return ""
        }
        var releases structs.Releases
        err = json.Unmarshal(bodybytes, &releases)
        if err != nil {
                fmt.Println(err)
                return ""
        }
        return releases[len(releases)-1].ID
}               

func GetVersion(app string, space string, buildid string) (s string, e error) {

	fmt.Println(app)
	fmt.Println(space)
	fmt.Println(buildid)

	req, err := http.NewRequest("GET", os.Getenv("APP_CONTROLLER_URL")+"/apps/"+app+"-"+space+"/builds/"+buildid, nil)
	req.Header.Add("Authorization", vault.GetField(os.Getenv("APP_CONTROLLER_AUTH_SECRET"), "authorization"))
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	if resp.StatusCode != 404 {
		defer resp.Body.Close()
		bodybytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return "", err
		}
		fmt.Println(string(bodybytes))
		var buildinfo structs.BuildSpec
		err = json.Unmarshal(bodybytes, &buildinfo)
		if err != nil {
			fmt.Println(err)
			return "", err
		}
		fmt.Println(buildinfo.SourceBlob.Version)
		return buildinfo.SourceBlob.Version, nil
	} else {
		return "", nil
	}

}

func GetAppControllerOrg(app string) (o string, e error) {
	var org string
	req, err := http.NewRequest("GET", os.Getenv("APP_CONTROLLER_URL")+"/v1/apps/"+app, nil)
	req.Header.Set("Authorization", vault.GetField(os.Getenv("APP_CONTROLLER_AUTH_SECRET"), "authorization"))
	if err != nil {
		fmt.Println(err)
		return org, err
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return org, err
	}
	if resp.StatusCode == 404 {
		return os.Getenv("DEFAULT_ORG"), nil
	}
	defer resp.Body.Close()
	bb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return org, err
	}
	fmt.Println(string(bb))
	var appcontrollerapp structs.AppControllerApp
	err = json.Unmarshal(bb, &appcontrollerapp)
	if err != nil {
		fmt.Println(err)
		return org, err
	}
	fmt.Println(appcontrollerapp.Organization.Name)
	org = appcontrollerapp.Organization.Name
	return org, nil
}

func IsValidSpace(space string) (v bool, e error) {
	req, err := http.NewRequest("GET", os.Getenv("APP_CONTROLLER_URL")+"/spaces/"+space, nil)
	req.Header.Add("Authorization", vault.GetField(os.Getenv("APP_CONTROLLER_AUTH_SECRET"), "authorization"))
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	if resp.StatusCode == 200 {
		return true, nil
	} else {
		return false, nil
	}
}

func IsProtectedSpace(space string) (p bool, err error) {
	req, err := http.NewRequest("GET", os.Getenv("APP_CONTROLLER_URL")+"/spaces/"+space, nil)
	req.Header.Add("Authorization", vault.GetField(os.Getenv("APP_CONTROLLER_AUTH_SECRET"), "authorization"))
	if err != nil {
		fmt.Println("1")
		fmt.Println(err)
		return false, err
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("2")
		fmt.Println(err)
		return false, err
	}
	var spaceinfo structs.SpaceInfo
	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("3")
		fmt.Println(err)
		return false, err
	}
	err = json.Unmarshal(bodybytes, &spaceinfo)
	if err != nil {
		fmt.Println("4")
		fmt.Println(err)
		return false, err
	}
	var toreturn bool
	toreturn = false
	for _, element := range spaceinfo.Compliance {
		if element == "socs" {
			toreturn = true
		}
	}
	return toreturn, nil
}

func GetHooks(app string) (h []structs.AppHook, e error) {
	req, err := http.NewRequest("GET", os.Getenv("APP_CONTROLLER_URL")+"/apps/"+app+"/hooks", nil)
	req.Header.Set("Authorization", vault.GetField(os.Getenv("APP_CONTROLLER_AUTH_SECRET"), "authorization"))
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	if resp.StatusCode == 404 {
		return nil, errors.New("App not found")
	}

	defer resp.Body.Close()
	bb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	var hooks []structs.AppHook
	err = json.Unmarshal(bb, &hooks)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return hooks, nil
}

func DeleteHook(app string, hookid string) (e error) {
	req, err := http.NewRequest("DELETE", os.Getenv("APP_CONTROLLER_URL")+"/apps/"+"app"+"/hooks/"+hookid, nil)
	req.Header.Set("Authorization", vault.GetField(os.Getenv("APP_CONTROLLER_AUTH_SECRET"), "authorization"))
	if err != nil {
		fmt.Println(err)
		return err
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(string(bodybytes))
	return nil
}
