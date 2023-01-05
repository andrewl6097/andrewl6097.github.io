#!/bin/bash

set -e
set -x

cat > get_all_files.go <<END
package main

func get_all_files() []string {
    ret := make([]string, 0)
    ret = append(ret, "path:")
END

for i in `find ../_site/posts|sed -e "s#../_site##"|grep -v index.html|grep -v "^/posts$"`; do
    cat >> get_all_files.go <<END
    ret = append(ret, "path:$i")
END
done

cat >> get_all_files.go <<END
    return ret
}
END

GOOS=linux GOARCH=amd64 go build -o main main.go get_all_files.go

zip main.zip main
