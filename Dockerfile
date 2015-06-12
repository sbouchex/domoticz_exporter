FROM       golang:onbuild
MAINTAINER Seb <sbouchex@gmail.com>

ENTRYPOINT [ "go-wrapper", "run" ]
EXPOSE     9103
