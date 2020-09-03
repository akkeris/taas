package pipelines

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	structs "taas/structs"
	"taas/utils"
)

func GetPipeline(pipelinename string) (p structs.PipelineSpec, e error) {
	var pipeline structs.PipelineSpec
	appcontrollerurl := os.Getenv("APP_CONTROLLER_URL") + "/pipelines/" + pipelinename + "/pipeline-couplings"
	req, err := http.NewRequest("GET", appcontrollerurl, nil)
	if err != nil {
		fmt.Println(err)
		return pipeline, err
	}
	req.Header.Add("Content-type", "application/json")
	req.Header.Set("Authorization", os.Getenv("APP_CONTROLLER_AUTH"))

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

func PromoteApp(promotion structs.PromotionSpec) (s string, e error) {
	p, err := json.Marshal(promotion)
	if err != nil {
		fmt.Println(err)
		return "failed", err
	}
	appcontrollerurl := os.Getenv("APP_CONTROLLER_URL") + "/pipeline-promotions"
	req, err := http.NewRequest("POST", appcontrollerurl, bytes.NewBuffer(p))
	if err != nil {
		fmt.Println(err)
		return "failed", err
	}
	req.Header.Add("Content-type", "application/json")
	req.Header.Set("Authorization", os.Getenv("APP_CONTROLLER_AUTH"))

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return "failed", err
	}
	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return "failed", err
	}
	utils.PrintDebug(string(bodybytes))
	var promotestatus structs.PromoteStatus
	err = json.Unmarshal(bodybytes, &promotestatus)
	if err != nil {
		fmt.Println(err)
		return "failed.  " + string(bodybytes), err
	}

	return promotestatus.Status, nil
}
