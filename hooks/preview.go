package hooks

import (
        "fmt"
        akkeris "taas/jobs"
        diagnostics "taas/diagnostics"
        githubapi "taas/githubapi"
        structs "taas/structs"

        "github.com/davecgh/go-spew/spew"
        "github.com/martini-contrib/binding"
        "github.com/martini-contrib/render"
)

func PreviewReleasedHook(previewreleasedhookpayload structs.PreviewReleasedHookSpec, berr binding.Errors, r render.Render) {
        if berr != nil {
                fmt.Println("Bad Request")
                fmt.Println(berr)
                return
        }
       fmt.Printf("%+v\n", previewreleasedhookpayload)
       diagnosticslist, err := diagnostics.GetDiagnostics(previewreleasedhookpayload.Space.Name, previewreleasedhookpayload.App.Name, "preview", "succeeded")
        if err != nil {
                fmt.Println(err)
        }
        for _, element := range diagnosticslist {
                var commitauthor string
                var commitmessage string
                commitauthor, commitmessage, err = githubapi.GetCommitAuthor(previewreleasedhookpayload.Slug.SourceBlob.Commit)
                if err != nil {
                     fmt.Println(err)
                }
                fmt.Println(commitauthor)
                fmt.Println(commitmessage) 
                org, err := akkeris.GetAppControllerOrg(element.App + "-" + element.Space)
                if err != nil {
                    fmt.Println(err)
                }
                element.Organization = org
                element.CommitAuthor = commitauthor
                element.CommitMessage = commitmessage
                spew.Dump(element)
                diagnostics.RunDiagnostic(element, true)
        }
       

}

func PreviewCreatedHook(previewcreatedhookpayload structs.PreviewCreatedHookSpec, berr binding.Errors, r render.Render) {
        if berr != nil {
                fmt.Println("Bad Request")
                fmt.Println(berr)
                return
        }
        fmt.Printf("%+v\n", previewcreatedhookpayload)
        fmt.Println(previewcreatedhookpayload.App.Name+"-"+previewcreatedhookpayload.Space.Name)
        fmt.Println(previewcreatedhookpayload.Preview.App.Name+"-"+previewcreatedhookpayload.Space.Name)
        diagnostic, err := diagnostics.GetDiagnosticByApp(previewcreatedhookpayload.App.Name, previewcreatedhookpayload.Space.Name)
        if err != nil {
           fmt.Println(err)
        }

        diagnostic.App=previewcreatedhookpayload.Preview.App.Name
        diagnostic.Space=previewcreatedhookpayload.Space.Name
        diagnostic.Action = "preview"
        diagnostic.PipelineName="manual"
        diagnostic.TransitionFrom = "manual"
        diagnostic.TransitionTo = "manual"
        fmt.Printf("%+v\n", diagnostic)
        err = akkeris.CreateService(diagnostic)
        if err != nil {
          fmt.Println(err)
        }
        r.Text(200,"")
}

