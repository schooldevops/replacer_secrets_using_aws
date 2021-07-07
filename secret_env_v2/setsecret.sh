#!/bin/bash

export DB_PASSWD="`./secret_env -s myproject/schooldevops/db -p schooldevops -k password`"
export DB_USERNAME="`./secret_env -s myproject/schooldevops/db -p schooldevops -k username`"
export USER_TOKEN="`./secret_env -s myproject/schooldevops/db -p schooldevops -k usertoken`"

echo $DB_PASSWD
echo $DB_USERNAME
echo $USER_TOKEN
