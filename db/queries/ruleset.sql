
-- name: GetApp :one
SELECT app
FROM ruleset
WHERE slice =$1 AND app = $2 AND class = $3 AND brwf = 'W';

-- name: GetClass :one
SELECT class 
FROM ruleset
WHERE slice = $1 AND app =$2  AND class = $3 AND brwf ='W';

-- name: GetWFActiveStatus :one
SELECT is_active 
FROM ruleset
WHERE slice = $1 AND app =$2  AND class = $3 AND brwf ='W' AND setname = $4;

-- name: GetWFInternalStatus :one
SELECT is_internal
FROM ruleset
WHERE slice = $1 AND app =$2  AND class = $3 AND brwf ='W' AND setname = $4;

--


