#! /bin/bash
echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
docker build -t forfuncsake/garagedoor:${TRAVIS_BRANCH} .
docker tag forfuncsake/garagedoor:${TRAVIS_BRANCH} forfuncsake/garagedoor:latest
docker push forfuncsake/garagedoor:${TRAVIS_BRANCH}
docker push forfuncsake/garagedoor:latest
