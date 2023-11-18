#run.sh <test> <basedir> <machine>

export TESTDIR=test/$1/
cd $2
source ${TESTDIR}common.sh
echo Starting $3
${TESTDIR}${3}.sh 2>&1 | tee ${TESTDIR}/logs/$3.log | awk -v NAME="[$3]" '{print NAME $0}'
echo Stopped $3