#!/bin/bash
jmeter -n -t $JMETER_HOME/jmeter.jmx -l /output/jmeter-log.jtl -JHOST=$JHOST -JPORT=$JPORT -JTHREADS=$JTHREADS -JTHROUGHPUT=$JTHROUGHPUT
