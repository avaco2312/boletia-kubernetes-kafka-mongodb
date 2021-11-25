cd data
del /q *.*
rmdir /q /s .
cd ..
kind create cluster --config=config.yaml
kind load docker-image bitnami/kafka bitnami/zookeeper
kubectl apply -f kafka.yaml
kind load docker-image mongo
kubectl apply -f mongo.yaml
kind load docker-image kafka-mongodb-connect
kubectl apply -f kafka-mongodb-connect.yaml
kind load docker-image boletia/eventos boletia/reservas boletia/inventario boletia/notificaciones
kubectl apply -f clientes.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait --namespace ingress-nginx ^
  --for=condition=ready pod ^
  --selector=app.kubernetes.io/component=controller ^
  --timeout=180s
kubectl apply -f ingress.yaml
