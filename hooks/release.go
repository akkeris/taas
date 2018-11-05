package hooks

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	alamo "taas/alamo"
	diagnostics "taas/diagnostics"
	githubapi "taas/githubapi"
	structs "taas/structs"
)

func ReleaseHook(releasehookpayload structs.ReleaseHookSpec, berr binding.Errors, r render.Render) {
	if berr != nil {
		fmt.Println("Bad Request")
		fmt.Println(berr)
		return
	}
	fmt.Println(releasehookpayload.App.Name)
	fmt.Println(releasehookpayload.Space.Name)
	fmt.Println(releasehookpayload.Action)
	fmt.Println(releasehookpayload.Release.Result)
	diagnosticslist, err := diagnostics.GetDiagnostics(releasehookpayload.Space.Name, releasehookpayload.App.Name, releasehookpayload.Action, releasehookpayload.Release.Result)
	if err != nil {
		fmt.Println(err)
	}
	for _, element := range diagnosticslist {
		element.BuildID = releasehookpayload.Build.ID
		version, err := alamo.GetVersion(element.App, element.Space, element.BuildID)
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
		org, err := alamo.GetAppControllerOrg(element.App + "-" + element.Space)
		element.Organization = org
		element.CommitAuthor = commitauthor
		element.CommitMessage = commitmessage
		spew.Dump(element)
		diagnostics.RunDiagnostic(element)
	}

}
