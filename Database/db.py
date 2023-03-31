import mysql.connector
from mysql.connector import Error


def create_server_connection(host_name, user_name, user_password):
    connection = None
    try:
        connection = mysql.connector.connect(
            host=host_name,
            user=user_name,
            password=user_password
        )
        print("Successfully connected to DB")
    except Error as err:
        print(f"Error:'{err}'")
    return connection


connection = create_server_connection('localhost', 'root', 'password')


# def create_database(connection, query):
#     cursor = connection.cursor()
#     try:
#         cursor.execute(query)
#         print('Database created successfully')
#     except Error as err:
#         print(f"Error '{err}'")
#
#
# create_db_query = 'CREATE DATABASE IF NOT EXISTS mydb'
# create_database(connection, create_db_query)

def create_db_connection(host_name, user_name, user_password, db_name):
    connection = None
    try:
        connection = mysql.connector.connect(
            host=host_name,
            user=user_name,
            password=user_password,
            database=db_name
        )
        print("Successfully connected to DB")
    except Error as err:
        print(f"Error:'{err}'")
    return connection


connection = create_db_connection('localhost', 'root', 'password', 'mydb')


def execute_query(connection, query):
    cursor = connection.cursor()
    try:
        cursor.execute(query)
        connection.commit()
        print("Query successful")
    except Error as err:
        print(f"Error: '{err}'")


table_name = 'questions'
create_questions_table = '''
CREATE TABLE IF NOT EXISTS {0} (
    question_id INT PRIMARY KEY,
    question VARCHAR(3000) character set utf8 NOT NULL,
    answer VARCHAR(1000) character set utf8 NOT NULL,
    option_1 VARCHAR(1000) character set utf8 NOT NULL,
    option_2 VARCHAR(1000) character set utf8 NOT NULL,
    option_3 VARCHAR(1000) character set utf8 NOT NULL,
    option_4 VARCHAR(1000) character set utf8 NOT NULL
);'''.format(table_name)

execute_query(connection, create_questions_table)

files = open('history.txt', 'r+')
content = files.readlines()
question_bank = []

length = len(content)
i, counter = 0, 1
while i < length:
    if len(content[i].strip()) == 0:
        i += 1
        continue
    if content[i][0] == '#':
        if content[i][-2] not in ['?', '.']:
            while i < length and len(content[i].strip()) != 0:
                i += 1
        else:
            line = content[i:i + 7]
            if len(line) != 7:
                break
            if line[4][0] != 'C':
                i += 4
                continue
            question_bank.append((counter, line[0][3:-1], line[1][2:-1], line[2][2:-1],
                                  line[3][2:-1], line[4][2:-1], line[5][2:-1]))
            i += 6
            counter += 1
    i += 1

values = ', '.join(map(str, question_bank))
insert_questions = "INSERT INTO {} VALUES {}".format(table_name, values)
print(insert_questions)
execute_query(connection, insert_questions)
