FROM mysql:5.7

RUN yum update -y && yum install -y iputils
RUN chown mysql:mysql /var/run/mysqld

COPY initiate.sh ./docker-entrypoint-initdb.d/initiate.sh
