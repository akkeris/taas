# Akkeris Testing As A Service

Akkeris TaaS gives developers a place to host and run automated testing frameworks. TaaS is a big step towards making Akkeris a fully featured CI/CD offering - with TaaS, you can commit code, run automated tests, and promote your app along a development pipeline in a single step. You can run almost any test framework of your choice, as long as it can run as a Docker image and exits with a zero (pass) or nonzero (fail).

## Why Use TaaS?

### Simplify the typical deployment process!

Say a developer has several apps in an Akkeris pipeline, and goes through the same deployment process each time new code is ready for production. When code is committed into the GitHub repository associated with the Development version of the app, Akkeris automatically builds and deploys a new version of the app in the Development stage of the pipeline. The developer then runs automated tests against the app. Once those tests pass, the developer promotes the app to Staging, and runs additional tests in that environment. Then, once those tests pass as well, the developer finally promotes the app to Production, and runs smoke tests there.

With TaaS, all a developer needs to do is the first step - push code to the GitHub repository of the app. The rest of the steps are performed for them automatically. TaaS diagnostics can be registered for each stage in the pipeline, and be configured to promote the application to another stage upon a successful test run. If at any point along the pipeline the diagnostic fails, the promotion stops, and the faulty code is prevented from negatively impacting production.

#### Example

**ferengi-pipeline**
| Development   | Staging       | Production     |
|---------------|---------------|----------------|
| ferengi-dev   | ferengi-stg   | ferengi-prd    |

**TaaS tests**
| Diagnostic          | Target App     | Pipeline
|---------------------|----------------|-------------------------------|
| ferengi-dev-taas    | ferengi-dev    | ferengi-pipeline: dev->stg    |
| ferengi-stg-taas    | ferengi-stg    | ferengi-pipeline: stg->prd    |
| ferengi-prd-taas    | ferengi-prd    | n/a                           |

*Successful Test Case*

1. Developer pushes code to the repository ferenginar/ferengi. Akkeris builds and deploys a new version of ferengi-dev.
2. TaaS diagnostic *ferengi-dev-taas* is run against ferengi-dev. Test is successful. TaaS promotes ferengi-dev to ferengi-stg.
3. TaaS diagnostic *ferengi-stg-taas* is run against ferengi-stg. Test is successful. TaaS promotes ferengi-stg to ferengi-prd.
4. TaaS diagnostic *ferengi-prd-taas* is run against ferengi-prd. Test is successful. No promotion necessary.

*Unsuccessful Test Case*

1. Developer pushes code to the repository ferenginar/ferengi. Akkeris builds and deploys a new version of ferengi-dev.
2. TaaS diagnostic *ferengi-dev-taas* is run against ferengi-dev. Test is successful. TaaS promotes ferengi-dev to ferengi-stg.
3. TaaS diagnostic *ferengi-stg-taas* is run against ferengi-stg. Test is unsuccessful. No promotion occurs. ferengi-prd is unaffected.

### Get detailed information on your tests!

TaaS provides developers with all logs output by their testing framework during the test run. It also provides an S3 bucket that developers can use to upload images or error log files for maximum transparency into the test run. For example, a UI test framework can take a screenshot of an error message that interfered with an automated test. Additionally, detailed information on the status of the Kubernetes pod during the test run is uploaded to the S3 bucket for debugging purposes.

These resources are all collected and sent to the Slack channel of your choosing once the test is complete.

## API

[API Reference](./docs/API-Reference.md)

## Tips

* TaaS provides each test run with an S3 bucket that can be used to store files during the run (artifacts) for later access. Use the following environment variables in your test framework to access the bucket:

| Environment Variable          | Description                                                                            |
|-------------------------------|----------------------------------------------------------------------------------------|
| TAAS_ARTIFACT_REGION          | Amazon AWS region of the S3 bucket                                                     |
| TAAS_ARTIFACT_BUCKET          | Name of the S3 bucket                                                                  |
| TAAS_AWS_ACCESS_KEY_ID        | Amazon AWS access key ID (for authentication)                                          |
| TAAS_AWS_SECRET_ACCESS_KEY    | Amazon AWS secret key (for authentication)                                             |
| TAAS_RUNID                    | Current run ID. Use this as a prefix for file uploads (root directory to upload to)    |

* A file called `describe.txt` is available as an artifact for every test run. It contains detailed information about the Kubernetes pod used to run the diagnostic - container status, time when image was successfully pulled, etc.


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

-   If not installed, type `aka plugins:install taas`
-  	 Add `export DIAGNOSTICS_API_URL=<akkeris system taas URL>` to your bashrc or zshrc file.
  

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

-   Use Akkeris TaaS plugin to register tests, `aka taas:tests:register`

-   Enter values requested by plugin as indicated below

	-   App Name = <app_name>, e.g. myapp-myspace-env
	-   Job Name = <app_name>
	-   Job Space = taas
	-   Image = < path_to_your_image >
	-   Pipeline Name = `pipelineName` or “manual” if manual promotion is desired
	-   Transition from = <spaceName_promoting_from:appName_promoting_from>, e.g. “development:myapp-myspace-staging”
	-   Transition to = <stage_promoting_to:appName_promoting_to>, e.g. “staging:myapp-myspace-prod”
	-   Timeout = how long before test run should timeout in seconds
	-   Start delay = how many seconds before running tests (gauge based on how long app takes to load).
	- 	Slack Channel (no leading #) = Slack channel where test results should be sent.
	-   Environment variables = Optional.  Format is KEY=value.

  

-   Enter the rest of the env vars Metis needs to run the tests

-   `aka taas:config:set KEY=value` 
-   For bulk loading of env vars you can use the following script from within the Metis root directory. Make sure your env files are up to date.

-   `cat ./path/to/.env.file | grep = | awk '{print "aka taas:config:set <name-of-test> "$0}' | sed -e 's/ = /=/g' > temp.sh` 

-   `aka taas:tests:info` to see all tests and find their names

-   `chmod +x ./temp.sh`
-   `./temp.sh`

-   Check your work.
	
	-   `aka taas:tests:info <name_of_test>` to see details of registered test
	-   `aka taas:tests:trigger <name_of_test>` to trigger a test run.
	-   Check `your_designated_slack_channel` for results

List of available TaaS commands
```aka taas:tests                              list tests
  aka taas:images                             list images
  aka taas:tests:info ID                      describe test
  aka taas:tests:register                     register test
  aka taas:tests:update ID                    update test
  aka taas:tests:destroy ID                   delete test
  aka taas:tests:trigger ID                   trigger a test
  aka taas:tests:runs ID                      list test runs
  aka taas:config ID                          list environment variables
  aka taas:config:set ID KVPAIR               set an environment variable
  aka taas:config:unset ID VAR                unset and environment variable
  aka taas:secret:create ID                   adds a secret to a test
  aka taas:secret:remove ID                   removed a secret from a test
  aka taas:hooks:create                       add testing hooks to an app
  aka taas:runs:info ID                       get info for a run
  aka taas:runs:output ID                     get logs for a run. If ID is a test name, gets latest
  aka taas:logs ID                            get logs for a run. If ID is a test name, gets latest
  aka taas:runs:rerun ID                      reruns a run