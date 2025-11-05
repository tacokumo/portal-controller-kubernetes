apiVersion: v1
kind: Service
metadata:
  name: {{ .Values.namePrefix }}-portal-ui
spec:
  type: ClusterIP
  selector:
    application: {{ .Values.namePrefix }}-portal-ui
  ports:
  - protocol: TCP
    port: {{ .Values.ui.service.port }}
    targetPort: {{ .Values.ui.service.targetPort }}