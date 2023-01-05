#!/bin/bash

set -e
set -x

aws lambda update-function-code --function-name base_metrics_pusher --zip-file fileb://./main.zip
