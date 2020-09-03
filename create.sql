do $$
BEGIN

CREATE TABLE IF NOT EXISTS actions (
    id text NOT NULL,
    action text
);

CREATE TABLE IF NOT EXISTS diagnostics (
    id text PRIMARY KEY,
    space text,
    app text,
    action text,
    result text,
    job text,
    jobspace text,
    image text,
    pipelinename text,
    transitionfrom text,
    transitionto text,
    timeout integer,
    startdelay integer,
    slackchannel text,
    command text,
    testpreviews boolean,
    ispreview boolean,
    webhookurls text,
    CONSTRAINT diagnostics_space_app_result_action_job_jobspace_key UNIQUE (space, app, result, action, job, jobspace)
);
CREATE UNIQUE INDEX IF NOT EXISTS diagnostics_pkey ON diagnostics(id text_ops);
ALTER TABLE diagnostics ADD COLUMN IF NOT EXISTS command TEXT;
ALTER TABLE diagnostics ADD COLUMN IF NOT EXISTS testpreviews BOOLEAN;
ALTER TABLE diagnostics ADD COLUMN IF NOT EXISTS ispreview BOOLEAN;
ALTER TABLE diagnostics ADD COLUMN IF NOT EXISTS webhookurls TEXT;

CREATE TABLE IF NOT EXISTS promotions (
    id text NOT NULL,
    pipelinename text,
    transitionfrom text,
    transitionto text
);

CREATE TABLE IF NOT EXISTS rspecexamples (
    runid text,
    id text,
    description text,
    fulldescription text,
    status text,
    filepath text,
    linenumber integer,
    runtime double precision,
    pendingmessage text
);

CREATE TABLE IF NOT EXISTS rspecsummary (
    runid text,
    version text,
    summaryline text,
    duration double precision,
    examplecount integer,
    failurecount integer,
    pendingcount integer,
    messages text
);

CREATE TABLE IF NOT EXISTS testcase (
    runid text,
    classname text,
    name text,
    file text,
    time text
);

CREATE TABLE IF NOT EXISTS testruns (
    testid text NOT NULL,
    runid text PRIMARY KEY,
    space text,
    app text,
    org text,
    buildid text,
    githubversion text,
    commitauthor text,
    commitmessage text,
    action text,
    result text,
    job text,
    jobspace text,
    image text,
    pipelinename text,
    transitionfrom text,
    transitionto text,
    timeout integer,
    startdelay integer,
    overallstatus text,
    CONSTRAINT testruns_testid_runid_key UNIQUE (testid, runid)
);

ALTER TABLE testruns add column if not exists releaseid text;
ALTER TABLE testruns add column if not exists run_on TIMESTAMP DEFAULT NOW();
CREATE UNIQUE INDEX IF NOT EXISTS testruns_pkey ON testruns(runid text_ops);
CREATE UNIQUE INDEX IF NOT EXISTS testruns_testid_runid_key ON testruns(testid text_ops,runid text_ops);

CREATE TABLE IF NOT EXISTS testsuite (
    runid text,
    name text,
    tests text,
    failures text,
    errors text,
    time text,
    timestamp text,
    hostname text
);

CREATE TABLE IF NOT EXISTS audits (
  auditid text NOT NULL,
  id text NOT NULL,
  audituser text NOT NULL,
  audittype text NOT NULL,
  auditkey text,
  newvalue text,
  created_at timestamp without time zone NOT NULL DEFAULT now(),
  CONSTRAINT pkey_audits PRIMARY KEY (auditid)
);

CREATE TABLE IF NOT EXISTS cronjobs(
  id text NOT NULL,
  job text,
  jobspace text,
  cronspec text,
  command text,
  disabled BOOLEAN DEFAULT false,
  CONSTRAINT cronjobs_pk PRIMARY KEY (id)
);
ALTER TABLE cronjobs ADD COLUMN IF NOT EXISTS disabled BOOLEAN;

CREATE TABLE IF NOT EXISTS cronruns(
  testid text,
  runid text,
  space text,
  app text,
  job text,
  jobspace text,
  image text,
  overallstatus text,
  starttime timestamp without time zone,
  endtime timestamp without time zone,
  cronid text
);

CREATE TABLE IF NOT EXISTS cronjobschedule(
  id text NOT NULL,
  next text,
  prev text
);

-- Function to listen to events (for cronjobs)
-- https://coussej.github.io/2015/09/15/Listening-to-generic-JSON-notifications-from-PostgreSQL-in-Go/
CREATE OR REPLACE FUNCTION notify_event() RETURNS TRIGGER AS $f$

    DECLARE 
        data json;
        notification json;

    BEGIN
    
        -- Convert the old or new row to JSON, based on the kind of action.
        IF (TG_OP = 'DELETE') THEN
            data = json_build_object('old_record', row_to_json(OLD));
        ELSIF (TG_OP = 'INSERT') THEN
            data = json_build_object('new_record', row_to_json(NEW));
        ELSE
            data = json_build_object('old_record', row_to_json(OLD), 'new_record', row_to_json(NEW));
        END IF;
        
        -- Contruct the notification as a JSON string.
        notification = json_build_object(
                          'table', TG_TABLE_NAME,
                          'action', TG_OP,
                          'data', data);
        
        -- Execute pg_notify(channel, notification)
        PERFORM pg_notify('events',notification::text);
        
        -- Result is ignored since this is an AFTER trigger
        RETURN NULL; 
    END;
    
$f$ LANGUAGE plpgsql;

-- Create trigger to send events if cronjob table changes
DROP TRIGGER IF EXISTS cronjobs_notify_event ON cronjobs;
CREATE TRIGGER cronjobs_notify_event
AFTER INSERT OR UPDATE OR DELETE ON cronjobs
    FOR EACH ROW EXECUTE PROCEDURE notify_event();

END
$$;
