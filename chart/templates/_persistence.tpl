{{/*
  ControlPlane Kind
*/}}
{{- define "vcluster.kind" -}}
{{ if include "vcluster.persistence.volumeClaim.enabled" . }}StatefulSet{{ else }}Deployment{{ end }}
{{- end -}}

{{/*
  StatefulSet Persistence Options
*/}}
{{- define "vcluster.persistence" -}}
{{- if .Values.controlPlane.statefulSet.persistence.volumeClaimTemplates }}
volumeClaimTemplates:
{{ toYaml .Values.controlPlane.statefulSet.persistence.volumeClaimTemplates | indent 2 }}
{{- else if include "vcluster.persistence.volumeClaim.enabled" . }}
volumeClaimTemplates:
- metadata:
    name: data
 {{- $annotations := merge dict .Values.controlPlane.statefulSet.annotations .Values.controlPlane.advanced.globalMetadata.annotations }}
{{- if $annotations }}
    annotations:
{{ toYaml $annotations | indent 6 }}
{{- end }}
  spec:
    accessModes: {{ .Values.controlPlane.statefulSet.persistence.volumeClaim.accessModes }}
    {{- if .Values.controlPlane.statefulSet.persistence.volumeClaim.storageClass }}
    storageClassName: {{ .Values.controlPlane.statefulSet.persistence.volumeClaim.storageClass }}
    {{- end }}
    resources:
      requests:
        storage: {{ .Values.controlPlane.statefulSet.persistence.volumeClaim.size }}
{{- end }}
{{- end -}}

{{/*
  is persistence enabled?
*/}}
{{- define "vcluster.persistence.volumeClaim.enabled" -}}
{{- if .Values.controlPlane.statefulSet.persistence.volumeClaimTemplates -}}
{{- true -}}
{{- else if eq (toString .Values.controlPlane.statefulSet.persistence.volumeClaim.enabled) "true" -}}
{{- true -}}
{{- else if and (eq (toString .Values.controlPlane.statefulSet.persistence.volumeClaim.enabled) "auto") (or (include "vcluster.database.embedded.enabled" .) .Values.controlPlane.backingStore.etcd.embedded.enabled) -}}
{{- true -}}
{{- end -}}
{{- end -}}
