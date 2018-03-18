#!/bin/sh

# Package
PACKAGE="garagedoor"
DNAME="GDHK"

# Others
INSTALL_DIR="/usr/local/${PACKAGE}"
INSTALLER_SCRIPT=`dirname $0`/installer
PATH="${PATH}:${INSTALL_DIR}/bin:${PATH}"
DAEMON="${INSTALL_DIR}/bin/gdhk"
PID_FILE="${INSTALL_DIR}/var/gdhk.pid"
LOG_FILE="${INSTALL_DIR}/var/log/gdhk"

start_daemon ()
{
    ${DAEMON} 2>&1 >> ${LOG_FILE} &
    echo $! > ${PID_FILE}
}

stop_daemon ()
{
    if daemon_status; then
        echo Stopping ${DNAME} ...
        kill `cat ${PID_FILE}`
        wait_for_status 1 20 || kill -9 `cat ${PID_FILE}`
    else
        echo ${DNAME} is not running
        exit 0
    fi

    test -e ${PID_FILE} || rm -f ${PID_FILE}
}

daemon_status ()
{
    if [ -f ${PID_FILE} ] && kill -0 `cat ${PID_FILE}` > /dev/null 2>&1; then
        return
    fi
    rm -f ${PID_FILE}
    return 1
}

wait_for_status ()
{
    counter=$2
    while [ ${counter} -gt 0 ]; do
        daemon_status
        [ $? -eq $1 ] && return
        let counter=counter-1
        sleep 1
    done
    return 1
}

case $1 in
    start)
        if daemon_status; then
            echo ${DNAME} is already running
            exit 0
        else
            echo Starting ${DNAME} ...
            start_daemon
            exit $?
        fi
        ;;
    stop)
            stop_daemon
            exit $?
        ;;
    restart)
        stop_daemon
        start_daemon
        exit $?
        ;;
    status)
        if daemon_status; then
            echo ${DNAME} is running
            exit 0
        else
            echo ${DNAME} is not running
            exit 1
        fi
        ;;
    log)
        echo ${LOG_FILE}
        ;;
    *)
        exit 1
        ;;
esac
