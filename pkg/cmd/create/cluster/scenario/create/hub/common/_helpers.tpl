{{- define "ocpImage" }}
  {{ $release := splitList ":" .managedCluster.ocpImage }}
  {{ if gt (len $release) 1 }}
    {{ $release = index $release 1 | replace "_" "-" | lower }}
    {{ $release = (print $release "-" .managedCluster.name ) }}
{{ $release }}
  {{ end }}
{{- end }}