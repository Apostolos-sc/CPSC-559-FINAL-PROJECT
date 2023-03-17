FROM python:2.7

# install compulsory tools
RUN apt-get update
RUN apt-get install -y wget
RUN pip install mysql-connector
RUN wget https://github.com/mysql/mysql-utilities/archive/release-1.6.5.tar.gz
RUN tar xf release-1.6.5.tar.gz
RUN cd mysql-utilities-release-1.6.5 \
    && python ./setup.py build \
    && python ./setup.py install
#ENTRYPOINT mysqlfailover \
#           --force \
#           --master=mydb_user:mydb_pwd@mysql_master:3306 \
#           --discover-slaves-login=mydb_slave_user:mydb_slave_pwd \
#           auto
ENTRYPOINT sleep 20000