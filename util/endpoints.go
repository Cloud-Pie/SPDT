package util

//Services endpoints
const URL_SERIE_PROCESSING = "http://localhost:5000/api/peaks"
const ENDPOINT_SERIE_PROCESSING = "/api/peaks"

const ENDPOINT_FORECAST = "/api/forecast"

const ENDPOINT_VMS_PROFILES = "/api/vms"
const ENDPOINT_STATES = "/api/states"
const ENDPOINT_INVALIDATE_STATES = "/api/invalidate/" //  :timestamp
const ENDPOINT_CURRENT_STATE = "/api/current"

const ENDPOINT_SERVICE_PROFILES = "/getRegressionTRNsMongoDBAll/{apptype}/{appname}/"
const ENDPOINT_VM_TIMES = "/getPerVMTypeOneBootShutDownData"
const ENDPOINT_ALL_VM_TIMES = "/getPerVMTypeAllBootShutDownData"

const ENDPOINT_SERVICE_UPDATE_PROFILE = "/getPredictedRegressionReplicas/{apptype}/{appname}/{msc}/{numcoresutil}/{numcoreslimit}/{nummemlimit}"
