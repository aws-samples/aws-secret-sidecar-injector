{{/*
Expand the name of the chart.
*/}}
{{- define "secret-inject.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}


{{/*
Generate certificates for AWS Secrets Controller webhook
*/}}
{{- define "secret-inject.gen-certs" -}}
{{- $altNames := "secret-inject.default.svc" -}}
{{- $ca := genCA "secret-inject-ca" 3650 -}}
{{- $cert := genSignedCert "secret-inject.default.svc" nil nil  3650 $ca -}}
caCert: {{ $ca.Cert | b64enc }}
clientCert: {{ $cert.Cert | b64enc }}
clientKey: {{ $cert.Key | b64enc }}
{{- end -}}

