#!/bin/bash

/bin/bash docker-entrypoint.sh nohup mysqld

export MYSQL_PWD=password
mysql -u root -e "CREATE USER 'slaveuser'@'%' IDENTIFIED BY 'password';GRANT REPLICATION SLAVE ON *.* TO 'slaveuser'@'%' IDENTIFIED BY 'password';"

if ping -c1 db"$REPLICA_ID" ; then
    echo "DB$REPLICA_ID is online. Setting db$ID as slave"
    mysql -u root -e "CHANGE MASTER TO MASTER_HOST='db$REPLICA_ID',MASTER_PORT=3306,MASTER_USER='root',MASTER_PASSWORD='password',MASTER_AUTO_POSITION=1;START SLAVE;"

else
    echo "DB$REPLICA_ID is offline"
fi