FROM {{.BaseImage}}

# RUN suseconnect -r {{.RegCode}}

CMD [ "sleep", "60m" ]