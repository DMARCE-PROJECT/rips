#!/bin/sh

usage(){
	echo "usage: nc.sh [-s]  testpath" 1>&2;
	exit 1
}

export ISSTATS=false
case $1 in
-s)
	ISSTATS=true
	shift
	;;
esac
case $# in
1)
	SOCPATH=$1
	shift
	;;
*)
	usage
esac

export ROOT=`git rev-parse --show-toplevel`
cd $ROOT

echo $SOCPATH >> /tmp/yyy

nc -UN "$SOCPATH"/sock.rips < ./extern/examples/msgN > /dev/null
if [ "$ISSTATS" = true ]; then
	if ! egrep -n '^Stats|(divide by zero)'  "$SOCPATH"/2 /dev/null 1>&2; then
		echo no stats enabled 1>&2
		exit 1
	fi
fi
rm -rf "$SOCPATH"


