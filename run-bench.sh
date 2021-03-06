#!/bin/bash
set -e
echo "Building benchmark Docker container"
docker build . -f Dockerfile.bench -t bverifybench > build.log
echo "Removing old container (if it exists)"
set +e
docker rm "bverifybench-$1" > /dev/null 2> /dev/null
echo "Starting benchmark '$1'"
docker run -p 6060:6060 -p 9100:9100 -it --name "bverifybench-$1" bverifybench bin/bench "-cpuprofile=cpu.pprof" "-memprofile=mem.pprof" "-$1"
echo "Cleaning up old output"
set -e
rm -rf ./tmp
echo "Copying output"
docker cp "bverifybench-$1":/app ./tmp
set -e
mkdir -p "./out"
cp ./tmp/*.tex "./out"
cp ./tmp/*.raw "./out"
cp ./tmp/*.pprof "./out"
rm -rf ./tmp
docker rm "bverifybench-$1" > /dev/null 2> /dev/null
echo "Done"
