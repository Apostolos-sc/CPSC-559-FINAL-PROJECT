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

ENTRYPOINT ["mysqlfailover", "--force", "--master=root:password@db1:3306", "--discover-slaves-login=root:password", "auto"]