    #!/bin/bash
    cp /properties/* /opt/bitnami/kafka/config
    /opt/bitnami/kafka/bin/connect-standalone.sh /opt/bitnami/kafka/config/connect-standalone.properties /opt/bitnami/kafka/config/mongo.sink.properties /opt/bitnami/kafka/config/mongo.source.properties 
