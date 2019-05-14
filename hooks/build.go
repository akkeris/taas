package hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	akkeris "taas/jobs"
	structs "taas/structs"

	vault "github.com/akkeris/vault-client"
	"github.com/davecgh/go-spew/spew"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
)

func BuildHook(payload structs.BuildPayload, berr binding.Errors, r render.Render) {
	if berr != nil {
		fmt.Println("Bad Request")
		fmt.Println(berr)
		return
	}
	spew.Dump(payload)
	if payload.Build.Result != "pending" {
		buildinfo, err := getBuildInfo(payload)
		if err != nil {
			fmt.Println(err)
		}

		buildinfo.App.Name = payload.App.Name
		buildinfo.Space.Name = payload.Space.Name
		org, err := akkeris.GetAppControllerOrg(buildinfo.App.Name + "-" + buildinfo.Space.Name)
		if err != nil {
			fmt.Println(err)
		}
		buildinfo.Organization = org
		spew.Dump(buildinfo)
		err = writeBuildOutputES(buildinfo)
		if err != nil {
			fmt.Println(err)
		}
	}
	r.Text(200, "Done\n")
}

func getBuildInfo(payload structs.BuildPayload) (b structs.BuildInfo, e error) {
	var buildinfo structs.BuildInfo
	req, err := http.NewRequest("GET", os.Getenv("APP_CONTROLLER_URL")+"/apps/"+payload.App.Name+"-"+payload.Space.Name+"/builds/"+payload.Build.ID, nil)
	req.Header.Add("Authorization", vault.GetField(os.Getenv("APP_CONTROLLER_AUTH_SECRET"), "authorization"))
	if err != nil {
		fmt.Println(err)
		return buildinfo, err
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return buildinfo, err
	}
	fmt.Println(resp.Status)
	defer resp.Body.Close()
	bb, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(bb))
	if err != nil {
		fmt.Println(err)
		return buildinfo, err
	}
	err = json.Unmarshal(bb, &buildinfo)
	return buildinfo, nil
}

func writeBuildOutputES(buildinfo structs.BuildInfo) error {

	var buildessend structs.BuildESSend
	buildessend.App = buildinfo.App.Name
	buildessend.Space = buildinfo.Space.Name
	buildessend.ID = buildinfo.ID
	buildessend.Version = buildinfo.SourceBlob.Version
	buildessend.Commit = buildinfo.SourceBlob.Commit
	buildessend.Status = buildinfo.Status
	buildessend.UpdatedAt = buildinfo.UpdatedAt
	buildessend.Organization = buildinfo.Organization
	spew.Dump(buildessend)
	p, err := json.Marshal(buildessend)
	if err != nil {
		fmt.Println(err)
		return err
	}

	req, err := http.NewRequest("PUT", os.Getenv("ES_URL")+"/builds/output/"+buildessend.ID, bytes.NewBuffer(p))
	if err != nil {
		fmt.Println(err)
		return err
	}
	req.Header.Add("Content-type", "application/json")
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

/*
   App string `json:"app"`
   Space string `json:"space"`
   ID string `json:"buildid"`
   Version string `json:"version"`
   Commit string `json:"commit"`
   Status string `json:"status"`
   UpdatedAt string `json:"updatedhrtimestamp"`




   App struct {
       Name string `json:"name"`
       ID string `json:"id"`
   } `json:"app"`
   Space struct {
       Name string `json:"name"`
   } `json:"space"`
   CreatedAt time.Time `json:"created_at"`
   ID string `json:"id"`
   SourceBlob struct {
       Version string `json:"version"`
       Commit string `json:"commit"`
   } `json:"source_blob"`
   Status string `json:"status"`
   UpdatedAt time.Time `json:"updated_at"`

*/
