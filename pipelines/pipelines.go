package pipelines

import (
	structs "alamo-self-diagnostics/structs"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
        "os"
        vault "scc-gitlab-1.dev.octanner.net/octanner/octvault"
)

func GetPipeline(pipelinename string) (p structs.PipelineSpec, e error) {
	var pipeline structs.PipelineSpec
	appcontrollerurl := "https://api.alamoapp.octanner.io/pipelines/" + pipelinename + "/pipeline-couplings"
	req, err := http.NewRequest("GET", appcontrollerurl, nil)
	if err != nil {
		fmt.Println(err)
		return pipeline, err
	}
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", vault.GetField(os.Getenv("APP_CONTROLLER_AUTH_SECRET"),"authorization"))

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return pipeline, err
	}
	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return pipeline, err
	}

	err = json.Unmarshal(bodybytes, &pipeline)
	if err != nil {
		fmt.Println(err)
		return pipeline, err
	}
	return pipeline, nil

}

func PromoteApp(promotion structs.PromotionSpec) (e error) {
	p, err := json.Marshal(promotion)
	if err != nil {
		fmt.Println(err)
		return err
	}
	appcontrollerurl := "https://api.alamoapp.octanner.io/pipeline-promotions"
	req, err := http.NewRequest("POST", appcontrollerurl, bytes.NewBuffer(p))
	if err != nil {
		fmt.Println(err)
		return err
	}
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", vault.GetField(os.Getenv("APP_CONTROLLER_AUTH_SECRET"),"authorization"))

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
