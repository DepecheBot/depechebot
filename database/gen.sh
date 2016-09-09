#!/bin/bash

DBNAME=chats.sqlite3

EXTRA=$1

SRC=$(realpath $(cd -P "$( dirname "${BASH_SOURCE[0]}" )" && pwd ))

DB=file:$SRC/$DBNAME

XOBIN=$(which xo)
if [ -e $SRC/../../xo ]; then
  XOBIN=$SRC/../../xo
fi

DEST=$SRC/models

set -x

mkdir -p $DEST
rm -f $DEST/*.go
rm -f $SRC/sqlite3
rm -f $SRC/$DBNAME

sqlite3 $DB << 'ENDSQL'
PRAGMA foreign_keys = OFF;
CREATE TABLE chat (
  primary_id INTEGER NOT NULL PRIMARY KEY,
  chat_id BIGINT UNIQUE NOT NULL,
  user_id INTEGER NOT NULL,
  user_name TEXT NOT NULL DEFAULT '',
  real_name TEXT NOT NULL,
  first_name TEXT NOT NULL,
  last_name TEXT NOT NULL,
  open_time timestamptz NOT NULL,
  last_time DATETIME NOT NULL,
  groups TEXT NOT NULL,
  state TEXT NOT NULL
);
ENDSQL

$XOBIN $DB -o $SRC/models $EXTRA

#pushd $SRC &> /dev/null
#
#go build
#./sqlite3 $EXTRA
#
#popd &> /dev/null
