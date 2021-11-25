FROM bitnami/kafka
RUN mkdir -p /opt/bitnami/kafka/plugins
COPY mongo-kafka-connect-1.6.1-all.jar /opt/bitnami/kafka/plugins
COPY init.sh /opt/bitnami/kafka/bin
CMD /opt/bitnami/kafka/bin/init.sh