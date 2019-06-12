package jobs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	structs "taas/structs"
	"time"

	vault "github.com/akkeris/vault-client"
	shellwords "github.com/mattn/go-shellwords"
)

var client *http.Client

type Response struct {
	Status int
	Body   []byte
}

var kubernetestoken string

// StartClient - Get K8S token from Vault, initialize HTTP client
func StartClient() {
	kubernetestoken = vault.GetField(os.Getenv("KUBERNETES_TOKEN_SECRET"), "token")
	client = &http.Client{}
}

func buildK8sRequest(method string, url string, body io.Reader) (r *http.Request, e error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		fmt.Println(err)
		return req, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Authorization", "Bearer "+kubernetestoken)
	return req, nil
}

func DeleteKubeJob(space string, jobName string) (e error) {
	kubernetesapiserver := os.Getenv("KUBERNETES_API_SERVER")
	kubernetesapiversion := "/apis/batch/v1/"
	uri := "https://" + kubernetesapiserver + kubernetesapiversion + "namespaces/" + space + "/jobs/" + jobName

	resp, jerr := kubernetesAPICall("DELETE", uri)
	if jerr != nil {
		return jerr
	}
	if resp.Status != http.StatusOK {
		return errors.New(string(resp.Body))
	}

	perr := deletePods(space, jobName)
	if perr != nil {
		return perr
	}

	return nil
}

func Startpod(oneoff structs.OneOffSpec) string {
	var oneoffpod structs.OneOffPod
	oneoffpod.APIVersion = "v1"
	oneoffpod.Kind = "Pod"
	oneoffpod.Metadata.Namespace = oneoff.Space
	oneoffpod.Metadata.Name = oneoff.Podname
	oneoffpod.Metadata.Labels.Name = oneoff.Podname
	oneoffpod.Metadata.Labels.Space = oneoff.Space
	var cont structs.ContainerItem
	cont.Name = oneoff.Podname
	cont.Image = oneoff.Image
	cont.Env = oneoff.Env
	cont.ImagePullPolicy = "Always"
	if oneoff.Command != "" {
		// Separate command into command and arguments
		args, err := shellwords.Parse(oneoff.Command)
		if err != nil {
			fmt.Println(err)
			cont.Command = append(cont.Command, oneoff.Command)
		} else if len(args) < 2 { // Command is a single string
			cont.Command = append(cont.Command, oneoff.Command)
		} else { // Command is a series of strings. First is exec, next are arguments
			cont.Command = append(cont.Command, args[0])
			cont.Args = args[1:]
		}
	}
	var si structs.SecretItem
	si.Name = os.Getenv("KUBERNETES_IMAGE_PULL_SECRET")
	cont.ImagePullSecrets = append(cont.ImagePullSecrets, si)
	oneoffpod.Spec.Containers = append(oneoffpod.Spec.Containers, cont)
	oneoffpod.Spec.RestartPolicy = "Never"
	bodybytes, err := json.Marshal(oneoffpod)
	if err != nil {
		fmt.Println(err)
	}
	kubernetesapiserver := os.Getenv("KUBERNETES_API_SERVER")
	kubernetesapiversion := os.Getenv("KUBERNETES_API_VERSION")
	req, err := buildK8sRequest("POST", "https://"+kubernetesapiserver+"/api/"+kubernetesapiversion+"/namespaces/"+oneoff.Space+"/pods", bytes.NewBuffer(bodybytes))
	req.Header.Add("Content-type", "application/json")
	fmt.Printf("%+v\n", req)
	resp, doerr := client.Do(req)

	if doerr != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	bodybytes, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(bodybytes))
	return string(bodybytes)
}

func Deletepod(spacename string, pod string) string {
	kubernetesapiserver := os.Getenv("KUBERNETES_API_SERVER")
	kubernetesapiversion := os.Getenv("KUBERNETES_API_VERSION")
	req, err := buildK8sRequest("DELETE", "https://"+kubernetesapiserver+"/api/"+kubernetesapiversion+"/namespaces/"+spacename+"/pods/"+pod, nil)
	if err != nil {
		fmt.Println("Error creating request " + err.Error())
	}
	client := http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(bodybytes))
	return string(bodybytes)

}

func deletePods(space string, podName string) (e error) {
	kubernetesapiserver := os.Getenv("KUBERNETES_API_SERVER")
	kubernetesapiversion := "v1"
	uri := "https://" + kubernetesapiserver + "/api/" + kubernetesapiversion + "/namespaces/" + space + "/pods?labelSelector=name=" + podName

	_, err := kubernetesAPICall("DELETE", uri)
	return err
}

func ScaleJob(space string, jobName string, replicas int, timeout int) (e error) {
	if space == "" {
		return errors.New("FATAL ERROR: Unable to scale job, space is blank.")
	}
	if jobName == "" {
		return errors.New("FATAL ERROR: Unable to scale job, the jobName is blank.")
	}
	kubernetesapiserver := os.Getenv("KUBERNETES_API_SERVER")
	req, e := buildK8sRequest("GET", "https://"+kubernetesapiserver+"/apis/batch/v1/namespaces/"+space+"/jobs/"+jobName, nil)
	if e != nil {
		return e
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("Unable to get job on scale, kubernetes returned: " + resp.Status)
	}
	defer resp.Body.Close()
	var job JobScaleGet
	bodybytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bodybytes, &job)
	if err != nil {
		return err
	}
	job.Spec.Parallelism = replicas
	job.Spec.BackOffLimit = 0
	p, err := json.Marshal(job)
	if err != nil {
		return err
	}
	req, e = buildK8sRequest("PUT", "https://"+kubernetesapiserver+"/apis/batch/v1/namespaces/"+space+"/jobs/"+jobName, bytes.NewBuffer(p))
	if e != nil {
		return e
	}
	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("Unable to scale job, kubernetes returned: " + resp.Status)
	}
	return nil
}

func kubernetesAPICall(method string, uri string) (re Response, err error) {
	req, err := buildK8sRequest(method, uri, nil)
	if err != nil {
		return re, err
	}
	fmt.Println(req)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return re, err
	}
	defer resp.Body.Close()
	re.Body, _ = ioutil.ReadAll(resp.Body)
	re.Status = resp.StatusCode
	return re, nil
}

type JobScaleGet struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Metadata   struct {
		Name              string    `json:"name"`
		Namespace         string    `json:"namespace"`
		SelfLink          string    `json:"selfLink"`
		UID               string    `json:"uid"`
		ResourceVersion   string    `json:"resourceVersion"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
		Labels            struct {
			Name  string `json:"name"`
			Space string `json:"space"`
		} `json:"labels"`
	} `json:"metadata"`
	Spec struct {
		Parallelism  int `json:"parallelism"`
		Completions  int `json:"completions"`
		BackOffLimit int `json:"backOffLimit"`
		Selector     struct {
			MatchLabels struct {
				ControllerUID string `json:"controller-uid"`
			} `json:"matchLabels"`
		} `json:"selector"`
		Template struct {
			Metadata struct {
				Name              string      `json:"name"`
				Namespace         string      `json:"namespace"`
				CreationTimestamp interface{} `json:"creationTimestamp"`
				Labels            struct {
					ControllerUID string `json:"controller-uid"`
					JobName       string `json:"job-name"`
					Name          string `json:"name"`
					Space         string `json:"space"`
				} `json:"labels"`
			} `json:"metadata"`
			Spec struct {
				Containers []struct {
					Name  string `json:"name"`
					Image string `json:"image"`
					Env   []struct {
						Name  string `json:"name"`
						Value string `json:"value"`
					} `json:"env"`
					Resources struct {
					} `json:"resources"`
					TerminationMessagePath   string `json:"terminationMessagePath"`
					TerminationMessagePolicy string `json:"terminationMessagePolicy"`
					ImagePullPolicy          string `json:"imagePullPolicy"`
					SecurityContext          struct {
						Capabilities struct {
						} `json:"capabilities"`
					} `json:"securityContext"`
				} `json:"containers"`
				RestartPolicy                 string `json:"restartPolicy"`
				TerminationGracePeriodSeconds int    `json:"terminationGracePeriodSeconds"`
				DNSPolicy                     string `json:"dnsPolicy"`
				SecurityContext               struct {
				} `json:"securityContext"`
				ImagePullSecrets []struct {
					Name string `json:"name"`
				} `json:"imagePullSecrets"`
				SchedulerName string `json:"schedulerName"`
			} `json:"spec"`
		} `json:"template"`
	} `json:"spec"`
	Status struct {
		StartTime time.Time `json:"startTime"`
	} `json:"status"`
}

func GetTestLogs(jobspace string, job string, instance string) (l []string, e error) {
	limitBytes := "10000000"
	var lines []string
	kubernetesapiserver := os.Getenv("KUBERNETES_API_SERVER")
	req, e := buildK8sRequest("GET", "https://"+kubernetesapiserver+"/api/v1/namespaces/"+jobspace+"/pods/"+instance+"/log?limitBytes="+limitBytes+"&container="+job, nil)
	if e != nil {
		return lines, e
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return lines, err
	}
	if resp.StatusCode != http.StatusOK {
		return lines, errors.New("Unable to get logs, kubernetes returned: " + resp.Status)
	}
	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return lines, errors.New("Unable to get Logs")
	}
	lines = strings.Split(string(bodybytes), "\n")
	return lines, nil
}
