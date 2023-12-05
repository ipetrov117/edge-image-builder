FROM {{.BaseImage}}

RUN suseconnect -r {{.RegCode}}
RUN SLE_SP=$(cat /etc/rpm/macros.sle | awk '/sle/ {print $2};' | cut -c4) && suseconnect -p PackageHub/15.$SLE_SP/x86_64
RUN zypper ref

RUN counter=1 && \
    for i in "{{.AddRepo}}"; \
    do \
      zypper ar --no-gpgcheck -f $i addrepo$counter; \
      counter=$((counter+1)); \
    done

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

RUN suseconnect -d

CMD [ "sleep", "60m" ]