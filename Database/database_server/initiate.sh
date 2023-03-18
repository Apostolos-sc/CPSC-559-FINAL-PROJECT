#!/bin/bash

create_slave_stmt="GRANT REPLICATION SLAVE ON *.* TO mydb_user; FLUSH PRIVILEGES;"
mysql -u root -ppassword -e "$create_slave_stmt"

if ping -c1 db"$REPLICA_ID" ; then
    echo "DB$REPLICA_ID is online. Setting db$ID as slave"
    start_slave_stmt="CHANGE MASTER TO MASTER_HOST='db$REPLICA_ID',MASTER_PORT=3306,MASTER_USER='mydb_user',MASTER_PASSWORD='mydb_pwd',MASTER_AUTO_POSITION=1; START SLAVE;"
    mysql -u root -ppassword -e "$start_slave_stmt"

else
    echo "DB$REPLICA_ID is offline. Creating slave user"
fi