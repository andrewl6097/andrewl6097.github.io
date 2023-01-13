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

cat > get_all_clients.go <<END
package main

func get_all_clients() []string {
    ret := make([]string, 0)
    ret = append(ret, "unknown")
END

IFS=$'\n'

for i in `grep "// UA" ../metrics_pusher/main.go|cut -d'"' -f 2|sed -e 's/^/"/'|sed -e 's/$/"/'|sort|uniq`; do
    cat >> get_all_clients.go <<END
    ret = append(ret, $i)
END
done

cat >> get_all_clients.go <<END
    return ret
}
END

GOOS=linux GOARCH=amd64 go build -o main main.go get_all_files.go get_all_clients.go

zip main.zip main
