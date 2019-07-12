## Table of Contents
- [Table of Contents](#Table-of-Contents)
- [Diagnostics](#Diagnostics)
  - [Create Diagnostic](#Create-Diagnostic)
  - [List Diagnostics](#List-Diagnostics)
  - [Get Diagnostic Info](#Get-Diagnostic-Info)
  - [Update Diagnostic Configuration](#Update-Diagnostic-Configuration)
  - [Delete Diagnostic](#Delete-Diagnostic)
  - [Set Configuration Variables](#Set-Configuration-Variables)
  - [Unset Configuration Variables](#Unset-Configuration-Variables)
  - [Create Hooks](#Create-Hooks)
  - [View Audits](#View-Audits)
  - [List Runs](#List-Runs)
  - [Get Run Info](#Get-Run-Info)
  - [Get Run Logs](#Get-Run-Logs)
  - [Get Array of Run Logs](#Get-Array-of-Run-Logs)
  - [Rerun Runs](#Rerun-Runs)
  - [Artifacts](#Artifacts)

## Diagnostics

A diagnostic is a test that can be run against Akkeris apps. There are only two requirements for a test framework to run as an Akkeris diagnostic - the framework must run in Docker, and it must exit with a zero (pass) or nonzero (fail). Logs and artifacts will be collected and provided to a Slack channel after the run is completed.

### Create Diagnostic

`POST /v1/diagnostic`

Creates a diagnostic. The diagnostic will be created in the given space and with the given name, and act upon the given app. Note that the job, jobspace, app, space, org, action, and result cannot be changed later. Use the prefix `akkeris://` to use the current image of an Akkeris app.

| Name           | Type             | Description                                                                   | Example                                            |
|----------------|------------------|-------------------------------------------------------------------------------|----------------------------------------------------|
| job            | required string  | The name of the diagnostic                                                    | ui-tests                                           |
| jobspace       | required string  | The name of the space where the diagnostic should live                        | taas                                               |
| app            | required string  | The name of the app that the diagnostic will target                           | appkitui                                           |
| space          | required string  | The name of the space where the targeted app lives                            | default                                            |
| action         | required string  | The app lifecycle action that the diagnostic will listen for                  | release                                            |
| result         | required string  | The result of the action that will trigger the diagnostic                     | succeeded                                          |
| image          | required string  | The Docker image to run                                                       | akkeris://appkitui-default                         |
| pipelinename   | required string  | The pipeline to promote the app in (set to manual for manual promotion)       | uipipeline                                         |
| transitionfrom | required string  | The pipeline stage to transition from (set to manual for manual promotion)    | dev                                                |
| transitionto   | required string  | The pipeline stage to transition to (set to manual for manual promotion)      | qa                                                 |
| timeout        | required integer | The amount of time in seconds before the diagnostic is marked as failed       | 180                                                |
| startdelay     | required integer | The amount of time in seconds that the diagnostic will wait before running    | 60                                                 |
| slackchannel   | required string  | The Slack channel to notify with test results                                 | taas_tests                                         |
| command   	   | optional string  | Override the Docker image command                                             | ./start.sh   		                                   |
| org          	 | optional string  | The name of the org for attribution (currently unused)                        | taas                                               |
| env            | optional array   | An array of name/value objects to add as environment variables                | [ { "name": "FOO", "value": "BAR" } ]              |

**CURL Example**

```bash
curl \
  -X POST \ 
  "http://localhost:4000/v1/diagnostic" \
  -d '{
    "job": "ui-tests",
    "jobspace": "taas",
    "app": "appkitui",
    "appspace": "default",
    "action": "release",
    "result": "succeeded",
    "image": "akkeris://appkitui-default",
    "pipelinename": "uipipeline",
    "transitionfrom": "dev",
    "transitionto": "qa",
    "timeout": 100,
    "startdelay": 10,
    "slackchannel": "taas_runs",
    "command": "./start.sh",
    "org": "akkeris",
    "env": [
      {
        "name": "FOO",
        "value": "BAR"
      }
    ]
  }'
```

**200 "OK" Response**

```json
{
  "status": "created"
}
```

### List Diagnostics

`GET /v1/diagnostics`

Retrieves a list of all of the currently configured diagnostics. 

An optional query parameter can be supplied to produce simplified output:

| Parameter Name     | Description                               | Example                   |
|--------------------|-------------------------------------------|---------------------------|
| simple             | Simplify the output for each diagnostic   | simple=true               |

**CURL Example**

```bash
curl \
  -X GET \
  "http://localhost:4000/v1/diagnostics?simple=true"
```

**200 "OK" Response**

```json
[
  {
    "id": "c9866ccd-2af0-458b-746e-48c954cc5774",
    "space": "default",
    "app": "appkitui",
    "org": "akkeris",
    "buildid": "",
    "version": "",
    "commitauthor": "",
    "commitmessage": "",
    "action": "release",
    "result": "succeeded",
    "job": "ui-tests",
    "jobspace": "taas",
    "image": "akkeris://appkitui-default",
    "pipelinename": "uipipeline",
    "transitionfrom": "dev",
    "transitionto": "qa",
    "timeout": 10,
    "startdelay": 10,
    "slackchannel": "taas_runs",
    "env": null,
    "runid": "5a5f8cc9-b931-4725-48e8-8ab52dd76d65",
    "overallstatus": "",
    "command": "./start.sh"
  }
]
```

### Get Diagnostic Info

`GET /v1/diagnostic/{diagnostic}`

Get info about a specific diagnostic

**CURL Example**

```bash
curl \
  -X GET \
  "http://localhost:4000/v1/diagnostic/ui-tests-taas"
```

**200 "OK" Response**

```json
{
  "id": "c9866ccd-2af0-458b-746e-48c954cc5774",
  "space": "default",
  "app": "appkitui",
  "org": "akkeris",
  "buildid": "",
  "version": "",
  "commitauthor": "",
  "commitmessage": "",
  "action": "release",
  "result": "succeeded",
  "job": "ui-tests",
  "jobspace": "taas",
  "image": "akkeris://appkitui-default",
  "pipelinename": "uipipeline",
  "transitionfrom": "dev",
  "transitionto": "qa",
  "timeout": 10,
  "startdelay": 10,
  "slackchannel": "taas_runs",
  "env": null,
  "runid": "5a5f8cc9-b931-4725-48e8-8ab52dd76d65",
  "overallstatus": "",
  "command": "./start.sh"
}
```

### Update Diagnostic Configuration

`PATCH /v1/diagnostic`

Update one or more properties of a specific diagnostic. Properties not included in the JSON body will be unset. Use the prefix `akkeris://` to use the current image of an Akkeris app.

| Name               | Type             | Description                                                                   | Example                                            |
|--------------------|------------------|-------------------------------------------------------------------------------|----------------------------------------------------|
| job                | required string  | The name of the job to update                                                 | ui-tests                                           |
| jobspace           | required string  | The name of the jobspace of the job to update                                 | taas                                               |
| image              | optional string  | The Docker image to run                                                       | akkeris://appkitui-default                         |
| pipelinename       | optional string  | The pipeline to promote the app in (set to manual for manual promotion)       | ui-pipeline                                        |
| transitionfrom     | optional string  | The pipeline stage to transition from (set to manual for manual promotion)    | dev                                                |
| transitionto       | optional string  | The pipeline stage to transition to (set to manual for manual promotion)      | qa                                                 |
| timeout            | optional integer | The amount of time in seconds before the diagnostic is marked as failed       | 180                                                |
| startdelay         | optional integer | The amount of time in seconds that the diagnostic will wait before running    | 60                                                 |
| slackchannel       | optional string  | The Slack channel to notify with test results                                 | taas_tests                                         |
| command   	       | optional string  | Override the Docker image command                                             | ./start.sh   		                                   |

**CURL Example**

```bash
curl \
  -X PATCH \ 
  "http://localhost:4000/v1/diagnostic" \
  -d '{
    "job": "ui-tests",
    "jobspace": "taas",
    "image": "hello-world",
    "timeout": 50
  }'
```

**200 "OK" Response**

```json
{
  "status": "updated"
}
```

### Delete Diagnostic

`DELETE /v1/diagnostic/{diagnostic}`

Delete a specific diagnostic

**CURL Example**

```bash
curl \
  -X DELETE \
  "http://localhost:4000/diagnostic/ui-tests-taas"
```

**200 "OK" Response**

```json
{
  "status": "deleted"
}
```

### Set Configuration Variables

`POST /v1/diagnostic/{diagnostic}/config`

Configuration variables are added to the diagnostic's Docker container as environment variables during a run.

| Name               | Type             | Description                                                                   | Example                                            |
|--------------------|------------------|-------------------------------------------------------------------------------|----------------------------------------------------|
| varname            | required string  | The name of the configuration variable                                        | FOO                                                |
| varvalue           | required string  | The value of the configuration variable                                       | BAR                                                |

**CURL Example**

```bash
curl \
  -X POST \
  "http://localhost:4000/v1/diagnostic/ui-tests-taas/config" \
  -d '{
    "varname": "FOO",
    "varvalue": "BAR"
  }'
```

**200 "OK" Response**
```json
{
  "response": "config variable set"
}
```

### Unset Configuration Variables

`DELETE /v1/diagnostic/{diagnostic}/config/{varname}`

Remove a configuration variable attached to a diagnostic.

**CURL Example**
```bash
curl \
  -X DELETE
  "http://localhost:4000/v1/diagnostic/ui-tests-taas/config/FOO"
```

**200 "OK" Response**
```json
{
  "response": "config variable unset"
}
```

### Create Hooks

`POST /v1/diagnostic/{diagnostic}/hooks`

Create TaaS build and release hooks on a diagnostic's target app. These are normally added during the diagnostic registration process.

**CURL Example**

```bash
curl \
  -X POST \
  "http://localhost:4000/v1/diagnostic/ui-tests-taas/hooks"
```

**200 "OK" Response**

```json
{
  "status": "hooks added"
}
```

### View Audits

`GET /v1/diagnostic/{diagnostic}/audits`

Get the audit log for a diagnostic, which includes various records of changes made to the diagnostic since it was created.

**CURL Example**

```bash
curl \
  -X GET \
  "http://localhost:4000/v1/diagnostic/ui-tests-taas/audits"
```

**200 "OK" Response**

```json
[
  {
    "auditid": "5a36b278-8c45-4c57-6ea2-dd730f173d4c",
    "id": "c9866ccd-2af0-458b-746e-48c954cc5774",
    "audituser": "Samuel.Beckett@octanner.com",
    "audittype": "configvarset",
    "auditkey": "test",
    "newvalue": "test",
    "created_at": "2019-07-11T10:52:34.897096Z"
  },
  {
    "auditid": "55cfa8bf-72bd-49a6-49ec-cab32af74991",
    "id": "c9866ccd-2af0-458b-746e-48c954cc5774",
    "audituser": "Samuel.Beckett@octanner.com",
    "audittype": "configvarunset",
    "auditkey": "test",
    "newvalue": "",
    "created_at": "2019-07-11T10:52:53.172998Z"
  }
]
```

### List Runs

`GET /v1/diagnostic/jobspace/{jobspace}/job/{job}/runs`

Get a list of the past runs of a diagnostic.

**CURL Example**

```bash
curl \
  -X GET \
  "http://localhost:4000/v1/diagnostic/jobspace/taas/job/ui-tests/runs
```

**200 "OK" Response**

```json
{
  "runs": [
    {
      "id": "2f689e9a-f8b0-4729-51f2-91eb8aae7b9c",
      "app": "appkitui",
      "space": "default",
      "job": "ui-tests",
      "jobspace": "taas",
      "hrtimestamp": "2019-07-11T17:33:44Z",
      "overallstatus": "success",
      "buildid": "cc35f230-e8be-4eec-bad0-80776cb6643a"
    }
  ]
}
```

### Get Run Info

`GET /v1/diagnostics/runs/info/{runid}`

Get detailed information on a specific diagnostic test run.

**CURL Example**

```bash
curl \
  -X GET \
  "http://localhost:4000/v1/diagnostics/runs/info/2f689e9a-f8b0-4729-51f2-91eb8aae7b9c"
```

**200 "OK" Response**

```json
{
  "_index": "logs",
  "_type": "run",
  "_id": "2f689e9a-f8b0-4729-51f2-91eb8aae7b9c",
  "_version": 1,
  "found": true,
  "_source": {
    "job": "ui-tests",
    "jobspace": "taas",
    "app": "appkitui",
    "space": "default",
    "testid": "ui-tests-taas-appkitui-default",
    "timestamp": 1562866424,
    "hrtimestamp": "2019-07-11T17:33:44Z",
    "buildid": "cc35f230-e8be-4eec-bad0-80776cb6643a",
    "logs": null
  }
}
```

### Get Run Logs

`GET /v1/diagnostic/logs/{runid}`

Get the output of a specific diagnostic test run in plaintext format.

**CURL Example**

```bash
curl \
  -X GET \
  "http://localhost:4000/v1/diagnostic/logs/2f689e9a-f8b0-4729-51f2-91eb8aae7b9c"
```

**200 "OK" Response**

```
[2019-07-11 11:33:43 AM]
[2019-07-11 11:33:43 AM]  Hello from Docker!
[2019-07-11 11:33:43 AM]  This message shows that your installation appears to be working correctly.
[2019-07-11 11:33:43 AM]
[2019-07-11 11:33:43 AM]  To generate this message, Docker took the following steps:
[2019-07-11 11:33:43 AM]   1. The Docker client contacted the Docker daemon.
[2019-07-11 11:33:43 AM]   2. The Docker daemon pulled the "hello-world" image from the Docker Hub.
[2019-07-11 11:33:43 AM]      (amd64)
[2019-07-11 11:33:43 AM]   3. The Docker daemon created a new container from that image which runs the
[2019-07-11 11:33:43 AM]      executable that produces the output you are currently reading.
[2019-07-11 11:33:43 AM]   4. The Docker daemon streamed that output to the Docker client, which sent it
[2019-07-11 11:33:43 AM]      to your terminal.
[2019-07-11 11:33:43 AM]
[2019-07-11 11:33:43 AM]  To try something more ambitious, you can run an Ubuntu container with:
[2019-07-11 11:33:43 AM]   $ docker run -it ubuntu bash
[2019-07-11 11:33:43 AM]
[2019-07-11 11:33:43 AM]  Share images, automate workflows, and more with a free Docker ID:
[2019-07-11 11:33:43 AM]   https://hub.docker.com/
[2019-07-11 11:33:43 AM]
[2019-07-11 11:33:43 AM]  For more examples and ideas, visit:
[2019-07-11 11:33:43 AM]   https://docs.docker.com/get-started/
[2019-07-11 11:33:43 AM]
```

### Get Array of Run Logs

`GET /v1/diagnostic/logs/{runid}/array`

Get the output of a specific diagnostic test run, line by line, in a JSON array.

**CURL Example**

```bash
curl \
  -X GET \
  "http://localhost:4000/v1/diagnostic/logs/2f689e9a-f8b0-4729-51f2-91eb8aae7b9c/array"
```

**200 "OK" Response**

```json
[
  "[2019-07-11 11:33:43 AM]  ",
  "[2019-07-11 11:33:43 AM]  Hello from Docker!",
  "[2019-07-11 11:33:43 AM]  This message shows that your installation appears to be working correctly.",
  "[2019-07-11 11:33:43 AM]  ",
  "[2019-07-11 11:33:43 AM]  To generate this message, Docker took the following steps:",
  "[2019-07-11 11:33:43 AM]   1. The Docker client contacted the Docker daemon.",
  "[2019-07-11 11:33:43 AM]   2. The Docker daemon pulled the \"hello-world\" image from the Docker Hub.",
  "[2019-07-11 11:33:43 AM]      (amd64)",
  "[2019-07-11 11:33:43 AM]   3. The Docker daemon created a new container from that image which runs the",
  "[2019-07-11 11:33:43 AM]      executable that produces the output you are currently reading.",
  "[2019-07-11 11:33:43 AM]   4. The Docker daemon streamed that output to the Docker client, which sent it",
  "[2019-07-11 11:33:43 AM]      to your terminal.",
  "[2019-07-11 11:33:43 AM]  ",
  "[2019-07-11 11:33:43 AM]  To try something more ambitious, you can run an Ubuntu container with:",
  "[2019-07-11 11:33:43 AM]   $ docker run -it ubuntu bash",
  "[2019-07-11 11:33:43 AM]  ",
  "[2019-07-11 11:33:43 AM]  Share images, automate workflows, and more with a free Docker ID:",
  "[2019-07-11 11:33:43 AM]   https://hub.docker.com/",
  "[2019-07-11 11:33:43 AM]  ",
  "[2019-07-11 11:33:43 AM]  For more examples and ideas, visit:",
  "[2019-07-11 11:33:43 AM]   https://docs.docker.com/get-started/",
  "[2019-07-11 11:33:43 AM]  "
]
```

### Rerun Runs

`GET /v1/diagnostic/rerun`

Redo a previous diagnostic test run. This simulates an {action} hook being recieved by TaaS, and will re-run each diagnostic configured to run with the given action and result. 

All of the following query parameters are required:

| Parameter Name     | Description                                       | Example                                           |
|--------------------|---------------------------------------------------|---------------------------------------------------|
| app                | The name of the app to re-run tests for           | app=appkitui                                     |
| space              | The name of the space where the app lives         | space=default                                    |
| action             | The action to use for the re-run                  | action=release                                   |
| result             | The action result to use for the re-run           | result=succeeded                                 |
| buildid            | The diagnostic build ID (found in the run info)   | buildid=cc35f230-e8be-4eec-bad0-80776cb6643a     |

**CURL Example**

```bash
curl \
  -X GET \
  "http://localhost:4000/v1/diagnostic/rerun?app=appkitui&space=default&action=release&result=succeeded&buildid=buildid=cc35f230-e8be-4eec-bad0-80776cb6643a"
```

**200 "OK" Response**

```json
{
  "status": "rerunning"
}
```

### Artifacts

`GET /v1/artifacts/{runid}/{directory}`

View artifacts uploaded to S3 as part of a test run. Result of this API call is HTML formatted and best viewed in a browser. All test runs will have `describe.txt` placed in the root directory - this file contains Kubernetes pod information gathered during the test run.

**CURL Example**

```bash
curl \
  -X GET \
  "http://localhost:4000/v1/artifacts/f2a5c469-581c-4887-699e-60b046b4e376/"
```

**200 "OK" Response**

```html
<!DOCTYPE html>
<html>
  <body>
    <font face = "courier">
      <ul>
        <li><a href="describe.txt">describe.txt</a> 2019-07-11 17:52:17 +0000 UTC</li>
      </ul>
    </font>
  </body>
</html>
```

