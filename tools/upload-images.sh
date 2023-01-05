#!/bin/sh

set -e
set -x

cp -av img/ _site/assets/img/
aws s3 sync _site/assets/img s3://run-parallel.sh/assets/img
