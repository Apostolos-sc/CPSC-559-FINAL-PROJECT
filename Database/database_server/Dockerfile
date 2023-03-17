FROM mysql:5.7

#Reset entrypoint and cmd from base image
ENTRYPOINT []
CMD []

RUN yum update -y && yum install -y iputils

COPY initiate.sh .

ENTRYPOINT ["./initiate.sh"]

