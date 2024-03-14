#!/bin/sh

usage(){
	echo "usage: bench.sh [-r] [-g] [-s] id file.rul" 1>&2;
	exit 1
}

export RUNGEN=false
case $1 in
-r)
	RUNGEN=true
	shift
	;;
esac


export ISGEN=false
case $1 in
-g)
	ISGEN=true
	shift
	;;
esac

export ISSTATS=false
case $1 in
-s)
	ISSTATS=true
	shift
	;;
esac


case $# in
2)
	TID=$1
	shift
	RULEFILE=$1
	shift
	;;
*)
	usage
esac


export BAD=false
waitfile() {
	for i in `seq 1 100`; do	
		if test -S $1; then
			return
			break
		fi
	 	sleep 0.001
	done
	export BAD=true
}


export TSTNAME=`basename $RULEFILE|sed 's/\.rul$//g'`

export ROOT=`git rev-parse --show-toplevel`

if [ "$RUNGEN" = false ]; then
	(cd $ROOT/rips; go build)
fi
cd $ROOT

export DNAME=/tmp/$TSTNAME$TID

rm -rf "$DNAME"
mkdir -p "$DNAME"

echo  "$DNAME"

case $ISGEN in
false)
	(./rips/rips -s "$DNAME"/sock.rips  -r $ROOT/xrips/examples $ROOT/extern/examples/scripts  $ROOT/$RULEFILE) 2>"$DNAME"/2 > "$DNAME"/1  &
	waitfile "$DNAME"/sock.rips
	if [ $BAD = true ]; then
		tail -5 "$DNAME"/2  1>&2
		echo rips did not work 1>&2
		exit 2
	fi
	;;
true)
	if [ "$RUNGEN" = false ]; then
		mkdir -p ./gen
		./rips/rips -c   ./extern/examples/scripts $RULEFILE  > ./gen/gen.go 2> "$DNAME"/2
		if [ "$ISSTATS" = true ]; then
				if ! egrep -n '^Stats'  "$DNAME"/2  "$DNAME"/2 | tail -1 1>&2; then
				echo no stats enabled 1>&2
				exit 1
			fi
		fi
		(cd $ROOT/gen; go build)
		mkdir -p  "$DNAME"
		exit 0
	fi
	(./gen/gen -s "$DNAME"/sock.rips  -r $ROOT/xrips/examples $ROOT/extern/examples/scripts )  2>"$DNAME"/2 > "$DNAME"/1  &
	waitfile "$DNAME"/sock.rips
	if [ $BAD = true ]; then
	ls -l "$DNAME"
		tail -5  "$DNAME"/2 1>&2
		echo gen did not work 1>&2
		exit 2
	fi
	if [ "$RUNGEN" = true ]; then
		rm -rf ./gen
	fi
	;;
*)
	usage
esac
#nc -UN "$DNAME"/sock.rips < ./extern/examples/msgN > "$DNAME"/nc
#wait
#rm -rf "$DNAME"
echo ok bench.sh
exit 0
 
