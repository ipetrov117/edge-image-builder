FROM {{.BaseImage}}

{{ if and (ne .FromRPMPath "") (ne .ToRPMPath "") -}}
COPY {{ .FromRPMPath }} {{ .ToRPMPath -}}
{{ end }}

{{ if ne .RegCode "" -}}
RUN suseconnect -r {{.RegCode}}
RUN SLE_SP=$(cat /etc/rpm/macros.sle | awk '/sle/ {print $2};' | cut -c4) && suseconnect -p PackageHub/15.$SLE_SP/x86_64
RUN zypper ref
{{ end }}

{{ if ne .AddRepo "" -}}
RUN counter=1 && \
    for i in {{.AddRepo}}; \
    do \
      zypper ar --no-gpgcheck -f $i addrepo$counter; \
      counter=$((counter+1)); \
    done
{{ end }}

RUN zypper \
    --pkg-cache-dir {{.CacheDir}} \ 
    --no-gpg-checks \
    install -y \
    --download-only \
    --force-resolution \
    --auto-agree-with-licenses \
    --allow-vendor-change \
    -n {{.PkgList}}

RUN touch {{.CacheDir}}/zypper-success

{{ if ne .RegCode "" -}}
RUN suseconnect -d
{{ end }}

CMD [ "sleep", "60m" ]