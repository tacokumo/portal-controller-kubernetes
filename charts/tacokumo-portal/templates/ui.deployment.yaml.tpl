apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Values.namePrefix }}-portal-ui
  namespace: {{ .Values.namespace }}
  annotations:
    {{- with .Values.ui.annotations }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
spec:
  replicas: {{ .Values.ui.replicaCount }}
  selector:
    matchLabels:
      application: {{ .Values.namePrefix }}-portal-ui
  template:
    metadata:
      labels:
        application: {{ .Values.namePrefix }}-portal-ui
      annotations:
        {{- with .Values.ui.podAnnotations }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
    spec:
      {{- with .Values.ui.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      containers:
        - name: {{ .Values.namePrefix }}-portal-ui
          image: "{{ .Values.ui.image }}"
          imagePullPolicy: {{ .Values.ui.imagePullPolicy }}