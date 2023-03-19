#!/bin/bash

create_slave_stmt="GRANT REPLICATION SLAVE ON *.* TO mydb_user_$ID; FLUSH PRIVILEGES;"
echo "$create_slave_stmt"
mysql -u root -ppassword -e "$create_slave_stmt"

if ping -c1 db"$REPLICA_ID" ; then
    echo "DB$REPLICA_ID is online. Setting db$ID as slave"
    start_slave_stmt="RESET SLAVE; CHANGE MASTER TO MASTER_HOST='db$REPLICA_ID',MASTER_USER='mydb_user_$REPLICA_ID',MASTER_PASSWORD='mydb_pwd_$REPLICA_ID',MASTER_AUTO_POSITION=1; START SLAVE;"
    echo "$start_slave_stmt"
    mysql -u root -ppassword -e "$start_slave_stmt"

else
    echo "DB$REPLICA_ID is offline. Creating slave user"
fi