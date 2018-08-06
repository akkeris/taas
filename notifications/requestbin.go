package notifications

import (
     "net/http"
     "io/ioutil"
     "os"
     structs "alamo-self-diagnostics/structs"
     "encoding/json"
     "fmt"
      "bytes"
)

func PostResults(result structs.ResultSpec) (e error) {

        postbackurl := os.Getenv("POSTBACKURL")
        p, err := json.Marshal(result)
        if err != nil {
                fmt.Println(err)
                return err
        }
        req, err := http.NewRequest("POST", postbackurl, bytes.NewBuffer(p))
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
        fmt.Println(bodybytes)
        return nil
}

