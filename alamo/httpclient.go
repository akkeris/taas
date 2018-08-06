package jobs

import (
    "errors"
    "crypto/tls"
    "crypto/x509"
    "fmt"
    "github.com/bitly/go-simplejson"
    "net/http"
    "os"
    "io"
    "io/ioutil"
)

var Client *http.Client

type Response struct {
        Status int
        Body   []byte
}

func Startclient(){
    vaulttoken := os.Getenv("VAULT_TOKEN")
    vaultaddr := os.Getenv("VAULT_ADDR")

    kubernetescertsecret := "secret/ops/alamo/ds1/certs"
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
        kubernetesapiserver := "alamo.ds1.octanner.io"
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
        kubernetesapiserver := "alamo.ds1.octanner.io"
        kubernetesapiversion := "v1"
        uri := "https://" + kubernetesapiserver + "/api/" + kubernetesapiversion + "/namespaces/" + space + "/pods?labelSelector=name=" + podName

        _, err := kubernetesAPICall("DELETE", uri)
        return err
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

