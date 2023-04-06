#!/bin/bash

if [[ $ID -eq 2 ]] ; then
    if ping -c1 db"$REPLICA_ID" ; then
        echo "Set db1 as master from db2"
        sleep 30s
        mysql --host="db$REPLICA_ID" -u root -ppassword -e "CREATE USER IF NOT EXISTS 'mydb_user_$REPLICA_ID'@'%' IDENTIFIED BY 'mydb_pwd_$REPLICA_ID';GRANT REPLICATION SLAVE ON *.* TO 'mydb_user_$REPLICA_ID'@'%'; FLUSH PRIVILEGES;"
        mysql --host="db$REPLICA_ID" -u root -ppassword -e "RESET MASTER;FLUSH TABLES WITH READ LOCK;SHOW MASTER STATUS;"
        mysqldump --host="db$REPLICA_ID" -u root -ppassword --all-databases --master-data=2 > /sqldump/dump.sql
        mysql --host="db$REPLICA_ID" -u root -ppassword -e "UNLOCK TABLES;"
        mysql -u root -ppassword -e "STOP SLAVE;"
        mysql -u root -ppassword < /sqldump/dump.sql
        CURRENT_LOG=$(mysql --host="db$REPLICA_ID" -u root -ppassword -e "show master status" -s | tail -n 1 | awk {'print $1'})
        CURRENT_POS=$(mysql --host="db$REPLICA_ID" -u root -ppassword -e "show master status" -s | tail -n 1 | awk {'print $2'})
        start_slave_stmt="RESET SLAVE;CHANGE MASTER TO MASTER_HOST='db$REPLICA_ID',MASTER_USER='mydb_user_$REPLICA_ID',MASTER_PASSWORD='mydb_pwd_$REPLICA_ID',MASTER_LOG_FILE='$CURRENT_LOG',MASTER_LOG_POS=$CURRENT_POS; START SLAVE;"
        echo "$start_slave_stmt"
        mysql -u root -ppassword -e "$start_slave_stmt"
    fi
else
    if ping -c1 db"$REPLICA_ID" ; then
      echo "Set db1 as master from db2"
      mysql -u root -ppassword -e "CREATE USER IF NOT EXISTS 'mydb_user_$ID'@'%' IDENTIFIED BY 'mydb_pwd_$ID';GRANT REPLICATION SLAVE ON *.* TO 'mydb_user_$ID'@'%'; FLUSH PRIVILEGES;"
      mysql -u root -ppassword -e "RESET MASTER;FLUSH TABLES WITH READ LOCK;SHOW MASTER STATUS;"
      mysqldump --host="db$REPLICA_ID" -u root -ppassword --all-databases --master-data=2 > /sqldump/dump.sql
      mysql --host="db$REPLICA_ID" -u root -ppassword -e "UNLOCK TABLES;"
      mysql -u root -ppassword -e "STOP SLAVE;"
      mysql -u root -ppassword < /sqldump/dump.sql
      CURRENT_LOG=$(mysql -u root -ppassword -e "show master status" -s | tail -n 1 | awk {'print $1'})
      CURRENT_POS=$(mysql -u root -ppassword -e "show master status" -s | tail -n 1 | awk {'print $2'})
      start_slave_stmt="RESET SLAVE;CHANGE MASTER TO MASTER_HOST='db$ID',MASTER_USER='mydb_user_$ID',MASTER_PASSWORD='mydb_pwd_$ID',MASTER_LOG_FILE='$CURRENT_LOG',MASTER_LOG_POS=$CURRENT_POS; START SLAVE;"
      echo "$start_slave_stmt"
      mysql --host="db$REPLICA_ID" -u root -ppassword -e "$start_slave_stmt"
    else
      echo "Importing questions to db1"
      mysql -u root -ppassword < /questions.sql
    fi
fi