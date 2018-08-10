package jobs

import (
    "bytes"
    "errors"
    "encoding/json"
    "crypto/tls"
    "crypto/x509"
    "fmt"
    "github.com/bitly/go-simplejson"
    "net/http"
    "os"
    "io"
    "io/ioutil"
    "time"
)

var Client *http.Client

type Response struct {
        Status int
        Body   []byte
}

func Startclient(){
    vaulttoken := os.Getenv("VAULT_TOKEN")
    vaultaddr := os.Getenv("VAULT_ADDR")

    kubernetescertsecret := os.Getenv("KUBERNETES_CERT_SECRET")
    vaultaddruri := vaultaddr + "/v1/" + kubernetescertsecret
    vreq, err := http.NewRequest("GET", vaultaddruri, nil)
    vreq.Header.Add("X-Vault-Token", vaulttoken)
    vclient := &http.Client{}
    vresp, err := vclient.Do(vreq)
    if err != nil {
    }
    defer vresp.Body.Close()
    bodyj, _ := simplejson.NewFromReader(vresp.Body)
    admincrt, _ := bodyj.Get("data").Get("admin-crt").String()
    adminkey, _ := bodyj.Get("data").Get("admin-key").String()
    cacrt, _ := bodyj.Get("data").Get("ca-crt").String()

    cert, err := tls.X509KeyPair([]byte(admincrt), []byte(adminkey))
    if err != nil {
        fmt.Println(err)
    }

    caCert := cacrt
    if err != nil {
        fmt.Println(err)
    }
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM([]byte(caCert))

    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      caCertPool,
    }
    tlsConfig.BuildNameToCertificate()
    transport := &http.Transport{TLSClientConfig: tlsConfig}

    Client = &http.Client{Transport: transport}
}

func buildK8sRequest(method string, url string, body io.Reader)  (r *http.Request, e error){

     var err error
     var req *http.Request
      fmt.Println("using cert")
      req, err = http.NewRequest(method, url, body)
      if err != nil {
         fmt.Println(err)
         return req, err
      }
     req.Header.Add("Accept","application/json")
     req.Header.Add("Content-type", "application/json")
     return req, nil
}

func DeleteKubeJob(space string, jobName string) (e error) {
        kubernetesapiserver := os.Getenv("KUBERNETES_API_SERVER")
        kubernetesapiversion:= "/apis/batch/v1/"
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
        req, e := buildK8sRequest("GET", "https://" + kubernetesapiserver +"/apis/batch/v1/namespaces/"+space+"/jobs/"+jobName, nil)
        if e != nil {
                return e
        }
        resp, err := Client.Do(req);
        if err != nil {
              return err
        }
        if resp.StatusCode != http.StatusOK {
                return errors.New("Unable to get job on scale, kubernetes returned: " + resp.Status)
        }
        defer resp.Body.Close()
        var job JobScaleGet
        bodybytes , err := ioutil.ReadAll(resp.Body)
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
        req, e = http.NewRequest("PUT", "https://" + kubernetesapiserver +"/apis/batch/v1/namespaces/"+space+"/jobs/"+jobName,  bytes.NewBuffer(p))
        if e != nil {
                return e
        }
        resp,err = Client.Do(req)
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
        resp, err := Client.Do(req)
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
                Parallelism int `json:"parallelism"`
                Completions int `json:"completions"`
                BackOffLimit int `json:"backOffLimit"`
                Selector    struct {
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
