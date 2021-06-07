#!/bin/bash

export DB_PASSWD="`./com.schooldevops.go.secret -s myproject/schooldevops/db -p schooldevops -k password`"
export DB_USERNAME="`./com.schooldevops.go.secret -s myproject/schooldevops/db -p schooldevops -k username`"
export USER_TOKEN="`./com.schooldevops.go.secret -s myproject/schooldevops/db -p schooldevops -k usertoken`"

echo $DB_PASSWD
echo $DB_USERNAME
echo $USER_TOKEN
