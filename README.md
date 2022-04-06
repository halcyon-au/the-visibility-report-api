# the-visibility-report-api

## Running

docker compose up

### Windows

Note, if editing `init.sh` ensure that you are using Linux line endings (LF) 

## Deploying

Change mode to development/prod
On github PR it deploys to github registry

## Routes

GET api/v1/hb -> return a heartbeat of the system
GET api/v1/countries/rankings -> return a ranked listed of countries