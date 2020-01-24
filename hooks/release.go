package hooks

import (
	"fmt"
	akkeris "taas/jobs"
	diagnostics "taas/diagnostics"
	githubapi "taas/githubapi"
	structs "taas/structs"
        "net/http"
	"github.com/davecgh/go-spew/spew"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
)


func ReleaseHook(req *http.Request, releasehookpayload structs.ReleaseHookSpec, berr binding.Errors, r render.Render) {
	if berr != nil {
		fmt.Println("Bad Request")
		fmt.Println(berr)
		return
	}
        ReleaseHookHandler(req, releasehookpayload, false)
}

func ReleaseHookHandler(req *http.Request, releasehookpayload structs.ReleaseHookSpec, isCron bool){

	fmt.Println(releasehookpayload.App.Name)
	fmt.Println(releasehookpayload.Space.Name)
	fmt.Println(releasehookpayload.Action)
	fmt.Println(releasehookpayload.Release.Result)
	diagnosticslist, err := diagnostics.GetDiagnostics(releasehookpayload.Space.Name, releasehookpayload.App.Name, releasehookpayload.Action, releasehookpayload.Release.Result)
	if err != nil {
		fmt.Println(err)
	}
	for _, element := range diagnosticslist {
                element.ReleaseID = releasehookpayload.Release.ID
		element.BuildID = releasehookpayload.Build.ID
                element.Token = req.Header.Get("x-akkeris-token")
		version, err := akkeris.GetVersion(element.App, element.Space, element.BuildID)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(version)
		var commitauthor string
		var commitmessage string
		if version != "" {
			element.GithubVersion = version
			commitauthor, commitmessage, err = githubapi.GetCommitAuthor(version)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(commitauthor)
		} else {
			commitauthor = "none"
			commitmessage = "none"
		}
		org, err := akkeris.GetAppControllerOrg(element.App + "-" + element.Space)
		element.Organization = org
		element.CommitAuthor = commitauthor
		element.CommitMessage = commitmessage
		spew.Dump(element)
		diagnostics.RunDiagnostic(element, isCron,structs.Cronjob{})
	}

}
