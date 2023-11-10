# Go HTTP server mock for testing
## Setup
Set environment variables
- `PORT` (optional) the socket port that the server will be listened on, `8080` is default
## Features
- Graceful shutdown test
  - In normal environment, set `GRACEFULLY_SHUTDOWN_TIMEOUT` (optional) The timeout in second, it's `0` by default
  - In kubernetes (k8s) environment, they don't respect the timeout, they just notify and SIGKILL after reaching `terminationGracePeriodSeconds` to all containers in the pod
  - Request with the `delay` query to delay the request in second, example: `curl 'localhost:8080/?delay=10'`