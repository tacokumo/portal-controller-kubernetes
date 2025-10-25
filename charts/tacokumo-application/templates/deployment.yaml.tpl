apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Values.main.applicationName }}
  annotations:
    {{- with .Values.main.annotations }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
spec:
  replicas: {{ .Values.main.replicaCount }}
  selector:
    matchLabels:
      application: {{ .Values.main.applicationName }}
  template:
    metadata:
      labels:
        application: {{ .Values.main.applicationName }}
      annotations:
        {{- with .Values.main.podAnnotations }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
    spec:
      {{- with .Values.main.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      containers:
        - name: {{ .Values.main.applicationName }}
          image: "{{ .Values.main.image }}"
          imagePullPolicy: {{ .Values.main.imagePullPolicy }}