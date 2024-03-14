#!/bin/sh

usage(){
	echo "usage: ab_test.sh file.rul" 1>&2;
	exit 1
}

case $# in
1)
	RULEFILE=$1
	shift
	;;
*)
	usage
esac


export BAD=false
waitfile() {
	for i in `seq 1 5`; do	
		if test -S $1; then
			return
			break
		fi
	 	sleep 1
	done
	export BAD=true
}

filterdiffs() {
	#eval exception and runtime exception, different message
	grep -v 'divide by zero'|grep -v '^Stats:'
}


export TSTNAME=`basename $RULEFILE|sed 's/\.rul$//g'`

export ROOT=`git rev-parse --show-toplevel`

(cd $ROOT/rips; go build)
cd $ROOT

export DNAME=/tmp/$TSTNAME$$

rm -rf "$DNAME"rips
rm -rf "$DNAME"gen
mkdir -p "$DNAME"rips

echo  "$DNAME"rips  "$DNAME"gen
mkdir -p gen

(./rips/rips -s "$DNAME"rips/sock.rips  -r $ROOT/xrips/examples $ROOT/extern/examples/scripts  $ROOT/$RULEFILE | filterdiffs) 2>&1 > "$DNAME"rips/1 | filterdiffs > "$DNAME"rips/2 &
waitfile "$DNAME"rips/sock.rips
if [ $BAD = true ]; then
	tail -5 "$DNAME"rips/2  1>&2
	echo rips did not work 1>&2
	exit 2
fi
nc -UN  "$DNAME"rips/sock.rips < ./extern/examples/msg1  > "$DNAME"rips/nc

./rips/rips -c   ./extern/examples/scripts $RULEFILE > ./gen/gen.go
(cd $ROOT/gen; go build)
mkdir -p  "$DNAME"gen
(./gen/gen -s "$DNAME"gen/sock.rips  -r $ROOT/xrips/examples $ROOT/extern/examples/scripts | filterdiffs)  2>&1 > "$DNAME"gen/1 | filterdiffs > "$DNAME"gen/2 &
waitfile "$DNAME"gen/sock.rips
if [ $BAD = true ]; then
ls -l "$DNAME"gen
	tail -5  "$DNAME"gen/2 1>&2
	echo gen did not work 1>&2
	exit 2
fi
nc -UN "$DNAME"gen/sock.rips < ./extern/examples/msg1 > "$DNAME"gen/nc
wait
if ! 9 diff -n "$DNAME"gen "$DNAME"rips >/tmp/out.$$; then
	tail -5 /tmp/out.$$ 2>&1
	echo  /tmp/out.$$ 2>&1
	echo FAIL 2>&1;
	exit 2
fi
rm  /tmp/out.$$
rm -rf "$DNAME"rips
rm -rf "$DNAME"gen
echo ok ab_test.sh
exit 0
 