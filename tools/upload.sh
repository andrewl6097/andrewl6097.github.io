#!/bin/sh

set -e
set -x

rbenv exec bundle exec jekyll b
cp -av img/ _site/assets/img/

aws s3 sync _site s3://run-parallel.sh/
aws cloudfront create-invalidation --distribution-id EO0JOSZAC367N --paths /index.html /feed.xml /categories/index.html /posts/index.html /tags/index.html /archives/index.html
