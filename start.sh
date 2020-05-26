kill -9 `lsof -i:1080 | grep -v PID | awk '{print $2}'`
kill -9 `lsof -i:3080 | grep -v PID | awk '{print $2}'`
git pull
go build
nohup ./httpMockServer 1080 &
nohup ./httpMockServer 3080 &
