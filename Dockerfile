FROM amazonlinux
COPY ./adm-controller /adm-controller
ENTRYPOINT ["/adm-controller"]
