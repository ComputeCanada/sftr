#!/bin/sh

fatal()
{
  echo "$@" >&2
  exit 128
}

failed()
{
  echo "Test $1 ($2) failed: $3" >&2
  exit 1
}

# determine OS
sftr=""
case `uname` in
  Darwin)
    sftr=dist/sftr_osx
    ;;
  Linux)
    sftr=dist/sftr_linux_x86-64
    ;;
  *)
    echo "Sorry, not supported"
    exit 1
    ;;
esac

testdir=$(mktemp -d) || fatal "Could not create test directory"

conffile=$(mktemp) || fatal "Could not create config file"
cat > $conffile <<EOF
---
resources:
  - paths: ['${testdir}/testfile']
    op: put
    from: 192.168.14.0/24
  - paths: ['${testdir}/testfile']
    op: get
    from: 192.168.16.233/32
  - paths: ['${testdir}/subdir/*']
    op: put
    from: 192.168.16.233/32
  - paths: ['${testdir}/testfile2']
    op: put
    from: 192.168.16.234/32
    command: ['echo', 'hello']
  - paths: ['${testdir}/testfile3']
    op: put
    from: 192.168.16.235/32
    script: |
      echo "hello"
      echo "How do you do?"
EOF

testfile=$(mktemp)
cat > $testfile <<EOF
This is a test
It is the best
Don't you dare rest
Or you'll be mess't
EOF

test=1
testdesc="Simple put"
echo "--------  $test: $testdesc"
cat $testfile | SSH_ORIGINAL_COMMAND="put ${testdir}/testfile" SSH_CONNECTION="192.168.14.3 X Y Z" $sftr --config $conffile
diff $testfile ${testdir}/testfile || failed $test "$testdesc" "Placed and expected files differ"

test=2
testdesc="Simple get"
echo "--------  $test: $testdesc"
tmpfile=$(mktemp)
SSH_ORIGINAL_COMMAND="get ${testdir}/testfile" SSH_CONNECTION="192.168.16.233" $sftr --config $conffile > $tmpfile
diff $testfile $tmpfile || failed $test "$testdesc" "Expected and retrieved files differ"

test=3
testdesc="Directory put"
echo "--------  $test: $testdesc"
mkdir ${testdir}/subdir || fatal "Could not create test subdirectory"
cat $testfile \
	| SSH_ORIGINAL_COMMAND="put ${testdir}/subdir/otherfile" SSH_CONNECTION="192.168.16.233" $sftr --config $conffile \
	|| failed $test "$testdesc" "Could not find wildcard match"

test=4
testdesc="File put with post-transfer command"
echo "--------  $test: $testdesc"
cat $testfile \
	| SSH_ORIGINAL_COMMAND="put ${testdir}/testfile2" SSH_CONNECTION="192.168.16.234" $sftr --config $conffile 2>&1 \
	| grep 'Command: hello' || failed $test "$testdesc" "Did not see expected output"

test=5
testdesc="File put with post-transfer script"
echo "--------  $test: $testdesc"
cat $testfile \
	| SSH_ORIGINAL_COMMAND="put ${testdir}/testfile3" SSH_CONNECTION="192.168.16.235" $sftr --config $conffile 2>&1 \
	| grep 'How do you do?' || failed $test "$testdesc" "Did not see expected output"
