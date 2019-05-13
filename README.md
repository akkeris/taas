## **Configure Akkeris TaaS**

**_Sample Test framework: Metis_**

*Note: Tests are registered per app per environment/space. Metis is Greek God of Quality.*
  

Install or update Akkeris

-   	Installation instructions: [https://docs.bigsquid.io/getting-started/prerequisites-and-installing.html](https://docs.bigsquid.io/getting-started/prerequisites-and-installing.html)

-   If you have Akkeris installed already, update it

-   `aka update`

  <!-- ToDo: Information regarding setting up a TaaS app from which to run tests -->

Install TaaS plugin

-   There is no UI for TaaS currently.  From command line type `aka`.  TaaS commands are at the bottom of the list if you have it installed.

-   If not installed, type `aka plugins:install TaaS`
-  	 Add `export DIAGNOSTICS_API_URL=<akkeris system TaaS URL>` to your bashrc or zshrc file.
  

Structure of Metis (your test framework/harness)

- TaaS is set up to trigger tests located in APP_PATH. You can set a home directory and further specify $APP_PATH from there or run all tests in $HOME, managing which tests do and don't run through environment variables or a combination of env vars and $APP_PATH.


Gather Info

-   Verify pipeline exists and note the name.

-   Create one if none exists.

-   	Creating pipelines: [https://docs.bigsquid.io/how-akkeris-works.html#defining-pipelines-and-environments](https://docs.bigsquid.io/how-akkeris-works.html#defining-pipelines-and-environments) if you haven’t done that before.

-   Verify pipeline at [https://akkeris.bigsquid.io/pipelines](https://akkeris.bigsquid.io/pipelines) or

	-   `aka apps:info –a <app_name>`

	-   Pipeline name will be listed if it exist
	-   Space_name = everything after the first dash in the app_name

  

Register tests

-   Use Akkeris TaaS plugin to register tests, `aka TaaS:tests:register`

-   Enter values requested by plugin as indicated below

	-   App Name = <app_name>, e.g. myapp-myspace-env
	-   Job Name = <app_name>
	-   Job Space = TaaS
	-   Image = < path_to_your_image >
	-   Pipeline Name = `pipelineName` or “manual” if manual promotion is desired
	-   Transition from = <spaceName_promoting_from:appName_promoting_from>, e.g. “development:myapp-myspace-staging”
	-   Transition to = <stage_promoting_to:appName_promoting_to>, e.g. “staging:myapp-myspace-prod”
	-   Timeout = how long before test run should timeout in seconds
	-   Start delay = how many seconds before running tests (gauge based on how long app takes to load).
	- 	Slack Channel (no leading #) = Slack channel where test results should be sent.
	-   Environment variables = Optional.  Format is KEY=value.

  

-   Enter the rest of the env vars Metis needs to run the tests

-   `aka TaaS:config:set KEY=value` 
-   For bulk loading of env vars you can use the following script from within the Metis root directory. Make sure your env files are up to date.

-   `cat ./path/to/.env.file | grep = | awk '{print "aka TaaS:config:set <name-of-test> "$0}' | sed -e 's/ = /=/g' > temp.sh` 

-   `aka TaaS:tests:info` to see all tests and find their names

-   `chmod +x ./temp.sh`
-   `./temp.sh`

-   Check your work.
	
	-   `aka TaaS:tests:info <name_of_test>` to see details of registered test
	-   `aka TaaS:tests:trigger <name_of_test>` to trigger a test run.
	-   Check `your_designated_slack_channel` for results

List of available TaaS commands
```aka TaaS:tests                              list tests
  aka TaaS:images                             list images
  aka TaaS:tests:info ID                      describe test
  aka TaaS:tests:register                     register test
  aka TaaS:tests:update ID                    update test
  aka TaaS:tests:destroy ID                   delete test
  aka TaaS:tests:trigger ID                   trigger a test
  aka TaaS:tests:runs ID                      list test runs
  aka TaaS:config ID                          list environment variables
  aka TaaS:config:set ID KVPAIR               set an environment variable
  aka TaaS:config:unset ID VAR                unset and environment variable
  aka TaaS:secret:create ID                   adds a secret to a test
  aka TaaS:secret:remove ID                   removed a secret from a test
  aka TaaS:hooks:create                       add testing hooks to an app
  aka TaaS:runs:info ID                       get info for a run
  aka TaaS:runs:output ID                     get logs for a run. If ID is a test name, gets latest
  aka TaaS:logs ID                            get logs for a run. If ID is a test name, gets latest
  aka TaaS:runs:rerun ID                      reruns a run