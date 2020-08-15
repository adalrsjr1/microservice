# Dockerfile to build a jmeter container able to drive acmeair
# Results appear on /output in the container
# Must specify the hostname for the acmeair application (or localhost will be assumed)
FROM ibmjava:8-jre

ENV JMETER_VERSION 5.2.1
# Install pre-requisite packages
RUN apt-get update && apt-get install -y --no-install-recommends wget unzip curl inetutils-ping \
       && rm -rf /var/lib/apt/lists/*

# Install jmeter
RUN   mkdir /jmeter \
      && mkdir /output \
      && cd /jmeter/ \
      && wget https://archive.apache.org/dist/jmeter/binaries/apache-jmeter-$JMETER_VERSION.tgz \
      && tar -xzf apache-jmeter-$JMETER_VERSION.tgz \
      && rm apache-jmeter-$JMETER_VERSION.tgz

RUN apt-get remove -y wget unzip && apt-get autoremove --purge -y && apt-get autoclean -y

#COPY apache-jmeter-$JMETER_VERSION /jmeter/apache-jmeter-$JMETER_VERSION
# Set jmeter home, add jmeter to the PATH and set JVM options
ENV JMETER_HOME="/jmeter/apache-jmeter-$JMETER_VERSION"
ENV PATH="$JMETER_HOME/bin:$PATH"
ENV JVM_ARGS="-Xms1g -Xmx1g -Xtune:virtualized -XX:+UseContainerSupport -XX:MaxRAMPercentage=90"

WORKDIR $JMETER_HOME

# We should set summariser.interval=6 in bin/jmeter.properties
RUN echo 'summariser.interval=6' >> $JMETER_HOME/bin/jmeter.properties

# Copy the script to be executed and other needed files
COPY jmeter.jmx $JMETER_HOME/
COPY init.sh $JMETER_HOME/bin/init.sh

RUN chmod u+x $JMETER_HOME/bin/init.sh

ENV JHOST=localhost \
    JPORT=8080 \
    JTHREADS=200 \
# Throughput per minute
    JTHROUGHPUT=4000

ENTRYPOINT ["/bin/bash", "-c", "init.sh"]
