#!/bin/sh

set -e
set -x

rbenv exec bundle exec jekyll b
cp -av img/ _site/assets/img/

aws s3 sync _site s3://run-parallel.sh/

# Create an invalidation list of every post that's changed since the
# last-uploaded commit, plus anything in staging.
CUR_COMMIT=`git log | head -n 1 | cut -d' ' -f 2`
LAST_COMMIT=`cat tools/last-uploaded-commit`
POSTS=""

for post in `git diff $LAST_COMMIT|grep -- "^--- a/_posts/"|cut -d'/' -f 3`; do
    POSTS="$POSTS $post"
done

POSTS="$POSTS `git status | grep _posts|cut -d'/' -f 2 | xargs`"
POSTS=`echo $POSTS | xargs | tr " " "\n" | sort -u | xargs`

PATHS=""

for post in `echo $POSTS`; do
    TITLE=`echo $post | cut -d'.' -f 1 | cut -d'-' -f 4-`
    PATHS="$PATHS /posts/$TITLE/index.html"
done

aws cloudfront create-invalidation --distribution-id EO0JOSZAC367N --paths /index.html /feed.xml /categories/index.html /posts/index.html /tags/index.html /archives/index.html $PATHS

echo $CUR_COMMIT > tools/last-uploaded-commit
